-- 检查scheduled_orders表是否存在和结构
SHOW TABLES LIKE 'scheduled_orders';

-- 如果存在，查看表结构
DESCRIBE scheduled_orders;

-- 查看最近的订单
SELECT * FROM scheduled_orders ORDER BY created_at DESC LIMIT 5;


