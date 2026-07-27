package mailer

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/wneessen/go-mail"
	"go.uber.org/zap"
)

type codeTemplate struct {
	FooterMessage,
	Code string
}

type anyTemplate struct {
	FooterMessage,
	Title,
	Body string
}

type balanceUpdateTemplate struct {
	FooterMessage,
	Username string
	Amount,
	NewBalance float64
}

type job struct {
	subject,
	body,
	to string
}

const (
	queueSize   = 100
	numWorkers  = 2
	sendTimeout = 15 * time.Second

	codeTemplateFooter           = "Если вы не запрашивали никаких кодов,\nпросто проигнорируйте это письмо."
	happyDayTemplateFooter       = "Хорошего Вам дня \U0001F604"
	balanceUpdatedTemplateFooter = "\U0001F4B0 Мы пересчитали — всё на месте. Теперь очередь за приятными тратами!"
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

func (m *SMTPMailer) enqueue(ctx context.Context, j job) {
	select {
	case <-ctx.Done():
		m.logger.Warn("mailer: worker: context canceled")
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

func (m *SMTPMailer) NotifyResetPassword(ctx context.Context, code, email string) error {
	body, err := Render(ResetPassword, codeTemplate{Code: code, FooterMessage: codeTemplateFooter})
	if err != nil {
		return fmt.Errorf("mailer.NotifyResetPassword: failed to render reset password: %w", err)
	}

	m.enqueue(ctx, job{
		subject: "ServiceHub | Сброс пароля",
		body:    body,
		to:      email,
	})

	return nil
}

func (m *SMTPMailer) NotifyEmailConfirmation(ctx context.Context, code, email string) error {
	body, err := Render(EmailConfirmation, codeTemplate{Code: code, FooterMessage: codeTemplateFooter})
	if err != nil {
		return fmt.Errorf("mailer.NotifyEmailConfirmation: failed to render email confirmation: %w", err)
	}

	m.enqueue(ctx, job{
		subject: "ServiceHub | Подтверждение электронного адреса",
		body:    body,
		to:      email,
	})

	return nil
}

func (m *SMTPMailer) NotifyAnyMessage(ctx context.Context, title, body, email string) error {
	body, err := Render(AnyMessage, anyTemplate{Title: title, Body: body, FooterMessage: happyDayTemplateFooter})
	if err != nil {
		return fmt.Errorf("mailer.NotifyAnyMessage: failed to render any message: %w", err)
	}

	m.enqueue(ctx, job{
		subject: fmt.Sprintf("ServiceHub | %s", title),
		body:    body,
		to:      email,
	})

	return nil
}

func (m *SMTPMailer) NotifyBalanceUpdate(ctx context.Context, amount, newBalance float64, username, email string) error {
	body, err := Render(BalanceUpdate, balanceUpdateTemplate{
		Username: username, Amount: amount, NewBalance: newBalance, FooterMessage: balanceUpdatedTemplateFooter})
	if err != nil {
		return fmt.Errorf("mailer.NotifyBalanceUpdate: failed to render balance update: %w", err)
	}

	m.enqueue(ctx, job{
		subject: "ServiceHub | Пополнение баланса",
		body:    body,
		to:      email,
	})

	return nil
}
