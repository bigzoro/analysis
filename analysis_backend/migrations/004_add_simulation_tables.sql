-- 创建模拟交易表
CREATE TABLE IF NOT EXISTS simulated_trades (
    id BIGINT UNSIGNED PRIMARY KEY AUTO_INCREMENT,
    user_id INT UNSIGNED NOT NULL COMMENT '用户ID',
    symbol VARCHAR(32) NOT NULL COMMENT '交易对，如BTCUSDT',
    base_symbol VARCHAR(16) NOT NULL COMMENT '基础币种，如BTC',
    kind VARCHAR(16) NOT NULL COMMENT 'spot/futures',
    quantity DECIMAL(20,8) NOT NULL COMMENT '买入数量',
    entry_price DECIMAL(20,8) NOT NULL COMMENT '买入价格',
    entry_time DATETIME NOT NULL COMMENT '买入时间',
    current_price DECIMAL(20,8) COMMENT '当前价格',
    current_value DECIMAL(20,8) COMMENT '当前价值',
    pnl DECIMAL(20,8) COMMENT '盈亏',
    pnl_percent DECIMAL(10,4) COMMENT '盈亏百分比',
    status VARCHAR(16) DEFAULT 'open' COMMENT 'open/closed',
    closed_at DATETIME COMMENT '平仓时间',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    
    INDEX idx_user_symbol (user_id, symbol),
    INDEX idx_entry_time (entry_time),
    INDEX idx_status (status)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='模拟交易持仓表';

-- 创建推荐回测表
CREATE TABLE IF NOT EXISTS recommendation_backtests (
    id BIGINT UNSIGNED PRIMARY KEY AUTO_INCREMENT,
    recommendation_id BIGINT UNSIGNED COMMENT '关联推荐ID',
    symbol VARCHAR(32) NOT NULL COMMENT '交易对',
    base_symbol VARCHAR(16) NOT NULL COMMENT '基础币种',
    recommended_at DATETIME NOT NULL COMMENT '推荐时间',
    recommended_price DECIMAL(20,8) NOT NULL COMMENT '推荐时价格',
    
    -- 回测时间点价格
    price_after_1h DECIMAL(20,8) COMMENT '1小时后价格',
    price_after_4h DECIMAL(20,8) COMMENT '4小时后价格',
    price_after_24h DECIMAL(20,8) COMMENT '24小时后价格',
    price_after_7d DECIMAL(20,8) COMMENT '7天后价格',
    
    -- 收益率
    return_1h DECIMAL(10,4) COMMENT '1小时收益率',
    return_4h DECIMAL(10,4) COMMENT '4小时收益率',
    return_24h DECIMAL(10,4) COMMENT '24小时收益率',
    return_7d DECIMAL(10,4) COMMENT '7天收益率',
    
    -- 回测状态
    status VARCHAR(16) DEFAULT 'pending' COMMENT 'pending/completed/failed',
    completed_at DATETIME COMMENT '完成时间',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    
    INDEX idx_recommendation_id (recommendation_id),
    INDEX idx_symbol (symbol),
    INDEX idx_recommended_at (recommended_at),
    INDEX idx_status (status)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='推荐回测结果表';

