-- ==================== 数据库索引优化迁移脚本 ====================
-- 执行前请备份数据库
-- 建议在低峰期执行，大表索引创建可能需要较长时间

-- ==================== TransferEvent 表索引 ====================

-- 复合索引：entity + occurred_at（最常用查询）
CREATE INDEX IF NOT EXISTS idx_te_entity_occurred 
ON transfer_events(entity, occurred_at DESC);

-- 复合索引：chain + occurred_at
CREATE INDEX IF NOT EXISTS idx_te_chain_occurred 
ON transfer_events(chain, occurred_at DESC);

-- 复合索引：coin + occurred_at
CREATE INDEX IF NOT EXISTS idx_te_coin_occurred 
ON transfer_events(coin, occurred_at DESC);

-- 复合索引：entity + chain + occurred_at
CREATE INDEX IF NOT EXISTS idx_te_entity_chain_occurred 
ON transfer_events(entity, chain, occurred_at DESC);

-- 复合索引：entity + coin + occurred_at
CREATE INDEX IF NOT EXISTS idx_te_entity_coin_occurred 
ON transfer_events(entity, coin, occurred_at DESC);

-- 单列索引：created_at（用于实时推送）
CREATE INDEX IF NOT EXISTS idx_te_created_at 
ON transfer_events(created_at DESC);

-- 单列索引：tx_id（用于去重和查询）
CREATE INDEX IF NOT EXISTS idx_te_txid 
ON transfer_events(tx_id);

-- 复合索引：address + occurred_at（用于地址追踪）
CREATE INDEX IF NOT EXISTS idx_te_address_occurred 
ON transfer_events(address, occurred_at DESC);

-- ==================== PortfolioSnapshot 表索引 ====================

-- 复合索引：entity + created_at
CREATE INDEX IF NOT EXISTS idx_ps_entity_created 
ON portfolio_snapshots(entity, created_at DESC);

-- 单列索引：as_of（用于时间范围查询）
CREATE INDEX IF NOT EXISTS idx_ps_as_of 
ON portfolio_snapshots(as_of DESC);

-- ==================== Holding 表索引 ====================

-- 复合索引：entity + chain（用于按链查询持仓）
CREATE INDEX IF NOT EXISTS idx_h_entity_chain 
ON holdings(entity, chain);

-- 复合索引：run_id + entity（已存在唯一索引，但可以添加普通索引用于查询优化）
-- 注意：如果已有唯一索引，可以跳过

-- ==================== DailyFlow 表索引 ====================

-- 复合索引：entity + day
CREATE INDEX IF NOT EXISTS idx_df_entity_day 
ON daily_flows(entity, day DESC);

-- 复合索引：coin + day
CREATE INDEX IF NOT EXISTS idx_df_coin_day 
ON daily_flows(coin, day DESC);

-- 单列索引：day（用于跨实体查询）
CREATE INDEX IF NOT EXISTS idx_df_day 
ON daily_flows(day DESC);

-- ==================== WeeklyFlow 表索引 ====================

-- 复合索引：entity + week
CREATE INDEX IF NOT EXISTS idx_wf_entity_week 
ON weekly_flows(entity, week DESC);

-- 复合索引：coin + week
CREATE INDEX IF NOT EXISTS idx_wf_coin_week 
ON weekly_flows(coin, week DESC);

-- ==================== BinanceMarketTop 表索引 ====================

-- 复合索引：snapshot_id + rank（已存在，确认即可）
-- 复合索引：symbol + rank（用于按币种查询）
CREATE INDEX IF NOT EXISTS idx_bmt_symbol_rank 
ON binance_market_tops(symbol, rank);

-- ==================== Announcement 表索引 ====================

-- 复合索引：source + release_time
CREATE INDEX IF NOT EXISTS idx_ann_source_release 
ON announcements(source, release_time DESC);

-- 复合索引：category + release_time
CREATE INDEX IF NOT EXISTS idx_ann_category_release 
ON announcements(category, release_time DESC);

-- 单列索引：release_time（用于时间范围查询）
CREATE INDEX IF NOT EXISTS idx_ann_release 
ON announcements(release_time DESC);

-- 全文索引：title（如果 MySQL 版本支持，用于搜索优化）
-- CREATE FULLTEXT INDEX idx_ann_title_ft ON announcements(title);

-- ==================== TwitterPost 表索引 ====================

-- 复合索引：username + tweet_time（已存在，确认即可）
-- 单列索引：tweet_time（用于时间范围查询）
CREATE INDEX IF NOT EXISTS idx_tp_time 
ON twitter_posts(tweet_time DESC);

-- ==================== ScheduledOrder 表索引 ====================

-- 复合索引：user_id + status
CREATE INDEX IF NOT EXISTS idx_so_user_status 
ON scheduled_orders(user_id, status);

-- 复合索引：status + trigger_time（用于调度器查询）
CREATE INDEX IF NOT EXISTS idx_so_status_trigger 
ON scheduled_orders(status, trigger_time ASC);

-- 单列索引：trigger_time（用于时间范围查询）
CREATE INDEX IF NOT EXISTS idx_so_trigger 
ON scheduled_orders(trigger_time ASC);

-- ==================== 分析现有索引 ====================
-- 执行以下查询查看表的所有索引：
-- SHOW INDEX FROM transfer_events;
-- SHOW INDEX FROM portfolio_snapshots;
-- SHOW INDEX FROM holdings;
-- SHOW INDEX FROM daily_flows;
-- SHOW INDEX FROM weekly_flows;

-- ==================== 索引使用情况分析 ====================
-- 使用 EXPLAIN 分析查询计划：
-- EXPLAIN SELECT * FROM transfer_events WHERE entity = 'binance' AND occurred_at > '2025-01-01' ORDER BY occurred_at DESC LIMIT 50;

-- ==================== 索引维护 ====================
-- 定期分析表以更新索引统计信息：
-- ANALYZE TABLE transfer_events;
-- ANALYZE TABLE portfolio_snapshots;
-- ANALYZE TABLE daily_flows;

