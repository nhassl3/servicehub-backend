package domain

import (
	"context"
	"io"
)

type PhotoStorage interface {
	Upload(ctx context.Context, objectName, contentType string, reader io.Reader, size int64) (url string, err error)
	Delete(ctx context.Context, objectName string) (err error)
}
