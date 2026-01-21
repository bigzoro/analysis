-- Migration: 030_extend_data_source_length
-- Date: 2026-01-09
-- Description: 扩展 realtime_gainers_items 表的 data_source 字段长度
-- Reason: 避免 "Data too long for column" 错误，支持更长的标识字符串

-- 扩展 data_source 字段长度从 16 到 32 字符
ALTER TABLE realtime_gainers_items
MODIFY COLUMN data_source VARCHAR(32) NOT NULL DEFAULT '' COMMENT '数据来源标识';

-- 验证修改结果
SELECT
    COLUMN_NAME,
    DATA_TYPE,
    CHARACTER_MAXIMUM_LENGTH,
    COLUMN_COMMENT
FROM information_schema.COLUMNS
WHERE TABLE_NAME = 'realtime_gainers_items'
  AND COLUMN_NAME = 'data_source';

-- 示例数据源标识：
-- 'websocket'        - WebSocket实时数据 (9 chars)
-- 'http_api'         - HTTP API数据 (7 chars)
-- 'init_populate'    - 初始化填充数据 (12 chars)
-- 'realtime_ws'      - 实时同步器数据 (11 chars)
-- 'manual_sync'      - 手动同步数据 (11 chars)