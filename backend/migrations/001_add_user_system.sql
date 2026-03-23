-- =============================================
-- 数据库初始化脚本
-- 版本: 001
-- 描述: 完整建表脚本（用户体系 + 图片生成记录）
-- 执行前请先备份数据库
-- =============================================

-- 1. 创建用户表
CREATE TABLE IF NOT EXISTS `users` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT COMMENT 'User ID',
  `email` varchar(255) NOT NULL COMMENT 'Email address',
  `password_hash` varchar(255) DEFAULT NULL COMMENT 'Password hash',
  `nickname` varchar(100) DEFAULT NULL COMMENT 'User nickname',
  `avatar` varchar(500) DEFAULT NULL COMMENT 'Avatar URL',
  `credits` int NOT NULL DEFAULT 0 COMMENT 'Available credits',
  `total_redeemed` int NOT NULL DEFAULT 0 COMMENT 'Total redeemed credits',
  `usage_count` int NOT NULL DEFAULT 0 COMMENT 'Total usage count',
  `status` varchar(20) NOT NULL DEFAULT 'active' COMMENT 'User status (active/disabled)',
  `email_verified` tinyint(1) NOT NULL DEFAULT 0 COMMENT 'Email verified flag',
  `last_login_at` datetime DEFAULT NULL COMMENT 'Last login time',
  `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT 'Creation time',
  `updated_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT 'Last update time',
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_users_email` (`email`),
  KEY `idx_users_status` (`status`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='用户表';

-- 2. 创建邮箱验证码表
CREATE TABLE IF NOT EXISTS `email_verifications` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT COMMENT 'Verification ID',
  `email` varchar(255) NOT NULL COMMENT 'Email address',
  `code` varchar(10) NOT NULL COMMENT 'Verification code',
  `type` varchar(20) NOT NULL COMMENT 'Type: register/login/reset',
  `expires_at` datetime NOT NULL COMMENT 'Expiration time',
  `used` tinyint(1) NOT NULL DEFAULT 0 COMMENT 'Whether used',
  `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT 'Creation time',
  PRIMARY KEY (`id`),
  KEY `idx_email_verifications_email` (`email`),
  KEY `idx_email_verifications_expires` (`expires_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='邮箱验证码表';

-- 3. 创建钻石兑换历史表
CREATE TABLE IF NOT EXISTS `redeem_histories` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT COMMENT 'Redeem ID',
  `user_id` bigint unsigned NOT NULL COMMENT 'User ID',
  `license_id` varchar(36) NOT NULL COMMENT 'License ID',
  `credits` int NOT NULL COMMENT 'Redeemed credits',
  `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT 'Creation time',
  PRIMARY KEY (`id`),
  KEY `idx_redeem_histories_user_id` (`user_id`),
  KEY `idx_redeem_histories_license_id` (`license_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='钻石兑换历史表';

-- 4. 创建图片生成记录表（统一格式，支持单图/多图）
-- input_images / output_images 使用 JSON 数组格式: ["url1", "url2", ...]
CREATE TABLE IF NOT EXISTS `image_records` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT COMMENT 'Record ID',
  `license_id` varchar(255) DEFAULT NULL COMMENT 'License ID / User ID',
  `prompt` longtext COMMENT 'Image generation prompt',
  `mode` varchar(50) DEFAULT NULL COMMENT 'Mode (text-to-image/image-editing/multi-image-group)',
  `model_id` varchar(100) DEFAULT NULL COMMENT 'Model ID used',
  `image_size` varchar(50) DEFAULT NULL COMMENT 'Image size (1K, 2K, 4K)',
  `input_images` text COMMENT 'Input image URLs (JSON array)',
  `output_images` text COMMENT 'Output image URLs (JSON array)',
  `output_count` int DEFAULT 1 COMMENT 'Number of output images',
  `credits_spent` int DEFAULT NULL COMMENT 'Credits spent',
  `status` varchar(50) DEFAULT NULL COMMENT 'Status (success/failed/pending/processing)',
  `error_message` text COMMENT 'Error message if failed',
  `created_at` datetime DEFAULT NULL COMMENT 'Creation time',
  PRIMARY KEY (`id`),
  KEY `idx_image_records_license_id` (`license_id`),
  KEY `idx_image_records_mode` (`mode`),
  KEY `idx_image_records_model_id` (`model_id`),
  KEY `idx_image_records_created_at` (`created_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='图片生成记录表';

-- 5. 创建图片模板表
CREATE TABLE IF NOT EXISTS `image_templates` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT COMMENT 'Template ID',
  `name` varchar(100) DEFAULT NULL COMMENT 'Template name',
  `icon` varchar(10) DEFAULT NULL COMMENT 'Template icon',
  `description` varchar(255) DEFAULT NULL COMMENT 'Template description',
  `prompt` longtext COMMENT 'Template prompt',
  `preview_image` longtext COMMENT 'Preview image URL (after editing)',
  `before_image` longtext COMMENT 'Before image URL (original image)',
  `is_active` tinyint(1) DEFAULT 1 COMMENT 'Is template active',
  `sort_order` int DEFAULT 0 COMMENT 'Sort order',
  `created_at` datetime DEFAULT NULL COMMENT 'Creation time',
  `updated_at` datetime DEFAULT NULL COMMENT 'Update time',
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='图片模板表';

-- 6. 创建 License 表
CREATE TABLE IF NOT EXISTS `licenses` (
  `id` varchar(36) NOT NULL COMMENT 'License ID',
  `balance` bigint DEFAULT 0 NULL COMMENT 'Remaining credits',
  `status` varchar(20) DEFAULT 'active' NULL COMMENT 'License status (active/disabled/redeemed)',
  `expires_at` datetime NULL COMMENT 'Expiration time',
  `usage_count` bigint DEFAULT 0 NULL COMMENT 'Total usage count',
  `created_at` datetime NULL COMMENT 'Creation time',
  `updated_at` datetime NULL COMMENT 'Last update time',
  `redeemed_by` bigint unsigned NULL COMMENT 'User ID who redeemed this license',
  `redeemed_at` datetime NULL COMMENT 'Redemption time',
  `original_key` text NULL COMMENT 'Original license key (for reference)',
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='License密钥表';

CREATE INDEX `idx_licenses_expires_at`
    ON `licenses` (`expires_at`);

CREATE INDEX `idx_licenses_redeemed_by`
    ON `licenses` (`redeemed_by`);

CREATE INDEX `idx_licenses_status`
    ON `licenses` (`status`);

-- 7. 创建 API 日志表
CREATE TABLE IF NOT EXISTS `api_logs` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT COMMENT 'Log ID',
  `license_id` varchar(255) DEFAULT NULL COMMENT 'License ID',
  `endpoint` varchar(255) DEFAULT NULL COMMENT 'API endpoint',
  `request_body` longtext COMMENT 'Request body (JSON)',
  `response_body` longtext COMMENT 'Response body (JSON)',
  `response_code` int DEFAULT NULL COMMENT 'HTTP response code',
  `duration_ms` int DEFAULT NULL COMMENT 'Request duration in milliseconds',
  `created_at` datetime DEFAULT NULL COMMENT 'Creation time',
  PRIMARY KEY (`id`),
  KEY `idx_api_logs_license_id` (`license_id`),
  KEY `idx_api_logs_created_at` (`created_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='API日志表';

-- =============================================
-- 验证脚本执行结果
-- =============================================

-- SHOW TABLES;
-- DESCRIBE image_records;
-- DESCRIBE users;
