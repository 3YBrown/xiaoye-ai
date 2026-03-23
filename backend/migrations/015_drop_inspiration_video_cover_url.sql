-- 015_drop_inspiration_video_cover_url.sql
-- Drops deprecated inspiration_posts.video_cover_url.

SET @col_exists := (
    SELECT COUNT(*)
    FROM information_schema.COLUMNS
    WHERE TABLE_SCHEMA = DATABASE()
      AND TABLE_NAME = 'inspiration_posts'
      AND COLUMN_NAME = 'video_cover_url'
);
SET @drop_sql := IF(
    @col_exists > 0,
    'ALTER TABLE inspiration_posts DROP COLUMN video_cover_url',
    'SELECT 1'
);
PREPARE stmt FROM @drop_sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;
