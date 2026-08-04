package clickhouse

import (
	"context"
	"fmt"

	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"github.com/nhassl3/servicehub-backend/internal/domain"
)

// AnalyticsRepo reads/writes OLAP analytics facts in ClickHouse.
type AnalyticsRepo struct {
	conn driver.Conn
}

// NewAnalyticsRepo creates the ClickHouse analytics repository.
func NewAnalyticsRepo(conn driver.Conn) *AnalyticsRepo {
	return &AnalyticsRepo{conn: conn}
}

// InsertEvents batch-inserts normalized analytics facts into the event log.
func (r *AnalyticsRepo) InsertEvents(ctx context.Context, events []domain.AnalyticsEvent) error {
	if len(events) == 0 {
		return nil
	}

	batch, err := r.conn.PrepareBatch(ctx, "INSERT INTO analytics.events")
	if err != nil {
		return fmt.Errorf("clickhouse.InsertEvents prepare: %w", err)
	}
	for _, e := range events {
		if err := batch.Append(
			e.OccurredAt,
			string(e.EventType), // EventType -> string
			e.ProductID,
			e.SellerID,
			e.AdminID,
			e.AdminUsername,
			uint32(e.CategoryID),
			e.Title,
			e.Status,
			e.Rating,
			uint32(e.ReviewsCount),
			uint32(e.SalesCount),
			e.Reason,
			e.OrderID,
			uint32(e.Quantity),
			e.Total,
		); err != nil {
			return fmt.Errorf("clickhouse.analytics insert append: %w", err)
		}
	}
	if err := batch.Send(); err != nil {
		return fmt.Errorf("clickhouse.analytics insert send: %w", err)
	}
	return nil
}

// GetAdminStatistics runs the OLAP aggregates for the admin dashboard.
func (r *AnalyticsRepo) GetAdminStatistics(ctx context.Context, params domain.AdminStatisticsParams) (*domain.AdminStatistics, error) {
	stats := &domain.AdminStatistics{}

	if err := r.loadProducts(ctx, &stats.Products, params); err != nil {
		return nil, err
	}
	if err := r.loadTopProducts(ctx, &stats.TopProducts, params); err != nil {
		return nil, err
	}
	if err := r.loadTopCategories(ctx, &stats.TopCategories, params); err != nil {
		return nil, err
	}
	if err := r.loadRegistrations(ctx, &stats.Registrations, params); err != nil {
		return nil, err
	}
	if err := r.loadModerations(ctx, &stats.Moderations, params); err != nil {
		return nil, err
	}
	return stats, nil
}

