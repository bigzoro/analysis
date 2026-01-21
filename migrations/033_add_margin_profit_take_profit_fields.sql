-- 添加保证金盈利止盈字段到trading_strategies表
-- 支持基于保证金盈利百分比的智能止盈功能

-- 为trading_strategies表添加保证金盈利止盈相关字段
ALTER TABLE trading_strategies
    ADD COLUMN enable_margin_profit_take_profit TINYINT(1) DEFAULT 0 COMMENT '启用保证金盈利止盈',
    ADD COLUMN margin_profit_take_profit_percent DECIMAL(5,2) DEFAULT 100.00 COMMENT '保证金盈利止盈百分比';

-- 为现有记录设置默认值
UPDATE trading_strategies
SET margin_profit_take_profit_percent = 100.00
WHERE margin_profit_take_profit_percent IS NULL;

-- 添加索引以提高查询性能
CREATE INDEX idx_trading_strategies_margin_profit_take_profit
ON trading_strategies(enable_margin_profit_take_profit, margin_profit_take_profit_percent);

-- enable_margin_profit_take_profit: 是否启用基于保证金盈利的止盈机制
-- margin_profit_take_profit_percent: 当保证金盈利达到此百分比时触发止盈（例如100.00表示100%）

-- 回滚语句（如果需要撤销迁移）
-- ALTER TABLE trading_strategies
--     DROP COLUMN enable_margin_profit_take_profit,
--     DROP COLUMN margin_profit_take_profit_percent;
--
-- DROP INDEX idx_trading_strategies_margin_profit_take_profit ON trading_strategies;