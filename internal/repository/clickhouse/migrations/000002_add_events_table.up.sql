CREATE TABLE IF NOT EXISTS analytics.events (
                                                occurred_at    DateTime('UTC'),
    event_type     LowCardinality(String),
    product_id     String,
    seller_id      String,
    admin_id       String,
    admin_username String,
    category_id    UInt32,
    category_name String,
    title          String,
    status         LowCardinality(String),
    rating         Float64,
    reviews_count  UInt32,
    sales_count    UInt32,
    reason         String,
    order_id       String,
    quantity       UInt32,
    total          Float64
    ) ENGINE = MergeTree
    PARTITION BY toYYYYMM(occurred_at)
    ORDER BY (event_type, occurred_at)
    TTL occurred_at + INTERVAL 90 DAY;