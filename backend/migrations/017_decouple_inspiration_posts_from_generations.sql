-- 017_decouple_inspiration_posts_from_generations.sql
-- Keep inspiration posts as independent published snapshots, even if source generations are deleted.

SET @fk_exists := (
  SELECT COUNT(1)
  FROM information_schema.REFERENTIAL_CONSTRAINTS
  WHERE CONSTRAINT_SCHEMA = DATABASE()
    AND CONSTRAINT_NAME = 'fk_inspiration_posts_generation'
    AND TABLE_NAME = 'inspiration_posts'
);

SET @drop_fk_sql := IF(
  @fk_exists > 0,
  'ALTER TABLE inspiration_posts DROP FOREIGN KEY fk_inspiration_posts_generation',
  'SELECT 1'
);

PREPARE stmt FROM @drop_fk_sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;
