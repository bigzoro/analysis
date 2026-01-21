-- 添加策略执行步骤跟踪表
CREATE TABLE IF NOT EXISTS strategy_execution_steps (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    execution_id BIGINT UNSIGNED NOT NULL,
    step_name VARCHAR(64) NOT NULL,
    step_type VARCHAR(32) NOT NULL,
    symbol VARCHAR(32),
    status VARCHAR(32) DEFAULT 'pending',
    start_time DATETIME NULL DEFAULT CURRENT_TIMESTAMP,
    end_time DATETIME NULL,
    duration INT DEFAULT 0,
    result TEXT,
    error_message TEXT,
    data TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    INDEX idx_execution_id (execution_id),
    INDEX idx_status (status),
    INDEX idx_symbol (symbol),
    FOREIGN KEY (execution_id) REFERENCES strategy_executions(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 为策略执行表添加新的字段
ALTER TABLE strategy_executions
ADD COLUMN IF NOT EXISTS current_step VARCHAR(64) DEFAULT '',
ADD COLUMN IF NOT EXISTS step_progress INT DEFAULT 0,
ADD COLUMN IF NOT EXISTS total_progress INT DEFAULT 0,
ADD COLUMN IF NOT EXISTS current_symbol VARCHAR(32) DEFAULT '',
ADD COLUMN IF NOT EXISTS error_message TEXT;

-- 为定时订单表添加执行ID字段
ALTER TABLE scheduled_orders
ADD COLUMN IF NOT EXISTS execution_id BIGINT UNSIGNED,
ADD INDEX IF NOT EXISTS idx_execution_id (execution_id);
