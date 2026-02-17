package email

import (
	"crypto/tls"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/smtp"
	"strings"
	"time"
)

// Sender is the interface for sending emails.
type Sender interface {
	Send(to, subject, htmlBody string) error
}

// SMTPSender sends emails via SMTP with STARTTLS.
type SMTPSender struct {
	Host     string
	Port     int
	Username string
	Password string
	From     string
	FromName string
}

func (s *SMTPSender) Send(to, subject, htmlBody string) error {
	addr := net.JoinHostPort(s.Host, fmt.Sprintf("%d", s.Port))

	from := s.From
	if s.FromName != "" {
		from = fmt.Sprintf("%s <%s>", s.FromName, s.From)
	}

	headers := map[string]string{
		"From":         from,
		"To":           to,
		"Subject":      subject,
		"MIME-Version": "1.0",
		"Content-Type": "text/html; charset=\"UTF-8\"",
		"Date":         time.Now().Format(time.RFC1123Z),
	}

	var msg strings.Builder
	for k, v := range headers {
		msg.WriteString(fmt.Sprintf("%s: %s\r\n", k, v))
	}
	msg.WriteString("\r\n")
	msg.WriteString(htmlBody)

	conn, err := net.DialTimeout("tcp", addr, 10*time.Second)
	if err != nil {
		return fmt.Errorf("smtp dial: %w", err)
	}

	client, err := smtp.NewClient(conn, s.Host)
	if err != nil {
		conn.Close()
		return fmt.Errorf("smtp client: %w", err)
	}
	defer client.Close()

	tlsConfig := &tls.Config{ServerName: s.Host}
	if ok, _ := client.Extension("STARTTLS"); ok {
		if err := client.StartTLS(tlsConfig); err != nil {
			return fmt.Errorf("smtp starttls: %w", err)
		}
	}

	if s.Username != "" {
		auth := smtp.PlainAuth("", s.Username, s.Password, s.Host)
		if err := client.Auth(auth); err != nil {
			return fmt.Errorf("smtp auth: %w", err)
		}
	}

	if err := client.Mail(s.From); err != nil {
		return fmt.Errorf("smtp mail from: %w", err)
	}
	if err := client.Rcpt(to); err != nil {
		return fmt.Errorf("smtp rcpt to: %w", err)
	}

	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("smtp data: %w", err)
	}
	if _, err := w.Write([]byte(msg.String())); err != nil {
		return fmt.Errorf("smtp write: %w", err)
	}
	if err := w.Close(); err != nil {
		return fmt.Errorf("smtp close data: %w", err)
	}

	return client.Quit()
}

// SendGridSender sends emails via the SendGrid v3 HTTP API.
type SendGridSender struct {
	APIKey   string
	From     string
	FromName string
}

func (s *SendGridSender) Send(to, subject, htmlBody string) error {
	payload := fmt.Sprintf(`{
		"personalizations": [{"to": [{"email": %q}]}],
		"from": {"email": %q, "name": %q},
		"subject": %q,
		"content": [{"type": "text/html", "value": %q}]
	}`, to, s.From, s.FromName, subject, htmlBody)

	req, err := http.NewRequest("POST", "https://api.sendgrid.com/v3/mail/send", strings.NewReader(payload))
	if err != nil {
		return fmt.Errorf("sendgrid request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+s.APIKey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("sendgrid send: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("sendgrid error status=%d body=%s", resp.StatusCode, string(body))
	}

	return nil
}

// NoopSender logs emails but does not actually send them. Useful for development.
type NoopSender struct{}

func (n *NoopSender) Send(to, subject, htmlBody string) error {
	log.Printf("[noop-email] to=%s subject=%q bodyLen=%d", to, subject, len(htmlBody))
	return nil
}
