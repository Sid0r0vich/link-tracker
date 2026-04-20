-- +goose Up
-- +goose StatementBegin
ALTER TABLE chat_subscription
    DROP CONSTRAINT chat_subscription_pkey,
    ADD COLUMN id BIGSERIAL PRIMARY KEY,
    ADD UNIQUE (chat_id, subscription_id);

ALTER TABLE subscription
    ALTER COLUMN updated_at TYPE TIMESTAMPTZ USING updated_at AT TIME ZONE 'UTC';

ALTER TABLE subscription_tag 
    DROP COLUMN subscription_id,
    ADD COLUMN chat_subscription_id BIGINT NOT NULL REFERENCES chat_subscription (id) ON DELETE CASCADE;
-- +goose StatementEnd
