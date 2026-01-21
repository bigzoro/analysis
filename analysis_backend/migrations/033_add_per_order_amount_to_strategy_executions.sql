-- 添加策略执行的每一单金额字段
ALTER TABLE strategy_executions ADD COLUMN per_order_amount DECIMAL(20,8) DEFAULT 0 COMMENT '每一单的金额（U单位），0表示使用默认金额';