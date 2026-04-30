package services

import (
	"context"
	"fmt"
	"log"

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
	msg := gomail.NewMessage()
	msg.SetHeader("From", m.from)
	msg.SetHeader("To", to)
	msg.SetHeader("Subject", subject)
	msg.SetBody("text/html", htmlBody)

	type result struct{ err error }
	ch := make(chan result, 1)
	go func() {
		d := gomail.NewDialer(m.host, m.port, m.username, m.password)
		ch <- result{d.DialAndSend(msg)}
	}()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case r := <-ch:
		return r.err
	}
}

// NoOpMailer is used when SMTP is not configured.
// It logs the email to stdout instead of sending it, which is useful during development.
type NoOpMailer struct{}

func (n *NoOpMailer) Send(_ context.Context, to, subject, htmlBody string) error {
	log.Printf("[mailer] no-op send | to=%s | subject=%s", to, subject)
	fmt.Printf("--- email ---\nTo: %s\nSubject: %s\n%s\n-------------\n", to, subject, htmlBody)
	return nil
}

// NewMailer returns an SMTPMailer if a host is configured, otherwise a NoOpMailer.
func NewMailer(host string, port int, username, password, from string) Mailer {
	if host == "" {
		return &NoOpMailer{}
	}
	return NewSMTPMailer(host, port, username, password, from)
}
