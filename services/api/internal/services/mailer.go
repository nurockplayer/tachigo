package services

import (
	"context"
	"crypto/tls"
	"fmt"
	"log"
	"net"
	"net/mail"
	"net/smtp"

	"gopkg.in/gomail.v2"
)

// Mailer is the interface for sending emails.
// It is abstracted so tests can swap in a mock without hitting real SMTP.
type Mailer interface {
	Send(ctx context.Context, to, subject, htmlBody string) error
}

// SMTPMailer sends email via SMTP using gomail.
type SMTPMailer struct {
	host     string
	port     int
	username string
	password string
	from     string
}

func NewSMTPMailer(host string, port int, username, password, from string) *SMTPMailer {
	return &SMTPMailer{
		host:     host,
		port:     port,
		username: username,
		password: password,
		from:     from,
	}
}

func (m *SMTPMailer) Send(ctx context.Context, to, subject, htmlBody string) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	msg := gomail.NewMessage()
	msg.SetHeader("From", m.from)
	msg.SetHeader("To", to)
	msg.SetHeader("Subject", subject)
	msg.SetBody("text/html", htmlBody)

	tlsCfg := &tls.Config{ServerName: m.host}
	addr := fmt.Sprintf("%s:%d", m.host, m.port)

	// DialContext lets the TCP dial itself be cancelled or time-out via ctx.
	var nd net.Dialer
	conn, err := nd.DialContext(ctx, "tcp", addr)
	if err != nil {
		return err
	}
	// Propagate ctx deadline to all subsequent SMTP I/O (STARTTLS, AUTH, DATA).
	if dl, ok := ctx.Deadline(); ok {
		_ = conn.SetDeadline(dl)
	}
	// For cancel-only contexts (no deadline), close conn when ctx is done so
	// blocked SMTP I/O (STARTTLS, AUTH, DATA) unblocks immediately.
	watchDone := make(chan struct{})
	go func() {
		select {
		case <-ctx.Done():
			_ = conn.Close()
		case <-watchDone:
		}
	}()
	defer close(watchDone)

	var client *smtp.Client
	if m.port == 465 {
		// Port 465: implicit TLS (SMTPS) — wrap conn before SMTP handshake.
		client, err = smtp.NewClient(tls.Client(conn, tlsCfg), m.host)
	} else {
		// Other ports (587, 25): upgrade via STARTTLS only if server advertises it.
		client, err = smtp.NewClient(conn, m.host)
		if err == nil {
			if ok, _ := client.Extension("STARTTLS"); ok {
				if startErr := client.StartTLS(tlsCfg); startErr != nil {
					client.Close()
					err = startErr
				}
			}
		}
	}
	if err != nil {
		conn.Close()
		return err
	}
	defer client.Close()

	if m.username != "" {
		if err := client.Auth(smtp.PlainAuth("", m.username, m.password, m.host)); err != nil {
			return err
		}
	}
	// m.from may carry a display name ("Name <addr>"); MAIL FROM requires a bare address.
	fromAddr, parseErr := mail.ParseAddress(m.from)
	if parseErr != nil {
		return fmt.Errorf("mailer: invalid from address %q: %w", m.from, parseErr)
	}
	if err := client.Mail(fromAddr.Address); err != nil {
		return err
	}
	if err := client.Rcpt(to); err != nil {
		return err
	}
	w, err := client.Data()
	if err != nil {
		return err
	}
	if _, err := msg.WriteTo(w); err != nil {
		return err
	}
	if err := w.Close(); err != nil {
		return err
	}
	return client.Quit()
}

// NoOpMailer is used when SMTP is not configured.
// It logs delivery metadata without the body so token-bearing links never reach logs.
type NoOpMailer struct{}

func (n *NoOpMailer) Send(_ context.Context, to, subject, htmlBody string) error {
	log.Printf("[mailer] no-op send | to=%s | subject=%s | body_bytes=%d | body=redacted", to, subject, len(htmlBody))
	return nil
}

// NewMailer returns an SMTPMailer if a host is configured, otherwise a NoOpMailer.
func NewMailer(host string, port int, username, password, from string) Mailer {
	if host == "" {
		return &NoOpMailer{}
	}
	return NewSMTPMailer(host, port, username, password, from)
}
