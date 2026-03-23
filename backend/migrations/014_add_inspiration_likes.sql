-- 014_add_inspiration_likes.sql
-- Stores user like relations for inspiration posts.

CREATE TABLE IF NOT EXISTS inspiration_likes (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY COMMENT 'Like ID',
    user_id BIGINT UNSIGNED NOT NULL COMMENT 'Liking user ID',
    post_id BIGINT UNSIGNED NOT NULL COMMENT 'Liked inspiration post ID',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT 'Like time',

    UNIQUE KEY uk_inspiration_likes_user_post (user_id, post_id),
    INDEX idx_inspiration_likes_user_id (user_id),
    INDEX idx_inspiration_likes_post_id (post_id),
    INDEX idx_inspiration_likes_created_at (created_at),

    CONSTRAINT fk_inspiration_likes_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    CONSTRAINT fk_inspiration_likes_post FOREIGN KEY (post_id) REFERENCES inspiration_posts(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='User likes for inspiration posts';
