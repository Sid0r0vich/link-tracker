-- +goose Down
-- +goose StatementBegin
ALTER TABLE chat_subscription
    DROP COLUMN id,
    ADD PRIMARY KEY (chat_id, subscription_id),
    DROP UNIQUE (chat_id, subscription_id);

ALTER TABLE subscription
    ALTER COLUMN updated_at TYPE TIMESTAMP USING updated_at AT TIME ZONE 'UTC';

ALTER TABLE subscription_tag
    DROP COLUMN chat_subscription_id,
    ADD COLUMN subscription_id BIGINT NOT NULL REFERENCES subscription (id) ON DELETE CASCADE;
-- +goose StatementEnd
