-- 添加订单关联字段到scheduled_orders表
-- 用于关联开仓订单和平仓订单
-- +migrate Up
ALTER TABLE scheduled_orders
ADD COLUMN parent_order_id INT DEFAULT 0,
ADD COLUMN close_order_ids TEXT;

-- 添加索引
CREATE INDEX idx_scheduled_orders_parent_order_id ON scheduled_orders(parent_order_id);

-- +migrate Down
DROP INDEX IF EXISTS idx_scheduled_orders_parent_order_id;
ALTER TABLE scheduled_orders
DROP COLUMN close_order_ids,
DROP COLUMN parent_order_id;
