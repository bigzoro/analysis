-- 创建币种推荐结果表
CREATE TABLE IF NOT EXISTS coin_recommendations (
    id BIGINT UNSIGNED PRIMARY KEY AUTO_INCREMENT,
    generated_at DATETIME NOT NULL COMMENT '生成时间',
    kind VARCHAR(16) NOT NULL COMMENT 'spot/futures',
    symbol VARCHAR(32) NOT NULL COMMENT '币种符号，如BTCUSDT',
    base_symbol VARCHAR(16) NOT NULL COMMENT '基础币种，如BTC',
    `rank` INT NOT NULL COMMENT '推荐排名，1-5',
    total_score DECIMAL(5,2) NOT NULL COMMENT '总分 0-100',
    
    -- 各因子得分
    market_score DECIMAL(5,2) NOT NULL COMMENT '市场表现得分',
    flow_score DECIMAL(5,2) NOT NULL COMMENT '资金流得分',
    heat_score DECIMAL(5,2) NOT NULL COMMENT '市场热度得分',
    event_score DECIMAL(5,2) NOT NULL COMMENT '事件得分',
    sentiment_score DECIMAL(5,2) NOT NULL COMMENT '情绪得分',
    
    -- 原始数据快照
    price_change_24h DECIMAL(10,4) COMMENT '24h涨幅',
    volume_24h DECIMAL(20,8) COMMENT '24h成交量',
    market_cap_usd DECIMAL(20,2) COMMENT '市值',
    net_flow_24h DECIMAL(20,8) COMMENT '24h净流入',
    has_new_listing BOOLEAN DEFAULT FALSE COMMENT '是否新上线',
    has_announcement BOOLEAN DEFAULT FALSE COMMENT '是否有公告',
    twitter_mentions INT COMMENT 'Twitter提及次数',
    
    -- 推荐理由（JSON格式）
    reasons JSON COMMENT '推荐理由详情',
    
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    
    INDEX idx_generated_at (generated_at),
    INDEX idx_kind_generated_at (kind, generated_at),
    INDEX idx_symbol (symbol)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='币种推荐结果表';

