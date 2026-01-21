-- 市场环境分析SQL查询
-- 执行前请确保连接到正确的数据库

-- 1. 基本市场概览
SELECT
    COUNT(*) as total_symbols,
    COUNT(CASE WHEN quote_volume > 1000000 THEN 1 END) as active_symbols,
    AVG(price_change_percent) as avg_price_change,
    AVG((high_price - low_price) / low_price * 100) as avg_volatility
FROM binance_24h_stats
WHERE created_at >= DATE_SUB(NOW(), INTERVAL 24 HOUR)
    AND market_type = 'spot'
    AND quote_volume > 100000;

-- 2. 波动率分布统计
SELECT
    CASE
        WHEN volatility < 1 THEN '<1%'
        WHEN volatility < 2 THEN '1-2%'
        WHEN volatility < 5 THEN '2-5%'
        WHEN volatility < 10 THEN '5-10%'
        ELSE '>10%'
    END as volatility_range,
    COUNT(*) as symbol_count
FROM (
    SELECT (high_price - low_price) / low_price * 100 as volatility
    FROM binance_24h_stats
    WHERE created_at >= DATE_SUB(NOW(), INTERVAL 24 HOUR)
        AND market_type = 'spot'
        AND quote_volume > 100000
) as vol_stats
GROUP BY
    CASE
        WHEN volatility < 1 THEN '<1%'
        WHEN volatility < 2 THEN '1-2%'
        WHEN volatility < 5 THEN '2-5%'
        WHEN volatility < 10 THEN '5-10%'
        ELSE '>10%'
    END
ORDER BY symbol_count DESC;

-- 3. 趋势分析
SELECT
    COUNT(CASE WHEN price_change_percent > 5 THEN 1 END) as bullish_symbols,
    COUNT(CASE WHEN price_change_percent < -5 THEN 1 END) as bearish_symbols,
    COUNT(CASE WHEN ABS(price_change_percent) <= 5 THEN 1 END) as oscillating_symbols,
    COUNT(CASE WHEN ABS(price_change_percent) > 2 THEN 1 END) as trending_symbols,
    COUNT(*) as total_analyzed
FROM binance_24h_stats
WHERE created_at >= DATE_SUB(NOW(), INTERVAL 24 HOUR)
    AND market_type = 'spot'
    AND quote_volume > 100000;

-- 4. 涨幅榜TOP10
SELECT symbol, price_change_percent, quote_volume,
       (high_price - low_price) / low_price * 100 as volatility
FROM binance_24h_stats
WHERE created_at >= DATE_SUB(NOW(), INTERVAL 24 HOUR)
    AND market_type = 'spot'
ORDER BY price_change_percent DESC
LIMIT 10;

-- 5. 跌幅榜TOP10
SELECT symbol, price_change_percent, quote_volume,
       (high_price - low_price) / low_price * 100 as volatility
FROM binance_24h_stats
WHERE created_at >= DATE_SUB(NOW(), INTERVAL 24 HOUR)
    AND market_type = 'spot'
ORDER BY price_change_percent ASC
LIMIT 10;

-- 6. 高波动率币种
SELECT symbol, price_change_percent,
       (high_price - low_price) / low_price * 100 as volatility,
       quote_volume
FROM binance_24h_stats
WHERE created_at >= DATE_SUB(NOW(), INTERVAL 24 HOUR)
    AND market_type = 'spot'
    AND (high_price - low_price) / low_price * 100 > 5
ORDER BY (high_price - low_price) / low_price * 100 DESC
LIMIT 10;

-- 7. 市场状态判断依据
SELECT
    AVG(price_change_percent) as market_sentiment,
    AVG((high_price - low_price) / low_price * 100) as market_volatility,
    SUM(CASE WHEN price_change_percent > 2 THEN 1 ELSE 0 END) / COUNT(*) as bullish_ratio,
    SUM(CASE WHEN price_change_percent < -2 THEN 1 ELSE 0 END) / COUNT(*) as bearish_ratio,
    SUM(CASE WHEN ABS(price_change_percent) > 3 THEN 1 ELSE 0 END) / COUNT(*) as trending_ratio
FROM binance_24h_stats
WHERE created_at >= DATE_SUB(NOW(), INTERVAL 24 HOUR)
    AND market_type = 'spot'
    AND quote_volume > 100000;
