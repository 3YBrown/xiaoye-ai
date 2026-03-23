-- 020_refactor_inspiration_media_urls.sql
-- Simplify inspiration media model to one field:
-- inspiration_posts.media_urls(JSON array string) + type(image/video) + cover_url.

SET NAMES utf8mb4 COLLATE utf8mb4_unicode_ci;
SET collation_connection = 'utf8mb4_unicode_ci';

ALTER TABLE inspiration_posts
    ADD COLUMN media_urls TEXT NULL COMMENT 'Media URLs JSON array' AFTER reference_images;

-- Backfill media_urls from inspiration_post_media (preferred, if 019 was used).
UPDATE inspiration_posts p
SET p.media_urls = CASE
    WHEN p.type = 'video' THEN
        CASE
            WHEN EXISTS (
                SELECT 1 FROM inspiration_post_media pm
                WHERE pm.post_id = p.id AND pm.media_type = 'video' AND pm.url IS NOT NULL AND TRIM(pm.url) <> ''
            )
            THEN (
                SELECT JSON_ARRAY(pm.url)
                FROM inspiration_post_media pm
                WHERE pm.post_id = p.id AND pm.media_type = 'video' AND pm.url IS NOT NULL AND TRIM(pm.url) <> ''
                ORDER BY pm.sort_order ASC, pm.id ASC
                LIMIT 1
            )
            ELSE JSON_ARRAY()
        END
    ELSE
        CASE
            WHEN EXISTS (
                SELECT 1 FROM inspiration_post_media pm
                WHERE pm.post_id = p.id AND pm.media_type = 'image' AND pm.url IS NOT NULL AND TRIM(pm.url) <> ''
            )
            THEN (
                SELECT JSON_ARRAYAGG(s.url)
                FROM (
                    SELECT pm.url
                    FROM inspiration_post_media pm
                    WHERE pm.post_id = p.id AND pm.media_type = 'image' AND pm.url IS NOT NULL AND TRIM(pm.url) <> ''
                    ORDER BY pm.sort_order ASC, pm.id ASC
                ) s
            )
            ELSE JSON_ARRAY()
        END
END
WHERE p.media_urls IS NULL OR TRIM(p.media_urls) = '';

-- Fallback backfill from legacy columns for rows still empty.
UPDATE inspiration_posts p
SET p.media_urls = CASE
    WHEN p.type = 'video' THEN
        CASE
            WHEN p.video_url IS NOT NULL AND TRIM(p.video_url) <> '' THEN JSON_ARRAY(p.video_url)
            ELSE JSON_ARRAY()
        END
    ELSE
        CASE
            WHEN p.images IS NOT NULL AND JSON_VALID(p.images) = 1 THEN CAST(p.images AS JSON)
            ELSE JSON_ARRAY()
        END
END
WHERE p.media_urls IS NULL OR TRIM(p.media_urls) = '';

-- Ensure cover_url is populated.
UPDATE inspiration_posts p
SET p.cover_url = CASE
    WHEN p.cover_url IS NOT NULL AND TRIM(p.cover_url) <> '' THEN p.cover_url
    WHEN p.type = 'video' THEN
        CASE
            WHEN JSON_VALID(p.media_urls) = 1
                 AND JSON_UNQUOTE(JSON_EXTRACT(CAST(p.media_urls AS JSON), '$[0]')) IS NOT NULL
                 AND JSON_UNQUOTE(JSON_EXTRACT(CAST(p.media_urls AS JSON), '$[0]')) <> ''
            THEN JSON_UNQUOTE(JSON_EXTRACT(CAST(p.media_urls AS JSON), '$[0]'))
            WHEN p.video_url IS NOT NULL AND TRIM(p.video_url) <> '' THEN p.video_url
            ELSE p.cover_url
        END
    ELSE
        CASE
            WHEN JSON_VALID(p.media_urls) = 1
                 AND JSON_UNQUOTE(JSON_EXTRACT(CAST(p.media_urls AS JSON), '$[0]')) IS NOT NULL
                 AND JSON_UNQUOTE(JSON_EXTRACT(CAST(p.media_urls AS JSON), '$[0]')) <> ''
            THEN JSON_UNQUOTE(JSON_EXTRACT(CAST(p.media_urls AS JSON), '$[0]'))
            ELSE p.cover_url
        END
END
WHERE p.cover_url IS NULL OR TRIM(p.cover_url) = '';

ALTER TABLE inspiration_posts
    DROP COLUMN images,
    DROP COLUMN video_url;

DROP TABLE IF EXISTS inspiration_post_media;
