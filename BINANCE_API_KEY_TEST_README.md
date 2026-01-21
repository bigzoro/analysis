# 币安API密钥测试工具使用说明

## 简介

`test_binance_api_key.go` 是一个用于快速测试币安正式环境API密钥是否可用的Go程序。该工具会验证API密钥的有效性、权限和基本功能。

## 使用步骤

### 1. 编辑API密钥

打开 `test_binance_api_key.go` 文件，找到以下两行：

```go
apiKey := "your_api_key_here"    // 替换为您的API Key
secretKey := "your_secret_here"  // 替换为您的Secret Key
```

将 `your_api_key_here` 替换为您的币安API Key，将 `your_secret_here` 替换为您的币安Secret Key。

### 2. 选择环境

如果需要测试币安测试网络，将以下行：

```go
useTestnet := false
```

改为：

```go
useTestnet := true
```

**注意：** 生产环境测试请保持 `useTestnet := false`

### 3. 运行测试

```bash
go run test_binance_api_key.go
```

## 测试内容

该工具会执行以下测试：

1. **账户信息验证** - 测试API密钥是否有效，检查账户权限
2. **余额检查** - 显示账户中的可用余额
3. **交易对信息** - 验证交易对查询功能
4. **价格查询** - 测试实时价格获取功能

## 测试结果说明

### 调试信息

程序现在会显示：
- 📄 原始响应预览：API返回的原始JSON数据前500字符
- 🔍 解析结果：显示账户类型和权限的解析情况

这有助于诊断API响应格式问题。

### 成功结果示例

```
🔑 币安API密钥测试工具
========================
🔗 使用正式网环境

1️⃣ 测试账户信息接口...
📄 原始响应预览: {"feeTier":0,"canTrade":true,"canDeposit":true,"canWithdraw":true,"feeBurn":true,"tradeGroupId":-1,"updateTime":0,"multiAssetsMargin":false,"totalInitialMargin":"0.00000000",...
🔍 解析结果 - 账户类型: '', 权限: []
✅ API密钥验证成功！
📊 费用等级: 0
💰 是否可以交易: true
💸 是否可以提币: true
📥 是否可以充币: true
🔥 费用燃烧: true
🔄 多资产保证金: false
🔐 权限列表: (无明确权限字段，由canTrade/canWithdraw等控制)

2️⃣ 检查账户余额...
   💰 总钱包余额: 1000.00455766 USDT
   💵 可用余额: 1000.00455766 USDT
   📊 保证金余额: 1000.00455766 USDT
   ✅ 账户有可用资金！

3️⃣ 测试其他API接口...
   🔍 测试交易对信息接口...
   ✅ 交易对信息接口正常
   💰 测试价格查询接口...
   ✅ 价格查询接口正常

🎉 所有测试完成！您的API密钥工作正常。
```

### 关于账户信息显示

程序现在会显示更详细的账户信息：
- **费用等级**：显示您的交易费用等级
- **总钱包余额**：账户的总资金
- **可用余额**：可用于交易的资金
- **保证金余额**：用于保证金的资金
- **费用燃烧**：是否启用费用燃烧功能
- **多资产保证金**：是否启用多资产保证金

如果您看到账户有余额但显示为0，说明程序现在正确解析了API响应！
```

### 常见错误及解决方法

#### API密钥错误
```
❌ API错误 (代码: -2015): Invalid API-key, IP, or permissions for action
```
**解决方法：**
- 检查API Key和Secret Key是否正确
- 确认API Key是否有期货交易权限
- 检查IP白名单设置

#### 网络连接问题
```
❌ 发送请求失败: dial tcp: i/o timeout
```
**解决方法：**
- 检查网络连接
- 如果在中国大陆，可能需要配置代理

#### 权限不足
```
❌ API错误 (代码: -2015): API key does not exist
```
**解决方法：**
- 确认API Key已启用
- 检查API Key是否有读取权限

## 安全提醒

⚠️ **重要安全注意事项：**

1. **不要在公共场所编辑或运行此代码**
2. **不要将包含真实API密钥的代码提交到版本控制系统**
3. **测试完成后，及时删除或修改代码中的API密钥**
4. **定期更换API密钥以确保安全**
5. **限制API密钥的权限，只启用必要的功能**

## 获取API密钥

1. 登录币安账户
2. 进入 [API管理页面](https://www.binance.com/en/my/settings/api-management)
3. 创建新的API Key
4. 设置适当的权限（推荐只开启读取权限进行测试）
5. 配置IP白名单（可选，但更安全）

## 故障排除

如果遇到问题，请：

1. 检查API密钥是否正确
2. 确认账户有足够权限
3. 检查网络连接
4. 查看币安API文档：[https://binance-docs.github.io/apidocs/futures/cn/](https://binance-docs.github.io/apidocs/futures/cn/)

## 技术细节

- 使用币安期货API v2 (`/fapi/v2/account`)
- 实现HMAC-SHA256签名认证
- 支持正式环境和测试环境切换
- 包含完整的错误处理和状态检查