-- 添加网格层级字段到scheduled_orders表
-- 用于网格交易策略跟踪订单的网格位置
-- +migrate Up
ALTER TABLE scheduled_orders
ADD COLUMN grid_level INT DEFAULT 0,
ADD COLUMN strategy_type VARCHAR(32) DEFAULT '';

-- 添加索引
CREATE INDEX idx_scheduled_orders_strategy_type ON scheduled_orders(strategy_type);
CREATE INDEX idx_scheduled_orders_grid_level ON scheduled_orders(grid_level);

-- +migrate Down
DROP INDEX IF EXISTS idx_scheduled_orders_grid_level;
DROP INDEX IF EXISTS idx_scheduled_orders_strategy_type;
ALTER TABLE scheduled_orders
DROP COLUMN strategy_type,
DROP COLUMN grid_level;