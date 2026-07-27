package mailer

import (
	"context"
	"fmt"

	"go.uber.org/zap"
)

type NoopNotifier struct {
	log *zap.Logger
}

func NewNoopNotifier(log *zap.Logger) *NoopNotifier {
	return &NoopNotifier{
		log: log,
	}
}

func (n *NoopNotifier) NotifyResetPassword(_ context.Context, code, email string) error {
	n.log.Info("NEW RESET PASSWORD",
		zap.String("title", "Request new password reset"),
		zap.String("code", code),
		zap.String("email", email),
	)
	return nil
}

func (n *NoopNotifier) NotifyEmailConfirmation(_ context.Context, code, email string) error {
	n.log.Info("NEW EMAIL CONFIRMATION",
		zap.String("title", "Request new email confirmation"),
		zap.String("code", code),
		zap.String("email", email),
	)
	return nil
}

func (n *NoopNotifier) NotifyAnyMessage(_ context.Context, title, body, email string) error {
	n.log.Info("NEW ANY MESSAGE",
		zap.String("title", title),
		zap.String("body", body),
		zap.String("email", email),
	)
	return nil
}

func (n *NoopNotifier) NotifyBalanceUpdate(_ context.Context, amount, newBalance float64, username, email string) error {
	n.log.Info("NEW BALANCE UPDATE",
		zap.String("title", "Balance updated"),
		zap.String("amount", fmt.Sprintf("%.2f", amount)),
		zap.String("new_balance", fmt.Sprintf("%.2f", newBalance)),
		zap.String("username", username),
		zap.String("email", email),
	)
	return nil
}

func (n *NoopNotifier) Close(_ context.Context) error { return nil }
