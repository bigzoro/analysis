-- 优化market_klines表的索引以提升查询性能
-- 解决慢SQL问题：SELECT * FROM market_klines WHERE symbol = ? AND kind = ? AND interval = ? ORDER BY open_time DESC LIMIT 1

-- 删除旧的索引（如果存在）
DROP INDEX IF EXISTS idx_symbol_time ON market_klines;
DROP INDEX IF EXISTS idx_kind_interval_time ON market_klines;

-- 创建优化的复合索引，包含查询条件的所有字段
-- 索引顺序：symbol, kind, interval, open_time 能够覆盖 WHERE + ORDER BY 的查询模式
CREATE INDEX idx_symbol_kind_interval_time ON market_klines (symbol, kind, interval, open_time DESC);

-- 保留一个只包含时间字段的索引，用于其他查询模式
CREATE INDEX idx_open_time ON market_klines (open_time DESC);
