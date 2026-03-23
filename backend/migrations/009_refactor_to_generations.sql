-- 009_refactor_to_generations.sql
-- 重构：从多对话模式改为单一生成历史模式
-- 移除 conversations 概念，直接关联 user_id

-- 1. 创建新的 generations 表
CREATE TABLE IF NOT EXISTS generations (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY COMMENT '生成记录ID',
    user_id BIGINT UNSIGNED NOT NULL COMMENT '用户ID',
    type VARCHAR(20) NOT NULL DEFAULT 'image' COMMENT '生成类型(image/video)',

    -- 用户输入
    prompt LONGTEXT NOT NULL COMMENT '用户提示词',
    reference_images TEXT COMMENT '参考图片URL(JSON数组)',
    params TEXT COMMENT '生成参数(JSON: model, aspectRatio, resolution等)',

    -- 生成结果
    images TEXT COMMENT '生成的图片URL(JSON数组)',
    video_url VARCHAR(500) COMMENT '生成的视频URL',
    video_cover_url VARCHAR(500) COMMENT '视频封面URL',

    -- 状态和元信息
    status VARCHAR(20) DEFAULT 'success' COMMENT '状态(generating/queued/running/success/failed)',
    credits_cost INT DEFAULT 0 COMMENT '消耗钻石数',
    error_msg TEXT COMMENT '错误信息',
    task_id VARCHAR(100) COMMENT '关联的任务ID（用于轮询）',

    -- 用户标记
    is_favorite TINYINT(1) DEFAULT 0 COMMENT '是否收藏',
    tags VARCHAR(500) COMMENT '标签(逗号分隔)',

    -- 时间戳
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',

    -- 索引
    INDEX idx_generations_user_id (user_id),
    INDEX idx_generations_type (type),
    INDEX idx_generations_status (status),
    INDEX idx_generations_is_favorite (is_favorite),
    INDEX idx_generations_created_at (created_at),
    INDEX idx_generations_task_id (task_id),

    -- 外键
    CONSTRAINT fk_generations_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='生成历史记录表';

-- 2. 迁移数据：将 messages 表中的助手消息迁移到 generations
-- 只迁移有实际生成结果的消息（有 images 或 video_url 的助手消息）
INSERT INTO generations (
    user_id,
    type,
    prompt,
    reference_images,
    params,
    images,
    video_url,
    video_cover_url,
    status,
    credits_cost,
    error_msg,
    task_id,
    created_at,
    updated_at
)
SELECT
    c.user_id,
    c.type,
    -- 获取对应的用户消息的提示词
    (SELECT m2.content
     FROM messages m2
     WHERE m2.conversation_id = m.conversation_id
       AND m2.role = 'user'
       AND m2.created_at < m.created_at
     ORDER BY m2.created_at DESC
     LIMIT 1) as prompt,
    -- 获取对应的用户消息的参考图片
    (SELECT m2.reference_images
     FROM messages m2
     WHERE m2.conversation_id = m.conversation_id
       AND m2.role = 'user'
       AND m2.created_at < m.created_at
     ORDER BY m2.created_at DESC
     LIMIT 1) as reference_images,
    -- 获取对应的用户消息的参数
    (SELECT m2.params
     FROM messages m2
     WHERE m2.conversation_id = m.conversation_id
       AND m2.role = 'user'
       AND m2.created_at < m.created_at
     ORDER BY m2.created_at DESC
     LIMIT 1) as params,
    m.images,
    m.video_url,
    m.video_cover_url,
    m.status,
    m.credits_cost,
    m.error_msg,
    m.task_id,
    m.created_at,
    m.created_at as updated_at
FROM messages m
JOIN conversations c ON m.conversation_id = c.id
WHERE m.role = 'assistant'
  AND (m.images IS NOT NULL OR m.video_url IS NOT NULL OR m.status IN ('generating', 'queued', 'running'));

-- 3. 删除旧表（先删除有外键的 messages，再删除 conversations）
DROP TABLE IF EXISTS messages;
DROP TABLE IF EXISTS conversations;

-- 迁移完成
