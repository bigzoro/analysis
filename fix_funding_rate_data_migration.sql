-- 资金费率数据迁移脚本
-- 将百分比数值转换为小数形式

-- 方案1：如果前端输入的是百分比（如1表示1%），需要转换为小数
UPDATE trading_strategies
SET conditions = JSON_SET(
    conditions,
    '$.futures_price_short_min_funding_rate', conditions->>'$.futures_price_short_min_funding_rate' / 100
)
WHERE conditions->>'$.futures_price_short_min_funding_rate' > 1
   OR conditions->>'$.futures_price_short_min_funding_rate' < -1;

-- 方案2：如果前端输入的是正确的小数形式，则无需迁移
-- 检查当前数据范围
SELECT
    id,
    conditions->>'$.futures_price_short_min_funding_rate' as funding_rate,
    name
FROM trading_strategies
WHERE conditions->>'$.futures_price_short_min_funding_rate' IS NOT NULL
ORDER BY CAST(conditions->>'$.futures_price_short_min_funding_rate' AS DECIMAL(10,4)) DESC;