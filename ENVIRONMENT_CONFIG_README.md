# 环境配置管理

## 概述

系统现在支持测试环境和生产环境的灵活切换，通过统一的配置文件管理不同环境的API密钥。

## 配置结构

```yaml
exchange:
  # 环境选择：testnet 或 mainnet
  environment: "testnet"

  binance:
    # 测试环境配置
    testnet:
      api_key: "GWPl7AdopAHSkrAih3SEJu0BMYhNqHhIczTYmTpLUhi1ejtu4g2nxlW5qSEx0f7q"
      secret_key: "NapaXbuLZd9ENt51iAvNBWjpxOsvJmZa99wvboeUzIYhU7uCTDgfcwxbG2JxVckC"
      enabled: true

    # 生产环境配置
    mainnet:
      api_key: ""  # 填入您的生产环境API密钥
      secret_key: ""  # 填入您的生产环境密钥
      enabled: false  # 生产环境默认禁用
```

## 环境切换

### 切换到测试环境
```yaml
exchange:
  environment: "testnet"
```

### 切换到生产环境
```yaml
exchange:
  environment: "mainnet"
```

⚠️ **重要**：切换到生产环境前，请确保：
1. 生产环境API密钥已正确配置
2. 生产环境已启用 (`enabled: true`)
3. 账户中有足够的测试资金
4. 已充分测试所有功能

## 安全配置建议

### 生产环境API密钥
- 🔐 **单独创建**：为交易系统单独创建API密钥
- 🌐 **IP限制**：设置IP白名单，只允许服务器IP访问
- 🚫 **权限控制**：只启用必要的权限，禁用提现功能
- 🔄 **定期轮换**：每3-6个月更换一次API密钥

### 配置管理
- 📁 **版本控制**：不要将真实API密钥提交到代码仓库
- 🔒 **环境变量**：考虑使用环境变量管理敏感信息
- 👥 **权限控制**：限制配置文件访问权限
- 📝 **审计日志**：记录配置变更历史

## 配置验证

系统启动时会自动验证配置：

### 测试环境验证
- 检查测试环境API密钥是否完整
- 验证API连接性

### 生产环境验证
- 检查生产环境API密钥是否已配置
- 验证生产环境是否已启用
- 警告生产环境风险

### 自动回退
如果选择的生产环境未启用，系统会自动回退到测试环境。

## 使用示例

### 开发环境
```yaml
exchange:
  environment: "testnet"  # 使用测试环境
```

### 生产部署
```yaml
exchange:
  environment: "mainnet"  # 使用生产环境

  binance:
    mainnet:
      api_key: "your_production_api_key"
      secret_key: "your_production_secret_key"
      enabled: true
```

## 故障排除

### 配置无效
```
[ERROR] 当前环境 (mainnet) 的API密钥配置不完整
```
**解决方案**：
1. 检查对应环境的API密钥配置
2. 确保 `enabled: true`
3. 验证API密钥格式

### 权限不足
```
API key does not exist or has insufficient permissions
```
**解决方案**：
1. 检查API密钥是否正确
2. 验证API权限设置
3. 确认IP白名单配置

### 网络连接
```
Connection timeout
```
**解决方案**：
1. 检查网络连接
2. 验证API端点可访问性
3. 确认防火墙设置

## 最佳实践

1. **渐进式部署**：先在测试环境充分测试，再切换到生产环境
2. **小额测试**：生产环境初期使用小金额测试
3. **监控告警**：启用所有监控和告警功能
4. **备份恢复**：准备应急恢复方案
5. **文档记录**：记录所有配置变更和测试结果