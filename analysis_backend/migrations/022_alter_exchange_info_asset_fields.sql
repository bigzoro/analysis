-- 修改Binance交易对信息表的资产字段长度
-- 以支持更长的资产名称（如PEPE等）

ALTER TABLE binance_exchange_info
    MODIFY COLUMN base_asset VARCHAR(20) NOT NULL,
    MODIFY COLUMN quote_asset VARCHAR(20) NOT NULL;
