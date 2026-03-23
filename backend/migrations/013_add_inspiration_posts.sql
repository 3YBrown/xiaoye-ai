-- 013_add_inspiration_posts.sql
-- Adds decoupled public inspiration posts sourced from user generations.

CREATE TABLE IF NOT EXISTS inspiration_posts (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY COMMENT 'Inspiration post ID',
    share_id VARCHAR(40) NOT NULL UNIQUE COMMENT 'Public share ID',
    user_id BIGINT UNSIGNED NOT NULL COMMENT 'Author user ID',
    source_generation_id BIGINT UNSIGNED NOT NULL UNIQUE COMMENT 'Source generation ID',
    type VARCHAR(20) NOT NULL DEFAULT 'image' COMMENT 'Generation type (image/video/ecommerce)',
    prompt LONGTEXT COMMENT 'Public prompt',
    params TEXT COMMENT 'Generation params JSON',
    reference_images TEXT COMMENT 'Reference image URLs JSON array',
    images TEXT COMMENT 'Output image URLs JSON array',
    video_url VARCHAR(500) COMMENT 'Output video URL',
    cover_url VARCHAR(500) COMMENT 'Card cover URL',
    status VARCHAR(20) NOT NULL DEFAULT 'published' COMMENT 'Status (published/hidden/rejected)',
    view_count INT NOT NULL DEFAULT 0 COMMENT 'View count',
    like_count INT NOT NULL DEFAULT 0 COMMENT 'Like count',
    remix_count INT NOT NULL DEFAULT 0 COMMENT 'Remix count',
    published_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT 'Published time',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT 'Created time',
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT 'Updated time',

    INDEX idx_inspiration_posts_user_id (user_id),
    INDEX idx_inspiration_posts_type (type),
    INDEX idx_inspiration_posts_status (status),
    INDEX idx_inspiration_posts_published_at (published_at),
    INDEX idx_inspiration_posts_source_generation_id (source_generation_id),

    CONSTRAINT fk_inspiration_posts_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    CONSTRAINT fk_inspiration_posts_generation FOREIGN KEY (source_generation_id) REFERENCES generations(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='Public inspiration posts';
