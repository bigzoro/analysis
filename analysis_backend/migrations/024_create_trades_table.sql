-- 创建实时交易数据表
CREATE TABLE IF NOT EXISTS binance_trades (
    id SERIAL PRIMARY KEY,
    symbol VARCHAR(20) NOT NULL,                   -- 交易对符号，如BTCUSDT
    market_type VARCHAR(10) NOT NULL,              -- 市场类型：spot/futures
    trade_id BIGINT NOT NULL,                      -- 交易ID
    price VARCHAR(32) NOT NULL,                    -- 成交价格
    quantity VARCHAR(32) NOT NULL,                 -- 成交数量
    trade_time BIGINT NOT NULL,                    -- 成交时间戳
    is_buyer_maker BOOLEAN NOT NULL,               -- 是否买方主动成交
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,

    UNIQUE KEY uniq_trade (symbol, trade_id),
    INDEX idx_symbol_trade_time (symbol, trade_time),
    INDEX idx_trade_time (trade_time),
    INDEX idx_market_type (market_type)
);
