-- 为market_klines表创建关键索引以优化慢查询
-- 这个脚本应该在数据库中执行

-- 索引1: 优化时间范围+币种查询 (第一个慢查询)
-- WHERE open_time >= ? AND open_time <= ? AND symbol IN (...) ORDER BY open_time ASC
CREATE INDEX IF NOT EXISTS idx_open_time_symbol ON market_klines (open_time, symbol);

-- 索引2: 优化币种+时间范围查询 (第二个慢查询)
-- WHERE symbol IN (...) AND open_time >= ? AND open_time <= ? ORDER BY symbol, open_time ASC
CREATE INDEX IF NOT EXISTS idx_symbol_open_time ON market_klines (symbol, open_time);

-- 检查索引是否创建成功
SELECT
    TABLE_NAME,
    INDEX_NAME,
    COLUMN_NAME,
    SEQ_IN_INDEX,
    CARDINALITY,
    INDEX_TYPE
FROM information_schema.statistics
WHERE table_schema = DATABASE()
    AND table_name = 'market_klines'
    AND index_name IN ('idx_open_time_symbol', 'idx_symbol_open_time')
ORDER BY INDEX_NAME, SEQ_IN_INDEX;

-- 显示表统计信息
SELECT
    COUNT(*) as total_rows,
    COUNT(DISTINCT symbol) as unique_symbols,
    MIN(open_time) as min_time,
    MAX(open_time) as max_time
FROM market_klines;