-- 添加实际使用的TP/SL百分比字段到scheduled_orders表
-- +migrate Up
ALTER TABLE scheduled_orders
ADD COLUMN actual_tp_percent DECIMAL(10,4) DEFAULT NULL COMMENT '实际使用的止盈百分比（自动调整后的）',
ADD COLUMN actual_sl_percent DECIMAL(10,4) DEFAULT NULL COMMENT '实际使用的止损百分比（自动调整后的）';

-- +migrate Down
ALTER TABLE scheduled_orders
DROP COLUMN actual_tp_percent,
DROP COLUMN actual_sl_percent;






