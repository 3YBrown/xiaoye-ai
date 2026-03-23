-- 修复 provider_response 字段类型
-- 原因：Go string 类型与 MySQL JSON 类型不兼容，空字符串会导致插入失败
-- 解决：将 JSON 改为 LONGTEXT，与 Go 结构体定义保持一致

ALTER TABLE video_tasks MODIFY COLUMN provider_response LONGTEXT;
