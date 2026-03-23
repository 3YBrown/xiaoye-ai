-- 006_add_conversations.sql
-- 新增会话和消息表，支持对话式 UI 的历史持久化

-- 会话表
CREATE TABLE IF NOT EXISTS conversations (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY COMMENT '会话ID',
    user_id BIGINT UNSIGNED NOT NULL COMMENT '用户ID',
    title VARCHAR(200) DEFAULT '新对话' COMMENT '会话标题',
    type VARCHAR(20) DEFAULT 'image' COMMENT '会话类型(image/video)',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    INDEX idx_conversations_user_id (user_id),
    INDEX idx_conversations_type (type),
    INDEX idx_conversations_created_at (created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='会话表';

-- 消息表
CREATE TABLE IF NOT EXISTS messages (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY COMMENT '消息ID',
    conversation_id BIGINT UNSIGNED NOT NULL COMMENT '所属会话ID',
    role VARCHAR(20) NOT NULL COMMENT '角色(user/assistant)',
    content LONGTEXT COMMENT '用户消息文本',
    reference_images TEXT COMMENT '用户上传的参考图(JSON数组)',
    params TEXT COMMENT '生成参数(JSON)',
    images TEXT COMMENT '助手返回的图片URL(JSON数组)',
    video_url VARCHAR(500) COMMENT '助手返回的视频URL',
    video_cover_url VARCHAR(500) COMMENT '视频封面URL',
    status VARCHAR(20) DEFAULT 'success' COMMENT '消息状态(generating/success/error)',
    credits_cost INT DEFAULT 0 COMMENT '消耗钻石数',
    error_msg TEXT COMMENT '错误信息',
    task_id BIGINT UNSIGNED COMMENT '关联的视频任务ID',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    INDEX idx_messages_conversation_id (conversation_id),
    INDEX idx_messages_created_at (created_at),
    CONSTRAINT fk_messages_conversation FOREIGN KEY (conversation_id) REFERENCES conversations(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='消息表';
