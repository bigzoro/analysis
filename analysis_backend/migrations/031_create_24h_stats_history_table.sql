-- 创建binance_24h_stats_history表 - 存储24小时统计数据的时间序列
-- 支持量化交易系统的历史数据分析，包括技术指标计算、策略回测等
-- +migrate Up

-- 创建历史统计数据表
CREATE TABLE IF NOT EXISTS binance_24h_stats_history (
    id SERIAL PRIMARY KEY,
    symbol VARCHAR(20) NOT NULL COMMENT '交易对符号',
    market_type VARCHAR(10) NOT NULL COMMENT '市场类型：spot/futures',

    -- 时间窗口标识（核心新增字段）
    window_start TIMESTAMP NOT NULL COMMENT '时间窗口开始时间',
    window_end TIMESTAMP NOT NULL COMMENT '时间窗口结束时间',
    window_duration INT NOT NULL DEFAULT 3600 COMMENT '窗口持续时间(秒，默认1小时)',

    -- 完整统计数据（与实时表保持一致）
    price_change DECIMAL(20,8) COMMENT '价格变化',
    price_change_percent DECIMAL(10,4) COMMENT '价格变化百分比',
    weighted_avg_price DECIMAL(20,8) COMMENT '加权平均价',
    prev_close_price DECIMAL(20,8) COMMENT '前收盘价',
    last_price DECIMAL(20,8) COMMENT '最新价',
    last_qty DECIMAL(20,8) COMMENT '最后交易数量',
    bid_price DECIMAL(20,8) COMMENT '买一价',
    bid_qty DECIMAL(20,8) COMMENT '买一档数量',
    ask_price DECIMAL(20,8) COMMENT '卖一价',
    ask_qty DECIMAL(20,8) COMMENT '卖一档数量',
    open_price DECIMAL(20,8) COMMENT '开盘价',
    high_price DECIMAL(20,8) COMMENT '最高价',
    low_price DECIMAL(20,8) COMMENT '最低价',
    volume DECIMAL(30,8) COMMENT '成交量',
    quote_volume DECIMAL(30,8) COMMENT '成交额',
    open_time BIGINT COMMENT '开盘时间',
    close_time BIGINT COMMENT '收盘时间',
    first_id BIGINT COMMENT '第一笔交易ID',
    last_id BIGINT COMMENT '最后一笔交易ID',
    count BIGINT COMMENT '成交笔数',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP COMMENT '记录创建时间',

    -- 时间序列唯一索引（防止同一时间窗口重复数据）
    UNIQUE KEY uk_history_stats (symbol, market_type, window_start),

    -- 时间序列查询优化索引
    INDEX idx_history_time_series (symbol, market_type, window_start),
    INDEX idx_history_symbol_window (symbol, window_start),
    INDEX idx_history_market_window (market_type, window_start),
    INDEX idx_history_window_range (window_start, window_end),
    INDEX idx_history_created_at (created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='Binance 24小时统计数据历史表 - 时间序列存储';

-- +migrate Down

-- 删除历史统计数据表
DROP TABLE IF EXISTS binance_24h_stats_history;