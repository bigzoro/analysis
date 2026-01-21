-- 创建策略执行记录表
CREATE TABLE IF NOT EXISTS strategy_executions (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    strategy_id BIGINT UNSIGNED NOT NULL,
    user_id BIGINT UNSIGNED NOT NULL,
    status VARCHAR(32) DEFAULT 'pending',
    start_time DATETIME NOT NULL,
    end_time DATETIME NULL,
    duration INT DEFAULT 0,

    -- 执行结果统计
    total_orders INT DEFAULT 0,
    success_orders INT DEFAULT 0,
    failed_orders INT DEFAULT 0,
    total_pnl DECIMAL(20,8) DEFAULT 0,
    win_rate DECIMAL(5,2) DEFAULT 0,

    -- 执行过程跟踪
    current_step VARCHAR(64) DEFAULT '',
    step_progress INT DEFAULT 0,
    total_progress INT DEFAULT 0,
    current_symbol VARCHAR(32) DEFAULT '',
    error_message TEXT,

    -- 执行日志
    logs TEXT,

    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,

    INDEX idx_strategy_id (strategy_id),
    INDEX idx_user_id (user_id),
    INDEX idx_status (status),
    INDEX idx_created_at (created_at),
    FOREIGN KEY (strategy_id) REFERENCES trading_strategies(id) ON DELETE CASCADE,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
