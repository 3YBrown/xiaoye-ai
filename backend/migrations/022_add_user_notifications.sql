-- 022_add_user_notifications.sql
-- Adds per-user notification table for in-app bell notifications.

CREATE TABLE IF NOT EXISTS user_notifications (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY COMMENT 'Notification ID',
    user_id BIGINT UNSIGNED NOT NULL COMMENT 'Receiver user ID',
    biz_key VARCHAR(100) NULL COMMENT 'Business key for idempotent writes',
    title VARCHAR(200) NOT NULL DEFAULT '' COMMENT 'Notification title',
    summary VARCHAR(500) NOT NULL DEFAULT '' COMMENT 'Notification summary',
    content TEXT COMMENT 'Notification content',
    is_read TINYINT(1) NOT NULL DEFAULT 0 COMMENT 'Whether read',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT 'Created time',
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT 'Updated time',

    UNIQUE KEY uk_user_notifications_user_biz_key (user_id, biz_key),
    INDEX idx_user_notifications_user_id (user_id),
    INDEX idx_user_notifications_user_is_read (user_id, is_read),
    INDEX idx_user_notifications_created_at (created_at),

    CONSTRAINT fk_user_notifications_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='In-app notifications';
