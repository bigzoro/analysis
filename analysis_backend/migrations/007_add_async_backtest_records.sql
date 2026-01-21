-- 创建异步回测记录表
CREATE TABLE IF NOT EXISTS async_backtest_records (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    user_id BIGINT UNSIGNED NOT NULL COMMENT '用户ID',
    symbol VARCHAR(32) NOT NULL COMMENT '交易币种',
    strategy VARCHAR(32) NOT NULL COMMENT '策略名称',
    start_date DATE NOT NULL COMMENT '回测开始日期',
    end_date DATE NOT NULL COMMENT '回测结束日期',
    initial_capital DECIMAL(20,8) NOT NULL COMMENT '初始资金',
    position_size DECIMAL(5,4) NOT NULL COMMENT '仓位大小',
    status ENUM('pending', 'running', 'completed', 'failed') DEFAULT 'pending' COMMENT '执行状态',
    result JSON COMMENT '回测结果JSON',
    error_message TEXT COMMENT '错误信息',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    completed_at TIMESTAMP NULL COMMENT '完成时间',

    INDEX idx_user_id (user_id),
    INDEX idx_symbol (symbol),
    INDEX idx_status (status),
    INDEX idx_created_at (created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='异步回测记录表';
