# 生产环境配置指南

## 环境配置说明

系统现在支持测试环境和生产环境的灵活切换，通过配置文件中的 `exchange.environment` 字段控制。

### 配置结构

```yaml
exchange:
  # 环境选择：testnet 或 mainnet
  environment: "testnet"  # 或 "mainnet"

  binance:
    # 测试环境配置
    testnet:
      api_key: "您的测试环境API密钥"
      secret_key: "您的测试环境密钥"
      enabled: true

    # 生产环境配置
    mainnet:
      api_key: "您的生产环境API密钥"
      secret_key: "您的生产环境密钥"
      enabled: false  # ⚠️ 生产环境请谨慎启用
```

## 配置正式环境密钥

### 步骤1：获取生产环境API密钥

1. 登录您的 [Binance 账户](https://www.binance.com)
2. 进入 API 管理页面：账户设置 → API 管理
3. 创建新的API密钥（建议为交易系统单独创建）
4. **重要安全设置**：
   - 启用 IP 白名单（只允许您的服务器IP访问）
   - 限制API权限：
     - ✅ 启用现货交易
     - ✅ 启用期货交易
     - ✅ 启用读取权限
     - ❌ 禁用提现权限（安全起见）
   - 开启双重认证

### 步骤2：配置生产环境

在 `analysis_backend/config.yaml` 中更新配置：

```yaml
exchange:
  environment: "mainnet"  # 切换到生产环境

  binance:
    testnet:
      api_key: "GWPl7AdopAHSkrAih3SEJu0BMYhNqHhIczTYmTpLUhi1ejtu4g2nxlW5qSEx0f7q"
      secret_key: "NapaXbuLZd9ENt51iAvNBWjpxOsvJmZa99wvboeUzIYhU7uCTDgfcwxbG2JxVckC"
      enabled: true

    mainnet:
      api_key: "您的生产环境API密钥"      # 🔴 填入您的真实API密钥
      secret_key: "您的生产环境密钥"       # 🔴 填入您的真实密钥
      enabled: true                       # 🔴 启用生产环境
```

### 步骤3：安全验证

生产环境启用前，请确保：

1. **备份测试环境配置**
2. **小额测试**：先用小金额测试交易功能
3. **监控设置**：启用所有监控和告警功能
4. **资金安全**：确保账户中有足够的测试资金，但不要存放过多资金

### 步骤4：切换环境

```bash
# 测试环境切换
exchange:
  environment: "testnet"

# 生产环境切换
exchange:
  environment: "mainnet"
```

## 安全建议

### API密钥安全
- 🔐 **定期轮换**：每3-6个月更换一次API密钥
- 🌐 **IP限制**：只允许可信IP访问API
- 🚫 **权限最小化**：只授予必要权限，禁用提现功能
- 🔑 **双重认证**：为账户和API启用2FA

### 资金安全
- 💰 **分层管理**：交易账户与存储账户分离
- 📊 **限额控制**：设置每日/单笔交易限额
- 🚨 **监控告警**：实时监控异常交易
- 🛑 **紧急停止**：配置自动停止机制

### 系统安全
- 🔄 **备份策略**：定期备份配置和数据
- 📝 **日志审计**：记录所有交易和配置变更
- 🧪 **测试验证**：生产部署前充分测试
- 📞 **应急预案**：准备故障恢复方案

## 环境切换命令

系统启动时会根据配置自动选择环境，您也可以通过修改配置文件来切换环境。

## 故障排除

如果遇到配置问题，请检查：
1. API密钥是否正确填写
2. IP白名单是否包含服务器IP
3. API权限是否正确设置
4. 账户余额是否充足

## 联系支持

如有配置问题，请联系技术支持团队。