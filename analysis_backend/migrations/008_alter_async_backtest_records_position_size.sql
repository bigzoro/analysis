-- 修改 async_backtest_records 表的 position_size 列精度
ALTER TABLE async_backtest_records MODIFY COLUMN position_size DECIMAL(8,2) NOT NULL COMMENT '仓位大小（百分比）';
