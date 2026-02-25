CREATE TABLE IF NOT EXISTS categories (
    id          SERIAL       PRIMARY KEY,
    slug        VARCHAR(100) NOT NULL UNIQUE,
    name        VARCHAR(255) NOT NULL,
    description TEXT         NOT NULL DEFAULT '',
    icon_url    TEXT         NOT NULL DEFAULT '',
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

-- Seed 6 categories
INSERT INTO categories (slug, name, description) VALUES
    ('api-services',    'API Services',    'REST and GraphQL API integrations'),
    ('osint',           'OSINT',           'Open-source intelligence tools and data'),
    ('parsers',         'Parsers',         'Web scrapers and data extractors'),
    ('software',        'Software',        'Desktop and server software'),
    ('general',         'General Services','General digital services'),
    ('scripts',         'Scripts',         'Automation and utility scripts')
ON CONFLICT DO NOTHING;
