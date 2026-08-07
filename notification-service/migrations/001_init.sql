CREATE TABLE IF NOT EXISTS notifications (
    id          VARCHAR(36) PRIMARY KEY,
    user_id     VARCHAR(36) NOT NULL,
    type        VARCHAR(50) NOT NULL,
    channel     VARCHAR(20) NOT NULL DEFAULT 'email',
    recipient   VARCHAR(255) NOT NULL,
    status      VARCHAR(20) NOT NULL DEFAULT 'sent',
    created_at  TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS processed_events (
    event_id     VARCHAR(36) PRIMARY KEY,
    processed_at TIMESTAMP NOT NULL DEFAULT NOW()
);
