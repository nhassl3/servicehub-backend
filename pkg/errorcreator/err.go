package errorcreator

import "fmt"

func NewError(op, f, msg string, err error) error {
	return fmt.Errorf("%s.%s: %s: %w", op, f, msg, err)
}
