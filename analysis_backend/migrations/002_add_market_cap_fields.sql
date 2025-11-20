-- 为 binance_market_tops 表添加市值相关字段
-- 如果字段已存在，会报错，需要先检查

-- 检查并添加 market_cap_usd 字段
SET @dbname = DATABASE();
SET @tablename = "binance_market_tops";
SET @columnname = "market_cap_usd";
SET @preparedStatement = (SELECT IF(
  (
    SELECT COUNT(*) FROM INFORMATION_SCHEMA.COLUMNS
    WHERE
      (table_name = @tablename)
      AND (table_schema = @dbname)
      AND (column_name = @columnname)
  ) > 0,
  "SELECT 'Column market_cap_usd already exists.';",
  CONCAT("ALTER TABLE ", @tablename, " ADD COLUMN ", @columnname, " DOUBLE NULL;")
));
PREPARE alterIfNotExists FROM @preparedStatement;
EXECUTE alterIfNotExists;
DEALLOCATE PREPARE alterIfNotExists;

-- 检查并添加 fdv_usd 字段
SET @columnname = "fdv_usd";
SET @preparedStatement = (SELECT IF(
  (
    SELECT COUNT(*) FROM INFORMATION_SCHEMA.COLUMNS
    WHERE
      (table_name = @tablename)
      AND (table_schema = @dbname)
      AND (column_name = @columnname)
  ) > 0,
  "SELECT 'Column fdv_usd already exists.';",
  CONCAT("ALTER TABLE ", @tablename, " ADD COLUMN ", @columnname, " DOUBLE NULL;")
));
PREPARE alterIfNotExists FROM @preparedStatement;
EXECUTE alterIfNotExists;
DEALLOCATE PREPARE alterIfNotExists;

-- 检查并添加 circulating_supply 字段
SET @columnname = "circulating_supply";
SET @preparedStatement = (SELECT IF(
  (
    SELECT COUNT(*) FROM INFORMATION_SCHEMA.COLUMNS
    WHERE
      (table_name = @tablename)
      AND (table_schema = @dbname)
      AND (column_name = @columnname)
  ) > 0,
  "SELECT 'Column circulating_supply already exists.';",
  CONCAT("ALTER TABLE ", @tablename, " ADD COLUMN ", @columnname, " DOUBLE NULL;")
));
PREPARE alterIfNotExists FROM @preparedStatement;
EXECUTE alterIfNotExists;
DEALLOCATE PREPARE alterIfNotExists;

-- 检查并添加 total_supply 字段
SET @columnname = "total_supply";
SET @preparedStatement = (SELECT IF(
  (
    SELECT COUNT(*) FROM INFORMATION_SCHEMA.COLUMNS
    WHERE
      (table_name = @tablename)
      AND (table_schema = @dbname)
      AND (column_name = @columnname)
  ) > 0,
  "SELECT 'Column total_supply already exists.';",
  CONCAT("ALTER TABLE ", @tablename, " ADD COLUMN ", @columnname, " DOUBLE NULL;")
));
PREPARE alterIfNotExists FROM @preparedStatement;
EXECUTE alterIfNotExists;
DEALLOCATE PREPARE alterIfNotExists;

