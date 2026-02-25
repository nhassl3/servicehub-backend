package grpc

import (
	"context"

	productv1 "github.com/nhassl3/servicehub-contracts/pkg/pb/product/v1"
	"github.com/nhassl3/servicehub/internal/domain"
	"github.com/nhassl3/servicehub/internal/service"
	"github.com/nhassl3/servicehub/internal/transport/grpc/interceptors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// ProductHandler implements productv1.ProductServiceServer.
//
// Implemented RPC methods:
//   - ListProducts
//   - GetProduct
//   - SearchProducts
//   - CreateProduct
//   - UpdateProduct
//   - DeleteProduct
type ProductHandler struct {
	productv1.UnimplementedProductServiceServer
	svc *service.ProductService
}

func NewProductHandler(svc *service.ProductService) *ProductHandler {
	return &ProductHandler{svc: svc}
}

func (h *ProductHandler) ListProducts(ctx context.Context, req *productv1.ListProductsRequest) (*productv1.ListProductsResponse, error) {
	params := domain.ListProductsParams{
		Limit:  req.Limit,
		Offset: req.Offset,
	}
	if req.CategoryId != nil {
		v := int(*req.CategoryId)
		params.CategoryID = &v
	}
	if req.SellerId != nil {
		v := *req.SellerId
		params.SellerID = &v
	}
	if req.MinPrice != nil {
		v := *req.MinPrice
		params.MinPrice = &v
	}
	if req.MaxPrice != nil {
		v := *req.MaxPrice
		params.MaxPrice = &v
	}

	products, total, err := h.svc.ListProducts(ctx, params)
	if err != nil {
		return nil, domainErr(err)
	}
	proto := make([]*productv1.Product, len(products))
	for i, p := range products {
		proto[i] = protoProduct(&p)
	}
	return &productv1.ListProductsResponse{Products: proto, Total: total}, nil
}

func (h *ProductHandler) GetProduct(ctx context.Context, req *productv1.GetProductRequest) (*productv1.GetProductResponse, error) {
	p, err := h.svc.GetProduct(ctx, req.Id)
	if err != nil {
		return nil, domainErr(err)
	}
	return &productv1.GetProductResponse{Product: protoProduct(p)}, nil
}

func (h *ProductHandler) SearchProducts(ctx context.Context, req *productv1.SearchProductsRequest) (*productv1.SearchProductsResponse, error) {
	products, total, err := h.svc.SearchProducts(ctx, domain.SearchProductsParams{
		Query:  req.Query,
		Limit:  req.Limit,
		Offset: req.Offset,
	})
	if err != nil {
		return nil, domainErr(err)
	}
	proto := make([]*productv1.Product, len(products))
	for i, p := range products {
		proto[i] = protoProduct(&p)
	}
	return &productv1.SearchProductsResponse{Products: proto, Total: total}, nil
}

func (h *ProductHandler) CreateProduct(ctx context.Context, req *productv1.CreateProductRequest) (*productv1.CreateProductResponse, error) {
	payload, ok := interceptors.PayloadFromContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "missing auth payload")
	}
	p, err := h.svc.CreateProduct(ctx, payload.Username, domain.CreateProductParams{
		CategoryID:  int(req.CategoryId),
		Title:       req.Title,
		Description: req.Description,
		Price:       req.Price,
		Tags:        req.Tags,
	})
	if err != nil {
		return nil, domainErr(err)
	}
	return &productv1.CreateProductResponse{Product: protoProduct(p)}, nil
}

func (h *ProductHandler) UpdateProduct(ctx context.Context, req *productv1.UpdateProductRequest) (*productv1.UpdateProductResponse, error) {
	payload, ok := interceptors.PayloadFromContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "missing auth payload")
	}
	p, err := h.svc.UpdateProduct(ctx, payload.Username, domain.UpdateProductParams{
		ID:          req.Id,
		Title:       req.Title,
		Description: req.Description,
		Price:       req.Price,
		Tags:        req.Tags,
		Status:      req.Status,
	})
	if err != nil {
		return nil, domainErr(err)
	}
	return &productv1.UpdateProductResponse{Product: protoProduct(p)}, nil
}

func (h *ProductHandler) DeleteProduct(ctx context.Context, req *productv1.DeleteProductRequest) (*productv1.DeleteProductResponse, error) {
	payload, ok := interceptors.PayloadFromContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "missing auth payload")
	}
	if err := h.svc.DeleteProduct(ctx, payload.Username, req.Id); err != nil {
		return nil, domainErr(err)
	}
	return &productv1.DeleteProductResponse{Success: true}, nil
}

// ── Proto mapper ─────────────────────────────────────────────────────────────

func protoProduct(p *domain.Product) *productv1.Product {
	return &productv1.Product{
		Id:           p.ID,
		SellerId:     p.SellerID,
		CategoryId:   int32(p.CategoryID),
		Title:        p.Title,
		Description:  p.Description,
		Price:        p.Price,
		Tags:         p.Tags,
		Status:       p.Status,
		SalesCount:   int32(p.SalesCount),
		Rating:       p.Rating,
		ReviewsCount: int32(p.ReviewsCount),
		CreatedAt:    safeTimestamp(p.CreatedAt),
		UpdatedAt:    safeTimestamp(p.UpdatedAt),
	}
}
