-- 检查market_klines表的索引状态
SELECT
    TABLE_NAME,
    INDEX_NAME,
    COLUMN_NAME,
    SEQ_IN_INDEX,
    CARDINALITY,
    INDEX_TYPE,
    INDEX_COMMENT
FROM information_schema.statistics
WHERE table_schema = DATABASE()
    AND table_name = 'market_klines'
ORDER BY INDEX_NAME, SEQ_IN_INDEX;

-- 检查表的基本信息
DESCRIBE market_klines;

-- 检查数据量
SELECT
    COUNT(*) as total_rows,
    MIN(open_time) as min_time,
    MAX(open_time) as max_time,
    COUNT(DISTINCT symbol) as unique_symbols
FROM market_klines;

-- 检查查询计划（如果支持）
-- EXPLAIN SELECT symbol, close_price as close, open_time as time
-- FROM market_klines
-- WHERE open_time >= '2025-12-23 06:53:33' AND open_time <= '2025-12-30 06:53:33'
--     AND symbol IN ('BTCUSDT','ETHUSDT')
-- ORDER BY open_time ASC
-- LIMIT 10;

-- EXPLAIN SELECT symbol, close_price FROM market_klines
-- WHERE symbol IN ('AEURUSDT','USDPUSDT')
--     AND open_time >= '2025-12-23 06:53:33' AND open_time <= '2025-12-30 06:53:33'
-- ORDER BY symbol, open_time ASC
-- LIMIT 10;