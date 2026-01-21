# 配置文件优化说明

## 变更概述

本次优化统一了Binance API密钥的配置管理，解决了配置文件中重复配置的问题。

## 具体变更

### 1. 配置结构优化
- **移除**: `data_sources.binance_futures` 配置项
- **保留**: `exchange.binance` 配置项作为统一配置源

### 2. 代码更新
更新了以下文件中的Binance客户端创建逻辑：
- `analysis_backend/internal/server/scheduler.go`
- `analysis_backend/internal/server/handlers.go`
- `analysis_backend/internal/server/orders.go`
- `analysis_backend/internal/exchange/binancefutures/client.go`

所有代码现在统一使用 `cfg.Exchange.Binance.APIKey` 和 `cfg.Exchange.Binance.SecretKey`。

### 3. 配置验证
添加了配置验证逻辑，在启动时检查 `exchange.binance` 配置是否完整。

## 迁移指南

### 对于现有用户：
1. 如果您已经在使用 `data_sources.binance_futures` 配置，请将其迁移到 `exchange.binance`
2. 更新您的配置文件，将API密钥从 `data_sources.binance_futures` 移动到 `exchange.binance`

### 配置示例：
```yaml
exchange:
  binance:
    api_key: "your_binance_api_key_here"
    secret_key: "your_binance_secret_key_here"
    testnet: true  # 生产环境设为 false
```

## 优势

1. **单一配置源**: 避免了重复配置，降低了维护成本
2. **配置一致性**: 确保数据获取和交易操作使用相同的API凭证
3. **错误减少**: 减少了因配置不一致导致的问题
4. **更清晰的职责**: `exchange.binance` 同时负责交易和数据获取

## 兼容性

- 向后兼容：如果配置文件中仍包含旧的 `data_sources.binance_futures` 配置，系统会记录警告但不会中断运行
- 验证机制：启动时会检查 `exchange.binance` 配置的完整性