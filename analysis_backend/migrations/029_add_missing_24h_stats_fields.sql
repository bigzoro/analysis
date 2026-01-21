-- 添加缺失的24小时统计数据字段到binance_24h_stats表
-- 完善Binance API数据的完整性，包括交易深度和交易ID信息
-- +migrate Up
ALTER TABLE binance_24h_stats
ADD COLUMN last_qty DECIMAL(20,8) DEFAULT 0 COMMENT '最后交易数量',
ADD COLUMN bid_qty DECIMAL(20,8) DEFAULT 0 COMMENT '买一档数量',
ADD COLUMN ask_qty DECIMAL(20,8) DEFAULT 0 COMMENT '卖一档数量',
ADD COLUMN first_id BIGINT DEFAULT 0 COMMENT '第一笔交易ID',
ADD COLUMN last_id BIGINT DEFAULT 0 COMMENT '最后一笔交易ID';

-- 添加索引优化查询性能
CREATE INDEX idx_binance_24h_stats_first_id ON binance_24h_stats(first_id);
CREATE INDEX idx_binance_24h_stats_last_id ON binance_24h_stats(last_id);

-- +migrate Down
DROP INDEX IF EXISTS idx_binance_24h_stats_last_id;
DROP INDEX IF EXISTS idx_binance_24h_stats_first_id;
ALTER TABLE binance_24h_stats
DROP COLUMN last_id,
DROP COLUMN first_id,
DROP COLUMN ask_qty,
DROP COLUMN bid_qty,
DROP COLUMN last_qty;