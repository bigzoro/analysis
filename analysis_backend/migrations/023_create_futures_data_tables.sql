-- 创建期货数据相关表

-- 期货合约信息表
CREATE TABLE IF NOT EXISTS binance_futures_contracts (
    id SERIAL PRIMARY KEY,
    symbol VARCHAR(20) NOT NULL UNIQUE,           -- 交易对符号，如BTCUSDT
    status VARCHAR(20) NOT NULL,                  -- 合约状态，如TRADING
    contract_type VARCHAR(20),                    -- 合约类型，如PERPETUAL
    base_asset VARCHAR(20) NOT NULL,              -- 基础资产
    quote_asset VARCHAR(20) NOT NULL,             -- 计价资产
    margin_asset VARCHAR(20),                     -- 保证金资产
    price_precision INT,                          -- 价格精度
    quantity_precision INT,                       -- 数量精度
    base_asset_precision INT,                     -- 基础资产精度
    quote_precision INT,                          -- 计价精度
    underlying_type VARCHAR(20),                  -- 标的类型
    underlying_sub_type TEXT,                     -- 标的子类型
    settle_plan INT,                              -- 清算计划
    trigger_protect DECIMAL(5,4),                 -- 触发保护
    filters TEXT,                                 -- 过滤器，JSON格式
    order_types TEXT,                             -- 支持的订单类型，JSON数组
    time_in_force TEXT,                           -- 支持的时间生效方式，JSON数组
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,

    INDEX idx_futures_symbol (symbol),
    INDEX idx_futures_status (status),
    INDEX idx_futures_contract_type (contract_type),
    INDEX idx_futures_updated_at (updated_at)
);

-- 资金费率表
CREATE TABLE IF NOT EXISTS binance_funding_rates (
    id SERIAL PRIMARY KEY,
    symbol VARCHAR(20) NOT NULL,                   -- 交易对符号
    funding_rate DECIMAL(10,8) NOT NULL,           -- 资金费率
    funding_time BIGINT NOT NULL,                  -- 资金费率时间戳
    mark_price DECIMAL(20,8),                      -- 标记价格
    index_price DECIMAL(20,8),                     -- 指数价格
    estimated_settle_price DECIMAL(20,8),          -- 预估结算价格
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,

    UNIQUE KEY uk_funding_rate (symbol, funding_time),
    INDEX idx_funding_symbol_time (symbol, funding_time),
    INDEX idx_funding_time (funding_time)
);

-- 订单簿深度表
CREATE TABLE IF NOT EXISTS binance_order_book_depth (
    id SERIAL PRIMARY KEY,
    symbol VARCHAR(20) NOT NULL,                   -- 交易对符号
    market_type VARCHAR(10) NOT NULL,              -- 市场类型：spot/futures
    last_update_id BIGINT NOT NULL,                -- 最后更新ID
    bids TEXT NOT NULL,                            -- 买单深度，JSON格式 [[price, quantity], ...]
    asks TEXT NOT NULL,                            -- 卖单深度，JSON格式 [[price, quantity], ...]
    snapshot_time BIGINT,                          -- 快照时间戳
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,

    INDEX idx_depth_symbol_type (symbol, market_type),
    INDEX idx_depth_symbol_time (symbol, created_at),
    INDEX idx_depth_type_time (market_type, created_at)
);

-- 24小时统计数据表
CREATE TABLE IF NOT EXISTS binance_24h_stats (
    id SERIAL PRIMARY KEY,
    symbol VARCHAR(20) NOT NULL,                   -- 交易对符号
    market_type VARCHAR(10) NOT NULL,              -- 市场类型：spot/futures
    price_change DECIMAL(20,8),                    -- 价格变化
    price_change_percent DECIMAL(10,4),            -- 价格变化百分比
    weighted_avg_price DECIMAL(20,8),              -- 加权平均价
    prev_close_price DECIMAL(20,8),                -- 前收盘价
    last_price DECIMAL(20,8),                      -- 最新价
    bid_price DECIMAL(20,8),                       -- 买一价
    ask_price DECIMAL(20,8),                       -- 卖一价
    open_price DECIMAL(20,8),                      -- 开盘价
    high_price DECIMAL(20,8),                      -- 最高价
    low_price DECIMAL(20,8),                       -- 最低价
    volume DECIMAL(30,8),                          -- 成交量
    quote_volume DECIMAL(30,8),                    -- 成交额
    open_time BIGINT,                              -- 开盘时间
    close_time BIGINT,                             -- 收盘时间
    count BIGINT,                                  -- 成交笔数
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,

    UNIQUE KEY uk_24h_stats (symbol, market_type, close_time),
    INDEX idx_24h_symbol_type_time (symbol, market_type, close_time),
    INDEX idx_24h_type_time (market_type, close_time)
);
