-- 添加策略执行的盈亏统计字段
ALTER TABLE strategy_executions ADD COLUMN pnl_percentage DECIMAL(8,4) DEFAULT 0 COMMENT '盈亏百分比';
ALTER TABLE strategy_executions ADD COLUMN total_investment DECIMAL(20,8) DEFAULT 0 COMMENT '买入总金额';
ALTER TABLE strategy_executions ADD COLUMN current_value DECIMAL(20,8) DEFAULT 0 COMMENT '当前资产价值';

-- 添加索引优化查询性能
CREATE INDEX idx_strategy_executions_pnl_percentage ON strategy_executions(pnl_percentage);
CREATE INDEX idx_strategy_executions_total_investment ON strategy_executions(total_investment);
CREATE INDEX idx_strategy_executions_current_value ON strategy_executions(current_value);
