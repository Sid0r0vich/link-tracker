-- +goose Up
-- +goose StatementBegin
CREATE TABLE chat (
    id BIGINT PRIMARY KEY
);

CREATE TABLE subscription (
    id BIGSERIAL PRIMARY KEY,
    url TEXT NOT NULL UNIQUE,
    updated_at TIMESTAMP NOT NULL
);

CREATE TABLE chat_subscription (
    chat_id BIGINT NOT NULL REFERENCES chat (id) ON DELETE CASCADE,
    subscription_id BIGINT NOT NULL REFERENCES subscription (id) ON DELETE CASCADE,
    PRIMARY KEY (chat_id, subscription_id)
);

CREATE TABLE subscription_tag (
    subscription_id BIGINT NOT NULL REFERENCES subscription (id) ON DELETE CASCADE,
    tag TEXT NOT NULL,
    PRIMARY KEY (subscription_id, tag)
);
-- +goose StatementEnd
