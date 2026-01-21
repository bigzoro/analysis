-- 数据库索引优化脚本
-- 用于提升实时涨幅榜系统的查询性能

-- 1. 实时涨幅榜快照表索引
-- 用于快速查找最新快照
CREATE INDEX IF NOT EXISTS idx_realtime_gainers_snapshots_lookup
ON realtime_gainers_snapshots (kind, timestamp DESC, id DESC);

-- 2. 涨幅榜数据项表索引
-- 用于根据快照ID快速查询数据项
CREATE INDEX IF NOT EXISTS idx_realtime_gainers_items_snapshot
ON realtime_gainers_items (snapshot_id, rank ASC);

-- 3. Binance 24h统计表索引
-- 用于涨幅榜查询和筛选
CREATE INDEX IF NOT EXISTS idx_binance_24h_stats_gainers
ON binance_24h_stats (market_type, created_at DESC, price_change_percent DESC, volume DESC);

-- 添加复合索引用于复杂查询
CREATE INDEX IF NOT EXISTS idx_binance_24h_stats_market_time
ON binance_24h_stats (market_type, created_at DESC);

-- 4. K线数据表索引
-- 用于基准价格查询
CREATE INDEX IF NOT EXISTS idx_market_klines_base_price
ON market_klines (symbol, `interval`, open_time DESC);

-- 5. 现有表索引检查和优化
-- 检查是否有重复索引，可以删除
-- SHOW INDEX FROM realtime_gainers_snapshots;
-- SHOW INDEX FROM realtime_gainers_items;
-- SHOW INDEX FROM binance_24h_stats;
-- SHOW INDEX FROM market_klines;

-- 6. 索引维护建议
-- 定期运行以下命令维护索引：
-- ANALYZE TABLE realtime_gainers_snapshots;
-- ANALYZE TABLE realtime_gainers_items;
-- ANALYZE TABLE binance_24h_stats;
-- ANALYZE TABLE market_klines;

-- 7. 查询性能监控
-- 可以使用以下查询监控慢查询：
-- SELECT sql_text, exec_count, avg_timer_wait/1000000000 avg_time_sec
-- FROM performance_schema.events_statements_summary_by_digest
-- WHERE sql_text LIKE '%realtime_gainers%' OR sql_text LIKE '%binance_24h_stats%'
-- ORDER BY avg_timer_wait DESC LIMIT 10;