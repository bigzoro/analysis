-- 将字段名从 total_pn_l 改为 total_pnl
ALTER TABLE strategy_executions
CHANGE COLUMN total_pn_l total_pnl DECIMAL(20,8) DEFAULT 0;
