package elasticsearch

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"

	"github.com/elastic/go-elasticsearch/v9"
	"github.com/nhassl3/servicehub-backend/internal/domain"
	"go.uber.org/zap"
)

const elsIndex = "products"

type ProductESRepo struct {
	es  *elasticsearch.Client
	log *zap.Logger
}

func NewProductESRepo(es *elasticsearch.Client, log *zap.Logger) *ProductESRepo {
	return &ProductESRepo{es: es, log: log}
}

// EnsureIndex checks indexes and if not exists creates index in connected Docker container
// Require Docker and Elasticsearch >=v9.8.3 with auth (username, password) and established ports
func (r *ProductESRepo) EnsureIndex(ctx context.Context) error {
	resp, err := r.es.Indices.Exists(
		[]string{elsIndex},
		r.es.Indices.Exists.WithContext(ctx),
	)
	if err != nil {
		return fmt.Errorf("elasticsearch: check index exists: %w", err)
	}
	resp.Body.Close()

	if resp.StatusCode == 200 {
		return nil
	}

	body := buildIndexMapping()
	bodyBytes, _ := json.Marshal(body)

	resp, err = r.es.Indices.Create(
		elsIndex,
		r.es.Indices.Create.WithContext(ctx),
		r.es.Indices.Create.WithBody(bytes.NewReader(bodyBytes)),
	)
	if err != nil {
		return fmt.Errorf("elasticsearch: create index: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode > 299 {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("elasticsearch: create index: status %d: %s", resp.StatusCode, string(b))
	}

	r.log.Info("elasticsearch: index created", zap.String("index", elsIndex))
	return nil
}

func (r *ProductESRepo) IndexProduct(ctx context.Context, product *domain.Product) error {
	bodyBytes, err := json.Marshal(product)
	if err != nil {
		return fmt.Errorf("elasticsearch: marshal product: %w", err)
	}

	resp, err := r.es.Index(
		elsIndex,
		bytes.NewReader(bodyBytes),
		r.es.Index.WithContext(ctx),
		r.es.Index.WithDocumentID(product.ID),
	)
	if err != nil {
		return fmt.Errorf("elasticsearch: index product: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode > 299 {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("elasticsearch: index product: status %d: %s", resp.StatusCode, string(b))
	}

	return nil
}

func (r *ProductESRepo) DeleteProductIndex(ctx context.Context, id string) error {
	resp, err := r.es.Delete(
		elsIndex,
		id,
		r.es.Delete.WithContext(ctx),
	)
	if err != nil {
		return fmt.Errorf("elasticsearch: delete product: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode > 299 && resp.StatusCode != 404 {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("elasticsearch: delete product: status %d: %s", resp.StatusCode, string(b))
	}

	return nil
}

func (r *ProductESRepo) BulkIndexProducts(ctx context.Context, products []*domain.Product) error {
	var buf bytes.Buffer
	for _, p := range products {
		meta := map[string]interface{}{
			"index": map[string]string{
				"_index": elsIndex,
				"_id":    p.ID,
			},
		}
		metaBytes, err := json.Marshal(meta)
		if err != nil {
			return fmt.Errorf("elasticsearch: bulk marshal meta: %w", err)
		}
		buf.Write(metaBytes)
		buf.WriteByte('\n')

		docBytes, err := json.Marshal(p)
		if err != nil {
			return fmt.Errorf("elasticsearch: bulk marshal doc: %w", err)
		}
		buf.Write(docBytes)
		buf.WriteByte('\n')
	}

	resp, err := r.es.Bulk(
		&buf,
		r.es.Bulk.WithContext(ctx),
		r.es.Bulk.WithIndex(elsIndex),
	)
	if err != nil {
		return fmt.Errorf("elasticsearch: bulk index: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode > 299 {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("elasticsearch: bulk index: status %d: %s", resp.StatusCode, string(b))
	}

	return nil
}

func (r *ProductESRepo) Search(ctx context.Context, params domain.SearchProductsParams) ([]*domain.Product, int64, error) {
	body := buildSearchBody(params)
	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return nil, 0, fmt.Errorf("elasticsearch: marshal search body: %w", err)
	}

	resp, err := r.es.Search(
		r.es.Search.WithContext(ctx),
		r.es.Search.WithIndex(elsIndex),
		r.es.Search.WithBody(bytes.NewReader(bodyBytes)),
		r.es.Search.WithFrom(int(params.Offset)),
		r.es.Search.WithSize(int(params.Limit)),
		r.es.Search.WithTrackTotalHits(body.TrackTotalHits),
	)
	if err != nil {
		return nil, 0, fmt.Errorf("elasticsearch: search: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode > 299 {
		b, _ := io.ReadAll(resp.Body)
		return nil, 0, fmt.Errorf("elasticsearch: search: status %d: %s", resp.StatusCode, string(b))
	}

	var result struct {
		Hits struct {
			Total struct {
				Value int64 `json:"value"`
			} `json:"total"`
			Hits []struct {
				Source domain.Product `json:"_source"`
			} `json:"hits"`
		} `json:"hits"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, 0, fmt.Errorf("elasticsearch: decode response: %w", err)
	}

	products := make([]*domain.Product, len(result.Hits.Hits))
	for i, h := range result.Hits.Hits {
		products[i] = &h.Source
	}

	return products, result.Hits.Total.Value, nil
}

// ── Index mapping ───────────────────────────────────────────────────────

func buildIndexMapping() map[string]interface{} {
	return map[string]interface{}{
		"settings": map[string]interface{}{
			"analysis": map[string]interface{}{
				"analyzer": map[string]interface{}{
					"default": map[string]string{
						"type": "standard",
					},
				},
			},
		},
		"mappings": map[string]interface{}{
			"properties": map[string]interface{}{
				"id":          map[string]string{"type": "keyword"},
				"seller_id":   map[string]string{"type": "keyword"},
				"category_id": map[string]string{"type": "integer"},
				"status":      map[string]string{"type": "keyword"},
				"price": map[string]interface{}{
					"type":           "scaled_float", // 2.34 ~ 234 in scaled float type by scaling factor 100
					"scaling_factor": 100,            // scaling factor 2.34 * 100 = 234
				},
				"tags":          map[string]string{"type": "keyword"},
				"sales_count":   map[string]string{"type": "integer"},
				"rating":        map[string]string{"type": "float"},
				"reviews_count": map[string]string{"type": "integer"},
				"created_at":    map[string]string{"type": "date", "format": "yyyy-MM-dd'T'HH:mm:ss'Z'"},
				"updated_at":    map[string]string{"type": "date", "format": "yyyy-MM-dd'T'HH:mm:ss'Z'"},
				"title": map[string]interface{}{
					"type":     "text",
					"analyzer": "standard",
					"fields": map[string]interface{}{
						"russian": map[string]interface{}{
							"type":     "text",
							"analyzer": "russian",
						},
					},
				},
				"description": map[string]interface{}{
					"type":     "text",
					"analyzer": "standard",
					"fields": map[string]interface{}{
						"russian": map[string]interface{}{
							"type":     "text",
							"analyzer": "russian",
						},
					},
				},
			},
		},
	}
}

// ── Search body builder ─────────────────────────────────────────────────

type searchBody struct {
	Query          searchQuery             `json:"query"`
	Sort           []map[string]sortClause `json:"sort"`
	TrackTotalHits bool                    `json:"track_total_hits"`
}

type searchQuery struct {
	Bool *boolQuery `json:"bool,omitempty"`
}

type boolQuery struct {
	Must   interface{} `json:"must,omitempty"`
	Filter interface{} `json:"filter,omitempty"`
}

type sortClause struct {
	Order string `json:"order,omitempty"`
}

func buildSearchBody(params domain.SearchProductsParams) searchBody {
	var filters []interface{}

	filters = append(filters, map[string]interface{}{
		"term": map[string]string{"status": "active"},
	})

	if params.CategoryID != nil && *params.CategoryID > 0 {
		filters = append(filters, map[string]interface{}{
			"term": map[string]int{"category_id": *params.CategoryID},
		})
	}

	if params.MinPrice != nil || params.MaxPrice != nil {
		r := make(map[string]interface{})
		if params.MinPrice != nil && *params.MinPrice > 0 {
			r["gte"] = *params.MinPrice
		}
		if params.MaxPrice != nil && *params.MaxPrice > 0 && *params.MaxPrice < 1_000_000 { // price not higher 1_000_000
			r["lte"] = *params.MaxPrice
		}
		filters = append(filters, map[string]interface{}{
			"range": map[string]interface{}{"price": r},
		})
	}

	if len(params.Tags) > 0 {
		filters = append(filters, map[string]interface{}{
			"terms": map[string][]string{"tags": params.Tags},
		})
	}

	var must []interface{}

	if params.Query != "" {
		must = append(must, map[string]interface{}{
			"multi_match": map[string]interface{}{
				"query":    params.Query,
				"fields":   []string{"title^3", "title.russian", "description", "description.russian", "tags"},
				"type":     "best_fields",
				"operator": "or",
			},
		})
	}

	bq := &boolQuery{}
	if len(must) > 0 {
		bq.Must = must
	}
	if len(filters) > 0 {
		bq.Filter = filters
	}

	sorts := buildSortClauses(params.SortBy)

	return searchBody{
		Query:          searchQuery{Bool: bq},
		Sort:           sorts,
		TrackTotalHits: true,
	}
}

func buildSortClauses(sortBy string) []map[string]sortClause {
	switch sortBy {
	case "price_asc":
		return []map[string]sortClause{{"price": {Order: "asc"}}}
	case "price_desc":
		return []map[string]sortClause{{"price": {Order: "desc"}}}
	case "rating":
		return []map[string]sortClause{{"rating": {Order: "desc"}}}
	case "date":
		return []map[string]sortClause{{"created_at": {Order: "desc"}}}
	case "sales":
		return []map[string]sortClause{{"sales_count": {Order: "desc"}}}
	default:
		return []map[string]sortClause{{"_score": {Order: "desc"}}}
	}
}
