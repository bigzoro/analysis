-- 为 async_backtest_records 表的 created_at 字段添加索引
ALTER TABLE async_backtest_records ADD INDEX idx_created_at (created_at);
