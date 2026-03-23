-- 023_drop_generations_tags.sql
-- Drops deprecated generations.tags compatibility column.

SET @col_exists := (
    SELECT COUNT(*)
    FROM information_schema.COLUMNS
    WHERE TABLE_SCHEMA = DATABASE()
      AND TABLE_NAME = 'generations'
      AND COLUMN_NAME = 'tags'
);
SET @drop_sql := IF(
    @col_exists > 0,
    'ALTER TABLE generations DROP COLUMN tags',
    'SELECT 1'
);
PREPARE stmt FROM @drop_sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;
