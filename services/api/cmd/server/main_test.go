package main

import (
	"context"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/tachigo/tachigo/internal/config"
	"github.com/tachigo/tachigo/internal/services"
)

func TestServerStartupDoesNotRunSchemaDDL(t *testing.T) {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve current test file")
	}

	dir := filepath.Dir(file)
	var source strings.Builder
	for _, name := range []string{"main.go", "bootstrap.go", "wiring.go"} {
		body, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			t.Fatalf("read %s: %v", name, err)
		}
		source.Write(body)
	}
	for _, forbidden := range []string{
		"AutoMigrate(",
		"initializeUserRoleEnum(",
		"ensureCouponRedemptionRuntimeSchema(",
		"CREATE UNIQUE INDEX IF NOT EXISTS",
		"ALTER TABLE tachi_balances ADD CONSTRAINT",
		"applyStreamerAgencyMigration(db)",
	} {
		if strings.Contains(source.String(), forbidden) {
			t.Fatalf("server startup must not run schema DDL %q after Atlas owns migrations", forbidden)
		}
	}
	bootstrapBody, err := os.ReadFile(filepath.Join(dir, "bootstrap.go"))
	if err != nil {
		t.Fatalf("read bootstrap.go: %v", err)
	}
	bootstrapSource := string(bootstrapBody)
	if !strings.Contains(bootstrapSource, "hashLegacyRaffleClaimTokens(hashCtx, db)") {
		t.Fatalf("bootstrap should keep the non-schema raffle claim token data repair")
	}
	if !strings.Contains(bootstrapSource, "db.WithContext(ctx).Exec") {
		t.Fatalf("legacy raffle claim token repair must respect startup timeout context")
	}
}

func TestServerStartupIsSplitByResponsibility(t *testing.T) {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve current test file")
	}
	dir := filepath.Dir(file)

	mainBody, err := os.ReadFile(filepath.Join(dir, "main.go"))
	if err != nil {
		t.Fatalf("read main.go: %v", err)
	}
	if lines := strings.Count(string(mainBody), "\n"); lines > 50 {
		t.Fatalf("main.go should stay at 50 lines or fewer after bootstrap/wiring split, got %d", lines)
	}
	for _, want := range []string{
		"bootstrap(cfg)",
		"wire(db, cfg, serverCtx)",
	} {
		if !strings.Contains(string(mainBody), want) {
			t.Fatalf("main.go should delegate startup with %q", want)
		}
	}

	bootstrapBody, err := os.ReadFile(filepath.Join(dir, "bootstrap.go"))
	if err != nil {
		t.Fatalf("read bootstrap.go: %v", err)
	}
	bootstrapSource := string(bootstrapBody)
	for _, want := range []string{
		"func bootstrap(cfg *config.Config) *gorm.DB",
		"database.Connect(cfg.Database.DSN)",
		"hashLegacyRaffleClaimTokens(hashCtx, db)",
	} {
		if !strings.Contains(bootstrapSource, want) {
			t.Fatalf("bootstrap.go should own %q", want)
		}
	}

	wiringBody, err := os.ReadFile(filepath.Join(dir, "wiring.go"))
	if err != nil {
		t.Fatalf("read wiring.go: %v", err)
	}
	wiringSource := string(wiringBody)
	for _, want := range []string{
		"func wire(db *gorm.DB, cfg *config.Config, ctx context.Context) *gin.Engine",
		"services.NewAuthService(db, cfg)",
		"context.WithTimeout(ctx, 10*time.Second)",
		"startRaffleSchedulerIfEnabled(ctx, cfg, raffleSvc)",
		"router.New(",
	} {
		if !strings.Contains(wiringSource, want) {
			t.Fatalf("wiring.go should own %q", want)
		}
	}
	if strings.Contains(wiringSource, "context.WithTimeout(context.Background(), 10*time.Second)") {
		t.Fatalf("Sepolia RPC dial timeout should inherit the server context")
	}
}

func TestStartRaffleSchedulerIfEnabledRespectsConfig(t *testing.T) {
	originalFactory := raffleSchedulerFactory
	defer func() { raffleSchedulerFactory = originalFactory }()

	var starts atomic.Int32
	raffleSchedulerFactory = func(_ *services.RaffleService) raffleSchedulerStarter {
		return fakeRaffleSchedulerStarter{starts: &starts}
	}

	startRaffleSchedulerIfEnabled(context.Background(), &config.Config{
		Server: config.ServerConfig{EnableScheduler: false},
	}, nil)
	if got := starts.Load(); got != 0 {
		t.Fatalf("disabled scheduler starts: want 0, got %d", got)
	}

	startRaffleSchedulerIfEnabled(context.Background(), &config.Config{
		Server: config.ServerConfig{EnableScheduler: true},
	}, nil)
	if got := starts.Load(); got != 1 {
		t.Fatalf("enabled scheduler starts: want 1, got %d", got)
	}
}

func TestStartRaffleSchedulerIfEnabledDefaultsToEnabledForNilConfig(t *testing.T) {
	originalFactory := raffleSchedulerFactory
	defer func() { raffleSchedulerFactory = originalFactory }()

	var starts atomic.Int32
	raffleSchedulerFactory = func(_ *services.RaffleService) raffleSchedulerStarter {
		return fakeRaffleSchedulerStarter{starts: &starts}
	}

	startRaffleSchedulerIfEnabled(context.Background(), nil, nil)
	if got := starts.Load(); got != 1 {
		t.Fatalf("nil config scheduler starts: want 1, got %d", got)
	}
}

