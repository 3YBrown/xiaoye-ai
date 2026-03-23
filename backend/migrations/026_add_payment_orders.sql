-- Create payment_orders table for online payment (linux.do credit, future PayPal/Stripe)
CREATE TABLE IF NOT EXISTS payment_orders (
    id              BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
    user_id         BIGINT UNSIGNED NOT NULL,
    order_no        VARCHAR(64)     NOT NULL,
    provider        VARCHAR(30)     NOT NULL,
    provider_trade_no VARCHAR(100)  DEFAULT NULL,
    amount          VARCHAR(20)     NOT NULL,
    diamonds        INT             NOT NULL,
    plan_name       VARCHAR(50)     DEFAULT NULL,
    status          VARCHAR(20)     NOT NULL DEFAULT 'pending',
    notify_data     TEXT            DEFAULT NULL,
    paid_at         DATETIME        DEFAULT NULL,
    created_at      DATETIME        NOT NULL,
    updated_at      DATETIME        NOT NULL,

    UNIQUE INDEX idx_payment_orders_order_no (order_no),
    INDEX idx_payment_orders_user_id (user_id),
    INDEX idx_payment_orders_provider (provider),
    INDEX idx_payment_orders_status (status),
    INDEX idx_payment_orders_created_at (created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
