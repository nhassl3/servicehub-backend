package mock

import "context"

type MockNotifier struct {
	NotifyResetPasswordFunc     func(ctx context.Context, code, email string) error
	NotifyEmailConfirmationFunc func(ctx context.Context, code, email string) error
	NotifyAnyMessageFunc        func(ctx context.Context, title, body, email string) error
	NotifyBalanceUpdateFunc     func(ctx context.Context, amount, newBalance float64, username, email string) error
	CloseFunc                   func(ctx context.Context) error
}

func (m *MockNotifier) NotifyResetPassword(ctx context.Context, code, email string) error {
	if m.NotifyResetPasswordFunc != nil {
		return m.NotifyResetPasswordFunc(ctx, code, email)
	}
	return nil
}

func (m *MockNotifier) NotifyEmailConfirmation(ctx context.Context, code, email string) error {
	if m.NotifyEmailConfirmationFunc != nil {
		return m.NotifyEmailConfirmationFunc(ctx, code, email)
	}
	return nil
}

func (m *MockNotifier) NotifyAnyMessage(ctx context.Context, title, body, email string) error {
	if m.NotifyAnyMessageFunc != nil {
		return m.NotifyAnyMessageFunc(ctx, title, body, email)
	}
	return nil
}

func (m *MockNotifier) NotifyBalanceUpdate(ctx context.Context, amount, newBalance float64, username, email string) error {
	if m.NotifyBalanceUpdateFunc != nil {
		return m.NotifyBalanceUpdateFunc(ctx, amount, newBalance, username, email)
	}
	return nil
}

func (m *MockNotifier) Close(ctx context.Context) error {
	if m.CloseFunc != nil {
		return m.CloseFunc(ctx)
	}
	return nil
}
