CREATE TABLE IF NOT EXISTS analytics.agg_moderations_daily (
    day            DateTime('UTC'),
    admin_id       String,
    admin_username String,
    cnt            UInt64
) ENGINE = SummingMergeTree
  PARTITION BY toYYYYMM(day)
  ORDER BY (day, admin_id)
  TTL day + INTERVAL 3 YEAR;