CREATE MATERIALIZED VIEW IF NOT EXISTS analytics.mv_registrations_daily
  TO analytics.agg_registrations_daily
AS
SELECT toStartOfDay(occurred_at) AS day, count() AS cnt
FROM analytics.events
WHERE event_type = 'user_registered'
GROUP BY day;