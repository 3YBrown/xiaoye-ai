-- =============================================
-- 邀请系统迁移脚本
-- 版本: 002
-- 描述: 添加邀请注册功能
-- 执行前请先备份数据库
-- =============================================

-- 1. 在用户表中添加邀请相关字段
ALTER TABLE `users`
ADD COLUMN `invite_code` varchar(20) DEFAULT NULL COMMENT 'User unique invite code' AFTER `email_verified`,
ADD COLUMN `invited_by` bigint unsigned DEFAULT NULL COMMENT 'User ID who invited this user' AFTER `invite_code`,
ADD COLUMN `invite_count` int NOT NULL DEFAULT 0 COMMENT 'Number of users invited' AFTER `invited_by`;

-- 添加邀请码唯一索引
ALTER TABLE `users`
ADD UNIQUE KEY `idx_users_invite_code` (`invite_code`);

-- 添加邀请人索引
ALTER TABLE `users`
ADD KEY `idx_users_invited_by` (`invited_by`);

-- 2. 创建邀请记录表
CREATE TABLE IF NOT EXISTS `invitation_records` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT COMMENT 'Record ID',
  `inviter_id` bigint unsigned NOT NULL COMMENT 'Inviter user ID',
  `invitee_id` bigint unsigned NOT NULL COMMENT 'Invitee user ID',
  `invitee_email` varchar(255) NOT NULL COMMENT 'Invitee email',
  `credits_rewarded` int NOT NULL DEFAULT 10 COMMENT 'Credits rewarded to inviter',
  `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT 'Invitation time',
  PRIMARY KEY (`id`),
  KEY `idx_invitation_records_inviter_id` (`inviter_id`),
  KEY `idx_invitation_records_invitee_id` (`invitee_id`),
  KEY `idx_invitation_records_created_at` (`created_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='邀请记录表';

-- 3. 为现有用户生成邀请码
-- 使用随机字符串生成邀请码（可选，在应用层处理更灵活）
-- UPDATE users SET invite_code = CONCAT(SUBSTRING(MD5(RAND()), 1, 8)) WHERE invite_code IS NULL;

-- =============================================
-- 验证脚本执行结果
-- =============================================

-- DESCRIBE users;
-- SHOW CREATE TABLE invitation_records;
