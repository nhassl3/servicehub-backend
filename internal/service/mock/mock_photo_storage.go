package mock

import (
	"context"
	"io"
)

type MockPhotoStorage struct {
	UploadFunc func(ctx context.Context, objectName, contentType string, reader io.Reader, size int64) (string, error)
	DeleteFunc func(ctx context.Context, objectName string) error
}

func (m *MockPhotoStorage) Upload(ctx context.Context, objectName, contentType string, reader io.Reader, size int64) (string, error) {
	if m.UploadFunc != nil {
		return m.UploadFunc(ctx, objectName, contentType, reader, size)
	}
	return "http://localhost:9000/servicehub-avatars/" + objectName, nil
}

func (m *MockPhotoStorage) Delete(ctx context.Context, objectName string) error {
	if m.DeleteFunc != nil {
		return m.DeleteFunc(ctx, objectName)
	}
	return nil
}
