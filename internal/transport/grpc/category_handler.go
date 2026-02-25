package grpc

import (
	"context"

	categoryv1 "github.com/nhassl3/servicehub-contracts/pkg/pb/category/v1"
	"github.com/nhassl3/servicehub/internal/service"
)

// CategoryHandler implements categoryv1.CategoryServiceServer.
//
// Implemented RPC methods:
//   - ListCategories
type CategoryHandler struct {
	categoryv1.UnimplementedCategoryServiceServer
	svc *service.CategoryService
}

func NewCategoryHandler(svc *service.CategoryService) *CategoryHandler {
	return &CategoryHandler{svc: svc}
}

func (h *CategoryHandler) ListCategories(ctx context.Context, _ *categoryv1.ListCategoriesRequest) (*categoryv1.ListCategoriesResponse, error) {
	cats, err := h.svc.ListCategories(ctx)
	if err != nil {
		return nil, domainErr(err)
	}
	proto := make([]*categoryv1.Category, len(cats))
	for i, c := range cats {
		proto[i] = &categoryv1.Category{
			Id:          int32(c.ID),
			Slug:        c.Slug,
			Name:        c.Name,
			Description: c.Description,
			IconUrl:     c.IconURL,
		}
	}
	return &categoryv1.ListCategoriesResponse{Categories: proto}, nil
}
