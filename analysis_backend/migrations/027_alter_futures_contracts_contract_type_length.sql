-- 修改binance_futures_contracts表中contract_type字段长度
ALTER TABLE binance_futures_contracts MODIFY COLUMN contract_type VARCHAR(50);
