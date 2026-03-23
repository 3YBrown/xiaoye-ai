-- 为 messages.task_id 添加索引，优化 syncVideoStatusToMessage 查询性能
ALTER TABLE messages ADD INDEX idx_messages_task_id (task_id);
