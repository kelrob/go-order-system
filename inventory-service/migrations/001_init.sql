CREATE TABLE IF NOT EXISTS inventory_items (
    product_id  VARCHAR(36) PRIMARY KEY,
    quantity    INT NOT NULL DEFAULT 0,
    reserved    INT NOT NULL DEFAULT 0,
    available   INT NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS outbox (
    id          VARCHAR(36) PRIMARY KEY,
    event_type  VARCHAR(100) NOT NULL,
    payload     JSONB NOT NULL,
    status      VARCHAR(20) NOT NULL DEFAULT 'pending',
    created_at  TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS processed_events (
    event_id     VARCHAR(36) PRIMARY KEY,
    processed_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- Seed some products with stock
INSERT INTO inventory_items (product_id, quantity, reserved, available)
VALUES 
    ('prod-1', 100, 0, 100),
    ('prod-2', 50, 0, 50),
    ('prod-3', 75, 0, 75)
ON CONFLICT DO NOTHING;