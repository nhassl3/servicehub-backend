CREATE MATERIALIZED VIEW IF NOT EXISTS analytics.mv_moderations_daily
  TO analytics.agg_moderations_daily
AS
SELECT toStartOfDay(occurred_at) AS day, admin_id, admin_username, count() AS cnt
FROM analytics.events
WHERE event_type IN ('moderation.approved', 'moderation.rejected')
GROUP BY day, admin_id, admin_username;