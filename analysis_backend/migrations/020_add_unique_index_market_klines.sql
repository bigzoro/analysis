-- 为market_klines表添加唯一索引，防止重复数据
-- 唯一键：(symbol, kind, interval, open_time)

-- 首先删除现有的非唯一索引
DROP INDEX IF EXISTS idx_symbol_kind_interval_time ON market_klines;

-- 创建唯一索引
CREATE UNIQUE INDEX uk_symbol_kind_interval_time ON market_klines (symbol, kind, `interval`, open_time);

-- 添加注释说明
-- 此唯一索引确保每种交易对、市场类型、时间间隔的每个时间点只有一条K线记录
-- 如果尝试插入重复数据，数据库会报错，应用程序需要处理这个错误
