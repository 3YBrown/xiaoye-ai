-- 019_upgrade_inspiration_publish_schema.sql
-- Upgrades inspiration publishing schema for user-uploaded posts and normalized tags.

ALTER TABLE inspiration_posts
    MODIFY COLUMN source_generation_id BIGINT UNSIGNED NULL COMMENT 'Source generation ID',
    ADD COLUMN source_type VARCHAR(20) NOT NULL DEFAULT 'generation' COMMENT 'Post source (generation/upload)' AFTER source_generation_id,
    ADD COLUMN review_status VARCHAR(20) NOT NULL DEFAULT 'approved' COMMENT 'Review status (pending/approved/rejected)' AFTER status,
    ADD COLUMN review_note VARCHAR(1000) NULL COMMENT 'Review note' AFTER review_status,
    ADD COLUMN reviewed_by_source VARCHAR(50) NULL COMMENT 'Reviewer source system' AFTER review_note,
    ADD COLUMN reviewed_by_id VARCHAR(100) NULL COMMENT 'Reviewer identifier in source system' AFTER reviewed_by_source,
    ADD COLUMN reviewed_at DATETIME NULL COMMENT 'Review time' AFTER reviewed_by_id,
    ADD INDEX idx_inspiration_posts_source_type (source_type),
    ADD INDEX idx_inspiration_posts_review_status (review_status);

CREATE TABLE IF NOT EXISTS inspiration_post_media (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY COMMENT 'Media ID',
    post_id BIGINT UNSIGNED NOT NULL COMMENT 'Inspiration post ID',
    media_type VARCHAR(20) NOT NULL COMMENT 'Media type (image/video)',
    url VARCHAR(1000) NOT NULL COMMENT 'Media URL',
    cover_url VARCHAR(1000) NULL COMMENT 'Video cover URL',
    sort_order INT NOT NULL DEFAULT 0 COMMENT 'Display sort order',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT 'Creation time',
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT 'Update time',

    INDEX idx_inspiration_post_media_post_id (post_id),
    INDEX idx_inspiration_post_media_media_type (media_type),
    INDEX idx_inspiration_post_media_sort_order (sort_order),

    CONSTRAINT fk_inspiration_post_media_post FOREIGN KEY (post_id) REFERENCES inspiration_posts(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='Media assets for inspiration posts';

CREATE TABLE IF NOT EXISTS inspiration_tags (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY COMMENT 'Tag ID',
    name VARCHAR(50) NOT NULL COMMENT 'Display tag name',
    slug VARCHAR(80) NOT NULL COMMENT 'Normalized tag slug',
    status VARCHAR(20) NOT NULL DEFAULT 'active' COMMENT 'Tag status (active/hidden/blocked)',
    usage_count INT NOT NULL DEFAULT 0 COMMENT 'Number of linked posts',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT 'Creation time',
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT 'Update time',

    UNIQUE KEY uk_inspiration_tags_name (name),
    UNIQUE KEY uk_inspiration_tags_slug (slug),
    INDEX idx_inspiration_tags_status (status),
    INDEX idx_inspiration_tags_usage_count (usage_count)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='Normalized inspiration tags';

CREATE TABLE IF NOT EXISTS inspiration_post_tags (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY COMMENT 'Relation ID',
    post_id BIGINT UNSIGNED NOT NULL COMMENT 'Inspiration post ID',
    tag_id BIGINT UNSIGNED NOT NULL COMMENT 'Tag ID',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT 'Creation time',

    UNIQUE KEY uk_inspiration_post_tags_post_tag (post_id, tag_id),
    INDEX idx_inspiration_post_tags_tag_post (tag_id, post_id),
    INDEX idx_inspiration_post_tags_post_id (post_id),

    CONSTRAINT fk_inspiration_post_tags_post FOREIGN KEY (post_id) REFERENCES inspiration_posts(id) ON DELETE CASCADE,
    CONSTRAINT fk_inspiration_post_tags_tag FOREIGN KEY (tag_id) REFERENCES inspiration_tags(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='Tag relations for inspiration posts';

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
