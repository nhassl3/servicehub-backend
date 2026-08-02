CREATE TABLE IF NOT EXISTS analytics.agg_registrations_daily (
                                                                 day  DateTime('UTC'),
    cnt  UInt64
    ) ENGINE = SummingMergeTree
    PARTITION BY toYYYYMM(day)
    ORDER BY (day)
    TTL day + INTERVAL 3 YEAR;