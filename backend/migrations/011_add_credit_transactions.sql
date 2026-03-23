-- 011: 新增用户钻石流水表（统一记录加减钻）
CREATE TABLE IF NOT EXISTS `credit_transactions` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT COMMENT 'Transaction ID',
  `user_id` bigint unsigned NOT NULL COMMENT 'User ID',
  `delta` int NOT NULL COMMENT 'Credits delta (positive/negative)',
  `balance_after` int NOT NULL DEFAULT 0 COMMENT 'Balance after this transaction',
  `type` varchar(40) NOT NULL COMMENT 'Transaction type',
  `source` varchar(40) DEFAULT NULL COMMENT 'Business source',
  `source_id` varchar(100) DEFAULT NULL COMMENT 'Business source ID',
  `note` varchar(255) DEFAULT NULL COMMENT 'Note',
  `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT 'Creation time',
  PRIMARY KEY (`id`),
  KEY `idx_credit_transactions_user_id` (`user_id`),
  KEY `idx_credit_transactions_type` (`type`),
  KEY `idx_credit_transactions_source` (`source`),
  KEY `idx_credit_transactions_source_id` (`source_id`),
  KEY `idx_credit_transactions_created_at` (`created_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='用户钻石流水表';
