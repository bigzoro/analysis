-- 创建回测记录表
CREATE TABLE IF NOT EXISTS backtest_records (
    id BIGINT UNSIGNED PRIMARY KEY AUTO_INCREMENT,
    recommendation_id BIGINT UNSIGNED COMMENT '关联推荐ID',
    symbol VARCHAR(32) NOT NULL COMMENT '币种符号',
    base_symbol VARCHAR(16) NOT NULL COMMENT '基础币种',
    recommended_at DATETIME NOT NULL COMMENT '推荐时间',
    recommended_price DECIMAL(20,8) NOT NULL COMMENT '推荐时价格',
    
    -- 回测结果
    price_after_24h DECIMAL(20,8) COMMENT '24h后价格',
    price_after_7d DECIMAL(20,8) COMMENT '7天后价格',
    price_after_30d DECIMAL(20,8) COMMENT '30天后价格',
    
    performance_24h DECIMAL(10,4) COMMENT '24h收益率 %',
    performance_7d DECIMAL(10,4) COMMENT '7天收益率 %',
    performance_30d DECIMAL(10,4) COMMENT '30天收益率 %',
    
    status VARCHAR(16) DEFAULT 'pending' COMMENT 'pending/completed/failed',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    
    INDEX idx_recommendation_id (recommendation_id),
    INDEX idx_symbol (symbol),
    INDEX idx_recommended_at (recommended_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='回测记录表';

-- 创建模拟交易表
CREATE TABLE IF NOT EXISTS simulated_trades (
    id BIGINT UNSIGNED PRIMARY KEY AUTO_INCREMENT,
    user_id INT UNSIGNED NOT NULL COMMENT '用户ID',
    recommendation_id BIGINT UNSIGNED COMMENT '关联推荐ID（可选）',
    symbol VARCHAR(32) NOT NULL COMMENT '币种符号',
    base_symbol VARCHAR(16) NOT NULL COMMENT '基础币种',
    kind VARCHAR(16) NOT NULL COMMENT 'spot/futures',
    
    -- 交易信息
    side VARCHAR(8) NOT NULL COMMENT 'BUY/SELL',
    quantity DECIMAL(20,8) NOT NULL COMMENT '数量',
    price DECIMAL(20,8) NOT NULL COMMENT '成交价格',
    total_value DECIMAL(20,8) NOT NULL COMMENT '总价值',
    
    -- 持仓信息
    is_open BOOLEAN DEFAULT TRUE COMMENT '是否持仓中',
    current_price DECIMAL(20,8) COMMENT '当前价格',
    unrealized_pnl DECIMAL(20,8) COMMENT '未实现盈亏',
    unrealized_pnl_percent DECIMAL(10,4) COMMENT '未实现盈亏百分比',
    
    -- 卖出信息
    sold_at DATETIME COMMENT '卖出时间',
    realized_pnl DECIMAL(20,8) COMMENT '已实现盈亏',
    realized_pnl_percent DECIMAL(10,4) COMMENT '已实现盈亏百分比',
    
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    
    INDEX idx_user_id (user_id),
    INDEX idx_recommendation_id (recommendation_id),
    INDEX idx_symbol (symbol),
    INDEX idx_is_open (is_open)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='模拟交易表';

