-- 检查数据库中的 exchange_info 表
.schema binance_exchange_info

-- 查看表中的记录数量
SELECT COUNT(*) as total_records FROM binance_exchange_info;

-- 查看一些示例记录
SELECT symbol, status, LENGTH(filters) as filters_length, updated_at
FROM binance_exchange_info
LIMIT 10;

-- 检查特定交易对
SELECT symbol, status, filters
FROM binance_exchange_info
WHERE symbol IN ('BTCUSDT', 'ETHUSDT', 'FILUSDT', 'FHEUSDT', 'RIVERUSDT');