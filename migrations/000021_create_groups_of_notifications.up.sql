create table if not exists notification_group (
                                                   id          SERIAL       PRIMARY KEY,
                                                   slug        VARCHAR(25)  NOT NULL UNIQUE,
                                                   name        VARCHAR(100) NOT NULL,
                                                   description TEXT         NOT NULL DEFAULT '',
                                                   icon_url    TEXT         NOT NULL DEFAULT '',
                                                   created_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

-- Seed 3 group
INSERT INTO notification_group (slug, name, description) VALUES
                                                              ('mnggood',    'Manage product', 'All actions with product'),
                                                              ('mngacc',           'Manage account', 'All actions with account'),
                                                              ('trans',         'Transactions', 'Billing notifications')
ON CONFLICT DO NOTHING;