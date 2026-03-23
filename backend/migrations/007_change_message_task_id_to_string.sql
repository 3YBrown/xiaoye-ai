-- 将 messages 表的 task_id 从 BIGINT 改为 VARCHAR(200)，用于存储视频服务商的字符串任务ID
ALTER TABLE messages MODIFY COLUMN task_id VARCHAR(200) COMMENT '关联的视频任务ID(服务商字符串ID)';
