-- 特征数据缓存表
-- 存储特征工程预计算的结果，支持快速查询和缓存管理

CREATE TABLE IF NOT EXISTS `feature_cache` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `symbol` varchar(32) NOT NULL COMMENT '交易对符号',
  `features` json NOT NULL COMMENT '特征数据(JSON格式)',
  `computed_at` timestamp(3) NOT NULL COMMENT '计算时间',
  `expires_at` timestamp(3) NOT NULL COMMENT '过期时间',
  `feature_count` int(11) NOT NULL DEFAULT '0' COMMENT '特征数量',
  `quality_score` float DEFAULT '0' COMMENT '质量评分(0-1)',
  `source` varchar(32) DEFAULT 'computed' COMMENT '数据来源(computed/realtime)',
  `time_window` int(11) DEFAULT '24' COMMENT '时间窗口(小时)',
  `data_points` int(11) DEFAULT '0' COMMENT '数据点数量',
  `created_at` timestamp(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `updated_at` timestamp(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_symbol_time_window` (`symbol`, `time_window`),
  KEY `idx_symbol` (`symbol`),
  KEY `idx_expires_at` (`expires_at`),
  KEY `idx_computed_at` (`computed_at`),
  KEY `idx_source` (`source`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='特征数据缓存表 - 存储特征工程预计算的结果';

-- 添加一些复合索引以优化查询性能
CREATE INDEX IF NOT EXISTS `idx_symbol_quality` ON `feature_cache` (`symbol`, `quality_score`, `computed_at`);
CREATE INDEX IF NOT EXISTS `idx_expires_cleanup` ON `feature_cache` (`expires_at`, `computed_at`);
