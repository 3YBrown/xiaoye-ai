-- 010: 删除已废弃的 video_tasks 表（数据已迁移至 generations，Worker 已改为读写 generations）
DROP TABLE IF EXISTS `video_tasks`;
