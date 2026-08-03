create table if not exists notifications (
                                             id UUID primary key default gen_random_uuid(),
                                             username varchar not null,
                                             message text not null,
                                             group_of_message integer not null references notification_group(id),
                                             created_at timestamptz not null default (now())
);

alter table notifications add foreign key (username) references users(username);

CREATE INDEX idx_notification_group_id ON notifications(group_of_message);
