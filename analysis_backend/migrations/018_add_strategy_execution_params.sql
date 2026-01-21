-- 为策略执行表添加启动参数字段
ALTER TABLE strategy_executions ADD COLUMN run_interval INT DEFAULT 60;
