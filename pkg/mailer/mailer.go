package mailer

import (
	"context"
)

type Notifier interface {
	NotifyResetPassword(ctx context.Context, code, email string) error
	NotifyEmailConfirmation(ctx context.Context, code, email string) error
	NotifyAnyMessage(ctx context.Context, title, body, email string) error
	NotifyBalanceUpdate(ctx context.Context, amount, newBalance float64, username, email string) error
	Close(ctx context.Context) error
}
