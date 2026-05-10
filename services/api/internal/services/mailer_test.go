package services

import (
	"bytes"
	"context"
	"io"
	"log"
	"os"
	"strings"
	"testing"
)

func TestNoOpMailer_DoesNotPrintEmailBody(t *testing.T) {
	originalStdout := os.Stdout
	readPipe, writePipe, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	os.Stdout = writePipe
	defer func() {
		os.Stdout = originalStdout
	}()

	var logOutput bytes.Buffer
	originalLogOutput := log.Writer()
	log.SetOutput(&logOutput)
	defer log.SetOutput(originalLogOutput)

	body := `<a href="https://app.example/reset?token=secret-token">reset</a>`
	if err := (&NoOpMailer{}).Send(context.Background(), "user@example.com", "Reset password", body); err != nil {
		t.Fatalf("send: %v", err)
	}

	if err := writePipe.Close(); err != nil {
		t.Fatalf("close stdout pipe: %v", err)
	}
	outputBytes, err := io.ReadAll(readPipe)
	if err != nil {
		t.Fatalf("read stdout pipe: %v", err)
	}

	combined := string(outputBytes) + logOutput.String()
	if strings.Contains(combined, "secret-token") || strings.Contains(combined, body) {
		t.Fatalf("no-op mailer leaked email body: %q", combined)
	}
}
