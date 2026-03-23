ALTER TABLE users ADD COLUMN checkin_streak INT NOT NULL DEFAULT 0 COMMENT 'Consecutive checkin days';
ALTER TABLE users ADD COLUMN last_checkin_date DATE NULL COMMENT 'Last checkin date';