func (r *AnalyticsRepo) loadProducts(ctx context.Context, out *domain.ProductStatusStats, params domain.AdminStatisticsParams) error {
	rows, err := r.conn.Query(ctx, fmt.Sprintf(`
		SELECT status, count()
		FROM (
			SELECT product_id, argMax(status, occurred_at) AS status
			FROM analytics.events
			WHERE event_type = '%s' AND occurred_at >= ? AND occurred_at <= ?
			GROUP BY product_id
		)
		GROUP BY status`, domain.ProductStatusChangedEventType),
		params.From, params.To)
	if err != nil {
		return fmt.Errorf("analytics loadProducts: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var status string
		var cnt uint64
		if err := rows.Scan(&status, &cnt); err != nil {
			return fmt.Errorf("analytics loadProducts scan: %w", err)
		}
		switch status {
		case "active":
			out.Verified = int(cnt)
		case "draft":
			out.Pending = int(cnt)
		case "inactive":
			out.Rejected = int(cnt)
		}
	}
	return rows.Err()
}

func (r *AnalyticsRepo) loadTopProducts(ctx context.Context, top *[]domain.TopProduct, params domain.AdminStatisticsParams) error {
	rows, err := r.conn.Query(ctx, fmt.Sprintf(`
		SELECT product_id,
		       argMax(title, occurred_at) AS title,
		       argMax(category_id, occurred_at) AS category_id,
		       argMax(rating, occurred_at) AS rating,
		       argMax(sales_count, occurred_at) AS sales_count,
		       argMax(reviews_count, occurred_at) AS reviews_count
		FROM analytics.events
		WHERE event_type IN ('%s', '%s')
		  AND occurred_at >= ? AND occurred_at <= ?
		GROUP BY product_id
		ORDER BY rating DESC
		LIMIT 20`, domain.ProductStatusChangedEventType, domain.ProductRatingChangedEventType),
		params.From, params.To)
	if err != nil {
		return fmt.Errorf("analytics loadTopProducts: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var p domain.TopProduct
		var cat, sales, reviews uint64
		if err := rows.Scan(&p.ID, &p.Title, &cat, &p.Rating, &sales, &reviews); err != nil {
			return fmt.Errorf("analytics loadTopProducts scan: %w", err)
		}
		p.CategoryID = int(cat)
		p.SalesCount = int(sales)
		p.ReviewsCount = int(reviews)
		*top = append(*top, p)
	}
	return rows.Err()
}

func (r *AnalyticsRepo) loadTopCategories(ctx context.Context, cats *[]domain.CategorySales, params domain.AdminStatisticsParams) error {
	rows, err := r.conn.Query(ctx, fmt.Sprintf(`
		SELECT category_id, sum(quantity) AS sales, sum(total) AS total
		FROM analytics.events
		WHERE event_type = '%s'
		  AND occurred_at >= ? AND occurred_at <= ?
		GROUP BY category_id
		ORDER BY total DESC
		LIMIT 10`, domain.OrderItemCreatedEventType),
		params.From, params.To)
	if err != nil {
		return fmt.Errorf("analytics loadTopCategories: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var c domain.CategorySales
		var cat, sales uint64
		if err := rows.Scan(&cat, &sales, &c.Total); err != nil {
			return fmt.Errorf("analytics loadTopCategories scan: %w", err)
		}
		c.CategoryID = int(cat)
		c.SalesCount = int(sales)
		*cats = append(*cats, c)
	}
	return rows.Err()
}

// bucketExpr returns the ClickHouse rounding function per requested granularity.
func bucketExpr(granularity string) string {
	if granularity == "hour" {
		return "toStartOfHour(occurred_at)"
	}
	return "toStartOfDay(occurred_at)"
}

func (r *AnalyticsRepo) loadRegistrations(ctx context.Context, regs *[]domain.RegistrationPoint, params domain.AdminStatisticsParams) error {
	expr := bucketExpr(params.Granularity)
	rows, err := r.conn.Query(ctx, fmt.Sprintf(`
		SELECT %s AS bucket, count() AS cnt
		FROM analytics.events
		WHERE event_type = '%s' AND occurred_at >= ? AND occurred_at <= ?
		GROUP BY bucket
		ORDER BY bucket`, expr, domain.UserRegisteredEventType), params.From, params.To)
	if err != nil {
		return fmt.Errorf("analytics loadRegistrations: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var rp domain.RegistrationPoint
		var cnt uint64
		if err := rows.Scan(&rp.Bucket, &cnt); err != nil {
			return fmt.Errorf("analytics loadRegistrations scan: %w", err)
		}
		rp.Count = int(cnt)
		*regs = append(*regs, rp)
	}
	return rows.Err()
}

func (r *AnalyticsRepo) loadModerations(ctx context.Context, mods *[]domain.ModerationPoint, params domain.AdminStatisticsParams) error {
	expr := bucketExpr(params.Granularity)
	rows, err := r.conn.Query(ctx, fmt.Sprintf(`
		SELECT %s AS bucket, admin_id, admin_username, count() AS cnt
		FROM analytics.events
		WHERE event_type IN ('%s','%s')
		  AND occurred_at >= ? AND occurred_at <= ?
		GROUP BY bucket, admin_id, admin_username
		ORDER BY bucket, admin_id`, expr, domain.ModerationApprovedEventType, domain.ModerationRejectedEventType), params.From, params.To)
	if err != nil {
		return fmt.Errorf("analytics loadModerations: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var mp domain.ModerationPoint
		var cnt uint64
		if err := rows.Scan(&mp.Bucket, &mp.AdminID, &mp.AdminUsername, &cnt); err != nil {
			return fmt.Errorf("analytics loadModerations scan: %w", err)
		}
		mp.Count = int(cnt)
		*mods = append(*mods, mp)
	}
	return rows.Err()
}
