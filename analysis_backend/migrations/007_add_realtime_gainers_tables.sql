-- 实时涨幅榜数据持久化表
-- 迁移编号: 007
-- 创建时间: 2024-12-12
-- 说明: 添加实时涨幅榜历史数据存储表

-- 创建实时涨幅榜快照表
CREATE TABLE IF NOT EXISTS `realtime_gainers_snapshots` (
    `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
    `kind` varchar(16) NOT NULL COMMENT '交易类型: spot/futures',
    `timestamp` datetime NOT NULL COMMENT '数据采集时间戳',
    `created_at` datetime(3) DEFAULT NULL,
    PRIMARY KEY (`id`),
    KEY `idx_gainers_kind_timestamp` (`kind`, `timestamp`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='实时涨幅榜快照表';

-- 创建涨幅榜数据项表
CREATE TABLE IF NOT EXISTS `realtime_gainers_items` (
    `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
    `snapshot_id` bigint(20) unsigned NOT NULL COMMENT '关联快照ID',
    `symbol` varchar(32) NOT NULL COMMENT '交易对符号',
    `rank` int(11) NOT NULL COMMENT '排名',
    `current_price` double NOT NULL COMMENT '当前价格',
    `price_change_24h` double NOT NULL COMMENT '24小时涨跌幅',
    `volume_24h` double NOT NULL COMMENT '24小时成交量',
    `data_source` varchar(16) NOT NULL COMMENT '数据来源: websocket/http_api/kline_calc',
    `price_change_percent` double DEFAULT NULL COMMENT '价格变化百分比',
    `confidence` double DEFAULT NULL COMMENT '置信度',
    `created_at` datetime(3) DEFAULT NULL,
    PRIMARY KEY (`id`),
    KEY `idx_snapshot_id` (`snapshot_id`),
    KEY `idx_symbol` (`symbol`),
    KEY `idx_rank` (`rank`),
    KEY `idx_data_source` (`data_source`),
    CONSTRAINT `fk_realtime_gainers_items_snapshot` FOREIGN KEY (`snapshot_id`) REFERENCES `realtime_gainers_snapshots` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='涨幅榜数据项表';

-- 添加一些有用的索引用于查询优化
CREATE INDEX IF NOT EXISTS `idx_gainers_timestamp_desc` ON `realtime_gainers_snapshots` (`timestamp` DESC);
CREATE INDEX IF NOT EXISTS `idx_gainers_items_symbol_timestamp` ON `realtime_gainers_items` (`symbol`, `created_at`);

-- 添加表注释
ALTER TABLE `realtime_gainers_snapshots` COMMENT '实时涨幅榜快照表 - 存储每次数据采集的时间点信息';
ALTER TABLE `realtime_gainers_items` COMMENT '涨幅榜数据项表 - 存储每个币种在快照中的详细信息';