type fakeRaffleSchedulerStarter struct {
	starts *atomic.Int32
}

func (f fakeRaffleSchedulerStarter) Start(context.Context) {
	f.starts.Add(1)
}

func TestHTTPServerConfiguresProductionTimeouts(t *testing.T) {
	handler := http.NewServeMux()

	srv := newHTTPServer(":8080", handler)

	if srv.Addr != ":8080" {
		t.Fatalf("Addr: want :8080, got %q", srv.Addr)
	}
	if srv.Handler != handler {
		t.Fatalf("Handler: want configured handler")
	}
	if srv.ReadTimeout <= 0 {
		t.Fatalf("ReadTimeout should be set")
	}
	if srv.WriteTimeout <= 0 {
		t.Fatalf("WriteTimeout should be set")
	}
	if srv.IdleTimeout <= 0 {
		t.Fatalf("IdleTimeout should be set")
	}
}

func TestRunHTTPServerShutsDownOnContextCancelAndClosesDatabase(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	fake := newFakeGracefulServer()
	var closeCalls atomic.Int32

	errCh := make(chan error, 1)
	go func() {
		errCh <- runHTTPServer(ctx, fake, func() error {
			closeCalls.Add(1)
			return nil
		})
	}()

	<-fake.listenStarted
	cancel()
	<-fake.shutdownCalled
	fake.finish(http.ErrServerClosed)

	select {
	case err := <-errCh:
		if err != nil {
			t.Fatalf("runHTTPServer returned error: %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("runHTTPServer did not return after shutdown")
	}
	if closeCalls.Load() != 1 {
		t.Fatalf("close hook calls: want 1, got %d", closeCalls.Load())
	}
}

func TestRunHTTPServerReturnsListenErrorAndClosesDatabase(t *testing.T) {
	ctx := context.Background()
	fake := newFakeGracefulServer()
	listenErr := errors.New("bind failed")
	var closeCalls atomic.Int32

	errCh := make(chan error, 1)
	go func() {
		errCh <- runHTTPServer(ctx, fake, func() error {
			closeCalls.Add(1)
			return nil
		})
	}()

	<-fake.listenStarted
	fake.finish(listenErr)

	select {
	case err := <-errCh:
		if !errors.Is(err, listenErr) {
			t.Fatalf("runHTTPServer error: want %v, got %v", listenErr, err)
		}
	case <-time.After(time.Second):
		t.Fatal("runHTTPServer did not return after listen error")
	}
	if closeCalls.Load() != 1 {
		t.Fatalf("close hook calls: want 1, got %d", closeCalls.Load())
	}
}

func TestRunHTTPServerReturnsShutdownErrorAndClosesDatabase(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	shutdownErr := errors.New("shutdown failed")
	fake := newFakeGracefulServer()
	fake.shutdownErr = shutdownErr
	var closeCalls atomic.Int32

	errCh := make(chan error, 1)
	go func() {
		errCh <- runHTTPServer(ctx, fake, func() error {
			closeCalls.Add(1)
			return nil
		})
	}()

	<-fake.listenStarted
	cancel()
	<-fake.shutdownCalled
	fake.finish(http.ErrServerClosed)

	select {
	case err := <-errCh:
		if !errors.Is(err, shutdownErr) {
			t.Fatalf("runHTTPServer error: want shutdown error %v, got %v", shutdownErr, err)
		}
	case <-time.After(time.Second):
		t.Fatal("runHTTPServer did not return after shutdown error")
	}
	if closeCalls.Load() != 1 {
		t.Fatalf("close hook calls: want 1, got %d", closeCalls.Load())
	}
}

func TestMainUsesHTTPServerGracefulShutdown(t *testing.T) {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve current test file")
	}
	dir := filepath.Dir(file)

	mainBody, err := os.ReadFile(filepath.Join(dir, "main.go"))
	if err != nil {
		t.Fatalf("read main.go: %v", err)
	}
	mainSource := string(mainBody)
	if strings.Contains(mainSource, ".Run(") {
		t.Fatalf("main.go should not start Gin with Run after graceful shutdown support is added")
	}
	for _, want := range []string{
		"newHTTPServer(addr, r)",
		"runHTTPServer(serverCtx, srv,",
		"closeServerResources(db, tracingShutdown)",
	} {
		if !strings.Contains(mainSource, want) {
			t.Fatalf("main.go should use graceful HTTP server wiring %q", want)
		}
	}
}

type fakeGracefulServer struct {
	listenStarted  chan struct{}
	shutdownCalled chan struct{}
	done           chan error
	shutdownErr    error
	closeStarted   sync.Once
	closeShutdown  sync.Once
}

func newFakeGracefulServer() *fakeGracefulServer {
	return &fakeGracefulServer{
		listenStarted:  make(chan struct{}),
		shutdownCalled: make(chan struct{}),
		done:           make(chan error, 1),
	}
}

func (f *fakeGracefulServer) ListenAndServe() error {
	f.closeStarted.Do(func() { close(f.listenStarted) })
	return <-f.done
}

func (f *fakeGracefulServer) Shutdown(context.Context) error {
	f.closeShutdown.Do(func() { close(f.shutdownCalled) })
	return f.shutdownErr
}

func (f *fakeGracefulServer) finish(err error) {
	f.done <- err
}
