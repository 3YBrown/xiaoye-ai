-- 018_add_inspiration_posts_title_description.sql
-- Adds title and description fields to inspiration_posts for richer sharing experience.

ALTER TABLE inspiration_posts
    ADD COLUMN title VARCHAR(200) COMMENT 'Post title' AFTER type,
    ADD COLUMN description VARCHAR(1000) COMMENT 'Post description' AFTER title;
