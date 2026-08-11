CREATE TABLE IF NOT EXISTS analytics.agg_categories_sales_daily (
                                                                    day         DateTime('UTC'),
    category_id UInt32,
    category_name String,
    qty         UInt64,
    total       Float64
    ) ENGINE = SummingMergeTree
    PARTITION BY toYYYYMM(day)
    ORDER BY (day, category_id, category_name)
    TTL day + INTERVAL 3 YEAR;