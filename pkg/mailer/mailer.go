package mailer

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/wneessen/go-mail"
	"go.uber.org/zap"
)

type Notifier interface {
	NotifyResetPassword(code, email string) error
	NotifyEmailConfirmation(code, email string) error
	Close(ctx context.Context) error
}

type NoopNotifier struct{}

func (n *NoopNotifier) NotifyResetPassword(_, _ string) error { return nil }

func (n *NoopNotifier) NotifyEmailConfirmation(_, _ string) error { return nil }

func (n *NoopNotifier) Close(_ context.Context) error { return nil }

type job struct {
	subject,
	body,
	to string
}

const (
	queueSize   = 100
	numWorkers  = 2
	sendTimeout = 15 * time.Second
)

type SMTPMailer struct {
	smtpClient *mail.Client
	from,
	ownerEmail string
	queue  chan job
	wg     sync.WaitGroup
	logger *zap.Logger
}

func NewSMTPMailer(host, username, password, from string, port int, logger *zap.Logger) (*SMTPMailer, error) {
	if from == "" {
		from = username
	}

	opts := []mail.Option{
		mail.WithPort(port),
		mail.WithSMTPAuth(mail.SMTPAuthPlain),
		mail.WithUsername(username),
		mail.WithPassword(password),
		mail.WithTimeout(10 * time.Second),
	}
	if port == 465 {
		opts = append(opts, mail.WithSSL())
	} else {
		opts = append(opts, mail.WithTLSPolicy(mail.TLSMandatory))
	}

	client, err := mail.NewClient(host, opts...)
	if err != nil {
		return nil, fmt.Errorf("mailer: NewSMTPClient: failed to create new smtp client: %w", err)
	}

	m := &SMTPMailer{
		smtpClient: client,
		from:       from,
		queue:      make(chan job, queueSize),
		logger:     logger,
	}

	for range numWorkers {
		m.wg.Add(1)
		go m.worker()
	}

	return m, nil
}

func (m *SMTPMailer) worker() {
	defer m.wg.Done()
	for j := range m.queue {
		ctx, cancel := context.WithTimeout(context.Background(), sendTimeout)
		if err := m.sendToUser(ctx, j.subject, j.body, j.to); err != nil {
			m.logger.Error("mailer: worker: failed to send email", zap.Error(err))
		}
		cancel()
	}
}

func (m *SMTPMailer) enqueue(j job) {
	select {
	case m.queue <- j:
	default:
		m.logger.Error("mailer: queue full, notification dropped", zap.String("subject", j.subject))
	}
}

func (m *SMTPMailer) Close(ctx context.Context) error {
	close(m.queue)
	done := make(chan struct{})
	go func() {
		m.wg.Wait()
		close(done)
	}()
	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return fmt.Errorf("mailer.Close: drain timed out: %w", ctx.Err())
	}
}

func (m *SMTPMailer) sendToUser(ctx context.Context, subject, body, to string) error {
	msg := mail.NewMsg()
	if err := msg.To(to); err != nil {
		return fmt.Errorf("mailer.sendToUser: failed set to user send user email: %w", err)
	}
	if err := msg.From(m.from); err != nil {
		return fmt.Errorf("mailer.sendToUser: failed set from owner email: %w", err)
	}
	msg.Subject(subject)
	msg.SetBodyString(mail.TypeTextHTML, body)
	if err := m.smtpClient.DialAndSendWithContext(ctx, msg); err != nil {
		return fmt.Errorf("mailer.DialAndSendWithContext: failed to send user email with 6-signs password for resetting it password: %w", err)
	}
	return nil
}

func (m *SMTPMailer) NotifyResetPassword(code, email string) error {
	body, err := Render(ResetPassword, struct{ Code string }{Code: code})
	if err != nil {
		return fmt.Errorf("mailer.NotifyResetPassword: failed to render reset password: %w", err)
	}

	m.enqueue(job{
		subject: "ServiceHub | Сброс пароля",
		body:    body,
		to:      email,
	})

	return nil
}

func (m *SMTPMailer) NotifyEmailConfirmation(code, email string) error {
	body, err := Render(EmailConfirmation, struct{ Code string }{Code: code})
	if err != nil {
		return fmt.Errorf("mailer.NotifyEmailConfirmation: failed to render email confirmation: %w", err)
	}

	m.enqueue(job{
		subject: "ServiceHub | Подтверждение электронного адреса",
		body:    body,
		to:      email,
	})

	return nil
}
