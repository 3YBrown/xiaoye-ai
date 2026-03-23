-- 021_finalize_inspiration_review_schema.sql
-- Finalize review schema:
-- 1) Keep snapshot fields on inspiration_posts.
-- 2) Move note/history responsibility to inspiration_review_logs.

SET NAMES utf8mb4 COLLATE utf8mb4_unicode_ci;
SET collation_connection = 'utf8mb4_unicode_ci';

-- Drop deprecated review_note from inspiration_posts when present.
SET @has_review_note := (
    SELECT COUNT(1)
    FROM INFORMATION_SCHEMA.COLUMNS
    WHERE TABLE_SCHEMA = DATABASE()
      AND TABLE_NAME = 'inspiration_posts'
      AND COLUMN_NAME = 'review_note'
);
SET @sql := IF(@has_review_note > 0,
               'ALTER TABLE inspiration_posts DROP COLUMN review_note',
               'SELECT 1');
PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

-- Ensure review snapshot fields exist on inspiration_posts.
SET @has_review_status := (
    SELECT COUNT(1)
    FROM INFORMATION_SCHEMA.COLUMNS
    WHERE TABLE_SCHEMA = DATABASE()
      AND TABLE_NAME = 'inspiration_posts'
      AND COLUMN_NAME = 'review_status'
);
SET @sql := IF(@has_review_status = 0,
               'ALTER TABLE inspiration_posts ADD COLUMN review_status VARCHAR(20) NOT NULL DEFAULT ''approved'' COMMENT ''Review status (pending/approved/rejected)'' AFTER status',
               'SELECT 1');
PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @has_reviewed_by_source := (
    SELECT COUNT(1)
    FROM INFORMATION_SCHEMA.COLUMNS
    WHERE TABLE_SCHEMA = DATABASE()
      AND TABLE_NAME = 'inspiration_posts'
      AND COLUMN_NAME = 'reviewed_by_source'
);
SET @sql := IF(@has_reviewed_by_source = 0,
               'ALTER TABLE inspiration_posts ADD COLUMN reviewed_by_source VARCHAR(50) NULL COMMENT ''Reviewer source system'' AFTER review_status',
               'SELECT 1');
PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @has_reviewed_by_id := (
    SELECT COUNT(1)
    FROM INFORMATION_SCHEMA.COLUMNS
    WHERE TABLE_SCHEMA = DATABASE()
      AND TABLE_NAME = 'inspiration_posts'
      AND COLUMN_NAME = 'reviewed_by_id'
);
SET @sql := IF(@has_reviewed_by_id = 0,
               'ALTER TABLE inspiration_posts ADD COLUMN reviewed_by_id VARCHAR(100) NULL COMMENT ''Reviewer identifier in source system'' AFTER reviewed_by_source',
               'SELECT 1');
PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @has_reviewed_at := (
    SELECT COUNT(1)
    FROM INFORMATION_SCHEMA.COLUMNS
    WHERE TABLE_SCHEMA = DATABASE()
      AND TABLE_NAME = 'inspiration_posts'
      AND COLUMN_NAME = 'reviewed_at'
);
SET @sql := IF(@has_reviewed_at = 0,
               'ALTER TABLE inspiration_posts ADD COLUMN reviewed_at DATETIME NULL COMMENT ''Review time'' AFTER reviewed_by_id',
               'SELECT 1');
PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @has_idx_review_status := (
    SELECT COUNT(1)
    FROM INFORMATION_SCHEMA.STATISTICS
    WHERE TABLE_SCHEMA = DATABASE()
      AND TABLE_NAME = 'inspiration_posts'
      AND INDEX_NAME = 'idx_inspiration_posts_review_status'
);
SET @sql := IF(@has_idx_review_status = 0,
               'ALTER TABLE inspiration_posts ADD INDEX idx_inspiration_posts_review_status (review_status)',
               'SELECT 1');
PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

-- Keep review history table for future moderation/admin system.
CREATE TABLE IF NOT EXISTS inspiration_review_logs (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY COMMENT 'Review log ID',
    post_id BIGINT UNSIGNED NOT NULL COMMENT 'Inspiration post ID',
    action VARCHAR(30) NOT NULL COMMENT 'Review action',
    from_status VARCHAR(20) NULL COMMENT 'Status before action',
    to_status VARCHAR(20) NULL COMMENT 'Status after action',
    note VARCHAR(1000) NULL COMMENT 'Review note',
    operator_source VARCHAR(50) NULL COMMENT 'Operator source system',
    operator_id VARCHAR(100) NULL COMMENT 'Operator id in source system',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT 'Creation time',

    INDEX idx_inspiration_review_logs_post_id (post_id),
    INDEX idx_inspiration_review_logs_action (action),
    INDEX idx_inspiration_review_logs_created_at (created_at),

    CONSTRAINT fk_inspiration_review_logs_post FOREIGN KEY (post_id) REFERENCES inspiration_posts(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='Review logs for inspiration posts';
