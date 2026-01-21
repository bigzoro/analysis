-- 添加均线策略相关字段到trading_strategies表
-- Migration: 025_add_moving_average_strategy_fields
-- Created: 2025-01-26

-- 为trading_strategies表添加均线策略字段
-- 注意：由于SQLite不支持ALTER TABLE添加多个列，我们需要重新创建表

-- 首先备份现有数据
CREATE TABLE trading_strategies_backup AS SELECT * FROM trading_strategies;

-- 删除原表
DROP TABLE trading_strategies;

-- 重新创建表，包含新的均线策略字段
CREATE TABLE trading_strategies (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER NOT NULL,
    name VARCHAR(128) NOT NULL,
    description TEXT,

    -- 策略条件（JSON格式存储）
    conditions JSON NOT NULL DEFAULT '{}',

    -- 运行状态
    is_running BOOLEAN DEFAULT FALSE,
    last_run_at DATETIME,
    run_interval INTEGER DEFAULT 60,

    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,

    -- 索引
    INDEX idx_user_id (user_id),
    INDEX idx_is_running (is_running),
    INDEX idx_created_at (created_at)
);

-- 恢复数据
INSERT INTO trading_strategies (
    id, user_id, name, description, conditions,
    is_running, last_run_at, run_interval, created_at, updated_at
)
SELECT
    id, user_id, name, description,
    -- 更新conditions字段，添加均线策略默认值
    json_set(
        json_set(
            json_set(
                json_set(
                    json_set(
                        json_set(
                            json_set(
                                json_set(
                                    json_set(conditions, '$.moving_average_enabled', false),
                                    '$.ma_type', 'SMA'
                                ),
                                '$.short_ma_period', 5
                            ),
                            '$.long_ma_period', 20
                        ),
                        '$.ma_cross_signal', 'BOTH'
                    ),
                    '$.ma_trend_filter', false
                ),
                '$.ma_trend_direction', 'UP'
            ),
            '$.technical_indicators', json('{}')
        ),
        '$.strategy_version', '2.0'
    ) as conditions,
    is_running, last_run_at, run_interval, created_at, updated_at
FROM trading_strategies_backup;

-- 删除备份表
DROP TABLE trading_strategies_backup;

-- 添加注释
-- moving_average_enabled: 是否启用均线策略
-- ma_type: 均线类型 ('SMA', 'EMA', 'WMA')
-- short_ma_period: 短期均线周期
-- long_ma_period: 长期均线周期
-- ma_cross_signal: 交叉信号类型 ('GOLDEN_CROSS', 'DEATH_CROSS', 'BOTH')
-- ma_trend_filter: 是否启用趋势过滤
-- ma_trend_direction: 趋势方向 ('UP', 'DOWN', 'BOTH')
