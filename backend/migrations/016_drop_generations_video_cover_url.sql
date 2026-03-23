-- 016_drop_generations_video_cover_url.sql
-- Drops deprecated generations.video_cover_url.

SET @col_exists := (
    SELECT COUNT(*)
    FROM information_schema.COLUMNS
    WHERE TABLE_SCHEMA = DATABASE()
      AND TABLE_NAME = 'generations'
      AND COLUMN_NAME = 'video_cover_url'
);
SET @drop_sql := IF(
    @col_exists > 0,
    'ALTER TABLE generations DROP COLUMN video_cover_url',
    'SELECT 1'
);
PREPARE stmt FROM @drop_sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;
