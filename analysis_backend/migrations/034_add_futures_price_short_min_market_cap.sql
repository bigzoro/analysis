-- 添加合约涨幅开空策略的市值过滤字段
ALTER TABLE trading_strategies ADD COLUMN futures_price_short_min_market_cap DECIMAL(20,2) DEFAULT 0 COMMENT '合约涨幅开空策略最低市值要求（万），0表示无限制';