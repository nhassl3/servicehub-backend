CREATE MATERIALIZED VIEW IF NOT EXISTS analytics.mv_categories_sales_daily
  TO analytics.agg_categories_sales_daily
AS
SELECT toStartOfDay(occurred_at) AS day, category_id, sum(quantity) AS qty, sum(total) AS total
FROM analytics.events
WHERE event_type = 'order_item_created'
GROUP BY day, category_id;
