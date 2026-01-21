-- 添加订单跟踪字段到scheduled_orders表
-- +migrate Up
ALTER TABLE scheduled_orders
ADD COLUMN client_order_id VARCHAR(64) DEFAULT '',
ADD COLUMN exchange_order_id VARCHAR(64) DEFAULT '',
ADD COLUMN executed_quantity VARCHAR(64) DEFAULT '',
ADD COLUMN avg_price VARCHAR(64) DEFAULT '';

-- 添加索引以提高查询性能
CREATE INDEX idx_scheduled_orders_client_order_id ON scheduled_orders(client_order_id);
CREATE INDEX idx_scheduled_orders_exchange_order_id ON scheduled_orders(exchange_order_id);

-- +migrate Down
DROP INDEX IF EXISTS idx_scheduled_orders_exchange_order_id;
DROP INDEX IF EXISTS idx_scheduled_orders_client_order_id;
ALTER TABLE scheduled_orders
DROP COLUMN avg_price,
DROP COLUMN executed_quantity,
DROP COLUMN exchange_order_id,
DROP COLUMN client_order_id;
