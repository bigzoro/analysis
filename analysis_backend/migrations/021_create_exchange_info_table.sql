-- 创建Binance交易对信息表
CREATE TABLE IF NOT EXISTS binance_exchange_info (
    id SERIAL PRIMARY KEY,
    symbol VARCHAR(20) NOT NULL UNIQUE,           -- 交易对符号，如BTCUSDT
    status VARCHAR(20) NOT NULL,                  -- 交易对状态，如TRADING
    base_asset VARCHAR(20) NOT NULL,              -- 基础资产，如BTC、PEPE等
    quote_asset VARCHAR(20) NOT NULL,             -- 计价资产，如USDT、BUSD等
    base_asset_precision INT NOT NULL,            -- 基础资产精度
    quote_asset_precision INT NOT NULL,           -- 计价资产精度
    base_commission_precision INT NOT NULL,       -- 基础资产手续费精度
    quote_commission_precision INT NOT NULL,      -- 计价资产手续费精度
    order_types TEXT,                             -- 支持的订单类型，JSON数组格式
    iceberg_allowed BOOLEAN DEFAULT FALSE,        -- 是否支持冰山订单
    oco_allowed BOOLEAN DEFAULT FALSE,            -- 是否支持OCO订单
    quote_order_qty_market_allowed BOOLEAN DEFAULT FALSE, -- 是否支持按计价资产数量下市价单
    allow_trailing_stop BOOLEAN DEFAULT FALSE,    -- 是否支持跟踪止损
    cancel_replace_allowed BOOLEAN DEFAULT FALSE, -- 是否支持取消替换
    is_spot_trading_allowed BOOLEAN DEFAULT TRUE, -- 是否允许现货交易
    is_margin_trading_allowed BOOLEAN DEFAULT FALSE, -- 是否允许杠杆交易
    filters TEXT,                                 -- 交易对过滤器，JSON格式
    permissions TEXT,                             -- 权限列表，JSON数组格式
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,

    INDEX idx_symbol (symbol),
    INDEX idx_status (status),
    INDEX idx_quote_asset (quote_asset),
    INDEX idx_base_asset (base_asset),
    INDEX idx_updated_at (updated_at)
);

-- 创建复合索引以提高查询性能
CREATE INDEX idx_exchange_info_symbol_status ON binance_exchange_info (symbol, status);
CREATE INDEX idx_exchange_info_quote_asset_status ON binance_exchange_info (quote_asset, status);
