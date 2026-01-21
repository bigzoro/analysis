-- 创建异步回测交易记录表
CREATE TABLE IF NOT EXISTS async_backtest_trades (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    backtest_record_id BIGINT UNSIGNED NOT NULL,
    timestamp DATETIME NOT NULL,
    symbol VARCHAR(20) NOT NULL,
    side VARCHAR(10) NOT NULL COMMENT 'buy/sell',
    price DECIMAL(20,8) NOT NULL,
    quantity DECIMAL(20,8) NOT NULL,
    value DECIMAL(20,8) NOT NULL COMMENT '成交金额',
    commission DECIMAL(20,8) DEFAULT 0 COMMENT '手续费',
    pnl DECIMAL(20,8) DEFAULT 0 COMMENT '盈亏',
    pnl_percent DECIMAL(20,8) DEFAULT 0 COMMENT '盈亏百分比',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,

    INDEX idx_backtest_record (backtest_record_id),
    INDEX idx_timestamp (timestamp),
    INDEX idx_symbol (symbol),
    INDEX idx_side (side),

    FOREIGN KEY (backtest_record_id) REFERENCES async_backtest_records(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='异步回测交易记录表';
