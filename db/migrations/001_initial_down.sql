-- +goose Down
-- +goose StatementBegin
DROP TABLE chat_subscription;
DROP TABLE subscription_tag;
DROP TABLE subscription;
DROP TABLE chat;
-- +goose StatementEnd
