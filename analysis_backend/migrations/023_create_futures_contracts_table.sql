-- 创建Binance期货合约信息表
CREATE TABLE IF NOT EXISTS binance_futures_contracts (
    id SERIAL PRIMARY KEY,
    symbol VARCHAR(20) NOT NULL UNIQUE,           -- 交易对符号，如BTCUSDT
    status VARCHAR(20) NOT NULL,                  -- 合约状态，如TRADING
    contract_type VARCHAR(20),                    -- 合约类型，如PERPETUAL
    base_asset VARCHAR(20) NOT NULL,              -- 基础资产，如BTC
    quote_asset VARCHAR(20) NOT NULL,             -- 计价资产，如USDT
    price_precision INT NOT NULL,                 -- 价格精度
    quantity_precision INT NOT NULL,              -- 数量精度
    base_asset_precision INT NOT NULL,            -- 基础资产精度
    quote_precision INT NOT NULL,                 -- 计价资产精度
    order_types TEXT,                             -- 支持的订单类型，JSON数组格式
    time_in_force TEXT,                           -- 支持的时间类型，JSON数组格式
    liquidation_fee DECIMAL(10,8) DEFAULT 0,      -- 强平手续费
    market_take_bound DECIMAL(10,8) DEFAULT 0,    -- 市价吃单比例
    filters TEXT,                                 -- 交易对过滤器，JSON格式
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,

    INDEX idx_symbol (symbol),
    INDEX idx_status (status),
    INDEX idx_contract_type (contract_type),
    INDEX idx_base_asset (base_asset),
    INDEX idx_quote_asset (quote_asset),
    INDEX idx_updated_at (updated_at)
);

-- 创建复合索引以提高查询性能
CREATE INDEX idx_futures_contracts_symbol_status ON binance_futures_contracts (symbol, status);
