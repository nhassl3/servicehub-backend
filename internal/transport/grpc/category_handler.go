package grpc

import (
	"context"

	"github.com/nhassl3/servicehub-backend/internal/domain"
	"github.com/nhassl3/servicehub-backend/internal/service"
	categoryv1 "github.com/nhassl3/servicehub-contracts/pkg/pb/category/v1"
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
	return &categoryv1.ListCategoriesResponse{Categories: protoCategoriesInfo(*cats)}, nil
}

func (h *CategoryHandler) UploadAvatar(ctx context.Context, req *categoryv1.UploadAvatarRequest) (*categoryv1.UploadAvatarResponse, error) {
	category, err := h.svc.UploadCategoryIcon(ctx, domain.UploadCategoryIconParams{
		Slug:        req.GetSlug(),
		FileData:    req.GetFileData(),
		ContentType: req.GetContentType(),
	})
	if err != nil {
		return nil, domainErr(err)
	}
	return &categoryv1.UploadAvatarResponse{
		Category: protoCategoryInfo(category),
	}, nil
}

// ── Shared proto mapper ───────────────────────────────────────────────────────

func protoCategoriesInfo(cats domain.ListCategories) []*categoryv1.Category {
	proto := make([]*categoryv1.Category, len(cats))
	for i, c := range cats {
		proto[i] = protoCategoryInfo(&c)
	}
	return proto
}

func protoCategoryInfo(cat *domain.Category) *categoryv1.Category {
	return &categoryv1.Category{
		Id:          int32(cat.ID),
		Slug:        cat.Slug,
		Name:        cat.Name,
		Description: cat.Description,
		IconUrl:     cat.IconURL,
	}
}
