-- =============================================
-- Migration: 003_rename_api_logs_license_id
-- Description: 将 api_logs 表中的 license_id 重命名为 user_id
-- Date: 2026-01-18
-- =============================================

-- 1. 重命名列
ALTER TABLE `api_logs` CHANGE COLUMN `license_id` `user_id` varchar(255) DEFAULT NULL COMMENT 'User ID';

-- 2. 删除旧索引
DROP INDEX `idx_api_logs_license_id` ON `api_logs`;

-- 3. 创建新索引
CREATE INDEX `idx_api_logs_user_id` ON `api_logs` (`user_id`);
