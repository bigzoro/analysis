-- 确保技术指标缓存表存在并有正确的结构
-- 这个迁移确保预计算服务可以正确使用现有的表结构

-- 检查表是否存在，如果不存在则创建
CREATE TABLE IF NOT EXISTS `technical_indicators_caches` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `symbol` varchar(32) NOT NULL COMMENT '交易对符号',
  `kind` varchar(16) NOT NULL COMMENT '市场类型 (spot/futures)',
  `interval` varchar(8) NOT NULL COMMENT '时间间隔 (1m/5m/1h/1d等)',
  `data_points` int(11) NOT NULL DEFAULT '0' COMMENT '数据点数量',
  `indicators` json NOT NULL COMMENT '技术指标数据(JSON格式)',
  `calculated_at` timestamp(3) NOT NULL COMMENT '计算时间',
  `data_from` timestamp NOT NULL COMMENT '数据起始时间',
  `data_to` timestamp NOT NULL COMMENT '数据结束时间',
  `created_at` timestamp(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `updated_at` timestamp(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_symbol_kind_interval_data_points` (`symbol`,`kind`,`interval`,`data_points`),
  KEY `idx_symbol_kind_updated` (`symbol`,`kind`,`calculated_at`),
  KEY `idx_calculated_at` (`calculated_at`),
  KEY `idx_kind_interval_time` (`kind`,`interval`,`calculated_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='技术指标缓存表 - 存储预计算的技术指标数据';

-- 添加一些有用的索引（如果不存在）
-- 注意：由于使用了UNIQUE KEY，重复执行这个迁移是安全的

-- 优化查询性能的复合索引
CREATE INDEX IF NOT EXISTS `idx_symbol_timeframe` ON `technical_indicators_caches` (`symbol`, `interval`, `calculated_at`);
CREATE INDEX IF NOT EXISTS `idx_expires_at` ON `technical_indicators_caches` (`calculated_at`);

-- 添加表注释说明预计算服务的使用方式
-- 这个表现在被技术指标预计算服务使用，支持：
-- 1. 内存缓存 + 数据库持久化
-- 2. 自动过期和清理
-- 3. 多时间框架支持 (1m, 5m, 15m, 1h, 4h, 1d)
-- 4. 多种技术指标存储 (RSI, MACD, 布林带, KDJ等)
