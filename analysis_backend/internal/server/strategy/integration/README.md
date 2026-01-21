# 策略系统集成测试

这个目录包含策略系统的完整集成测试套件，用于验证策略执行、扫描、路由器和工厂等组件之间的协同工作。

## 测试结构

```
integration/
├── test_server.go          # 测试服务器设置和基础设施
├── strategy_execution_test.go # 策略执行集成测试
├── strategy_scanning_test.go  # 策略扫描集成测试
├── router_factory_test.go     # 路由器和工厂集成测试
├── end_to_end_test.go         # 端到端完整流程测试
└── README.md                  # 本文档
```

## 测试套件说明

### 1. StrategyExecutionTestSuite - 策略执行测试
- **TestStrategyExecution_TraditionalStrategy**: 测试传统策略的完整执行流程
- **TestStrategyExecution_MovingAverageStrategy**: 测试均线策略执行
- **TestStrategyExecution_MeanReversionStrategy**: 测试均值回归策略执行
- **TestStrategyExecution_InvalidStrategyID**: 测试无效策略ID的错误处理
- **TestStrategyExecution_BatchExecution**: 测试批量执行（如果支持）
- **TestStrategyExecution_DisabledStrategy**: 测试禁用策略的执行
- **TestStrategyExecution_Timeout**: 测试执行超时处理
- **TestStrategyExecution_ConcurrentRequests**: 测试并发请求处理

### 2. StrategyScanningTestSuite - 策略扫描测试
- **TestStrategyScanning_TraditionalStrategyScan**: 测试传统策略扫描
- **TestStrategyScanning_MovingAverageStrategyScan**: 测试均线策略扫描
- **TestStrategyScanning_InvalidStrategyID**: 测试无效策略ID扫描
- **TestStrategyScanning_MissingStrategyID**: 测试缺少策略ID的情况
- **TestStrategyScanning_DisabledStrategy**: 测试禁用策略的扫描
- **TestStrategyScanning_Timeout**: 测试扫描超时
- **TestStrategyScanning_ConcurrentRequests**: 测试并发扫描请求
- **TestStrategyScanning_ResultFormat**: 测试扫描结果格式

### 3. RouterFactoryTestSuite - 路由器和工厂测试
- **TestRouterFactory_Integration**: 测试路由器和工厂的完整集成
- **TestRouterFactory_PriorityHandling**: 测试策略优先级处理
- **TestRouterFactory_ConfigValidation**: 测试配置验证
- **TestRouterFactory_ErrorHandling**: 测试错误处理
- **TestRouterFactory_AllRoutes**: 测试所有路由的工厂集成
- **TestRouterFactory_ConfigBuilders**: 测试配置构建器
- **TestRouterFactory_MarketDataBuilders**: 测试市场数据构建器
- **TestRouterFactory_ContextBuilders**: 测试上下文构建器

### 4. EndToEndTestSuite - 端到端测试
- **TestEndToEnd_StrategyLifecycle**: 测试完整的策略生命周期
- **TestEndToEnd_MultipleStrategies**: 测试多个策略的端到端流程
- **TestEndToEnd_ErrorScenarios**: 测试各种错误场景
- **TestEndToEnd_Performance**: 测试性能表现
- **TestEndToEnd_DataConsistency**: 测试数据一致性
- **TestEndToEnd_ResourceCleanup**: 测试资源清理

## 运行测试

### 运行所有集成测试
```bash
go test ./internal/server/strategy/integration/... -v
```

### 运行特定测试套件
```bash
# 策略执行测试
go test ./internal/server/strategy/integration/ -run TestStrategyExecutionSuite -v

# 策略扫描测试
go test ./internal/server/strategy/integration/ -run TestStrategyScanningSuite -v

# 路由器和工厂测试
go test ./internal/server/strategy/integration/ -run TestRouterFactorySuite -v

# 端到端测试
go test ./internal/server/strategy/integration/ -run TestEndToEndSuite -v
```

### 运行性能测试
```bash
go test ./internal/server/strategy/integration/ -run TestEndToEnd_Performance -v -bench=.
```

## 测试基础设施

### TestServer
`test_server.go` 提供完整的测试服务器基础设施：

- **内存SQLite数据库**: 用于测试的隔离数据库环境
- **Gin路由器**: 完整的HTTP路由设置
- **策略组件**: 预初始化的路由器、工厂等
- **测试数据创建**: 便捷的测试策略创建方法
- **HTTP请求模拟**: 简化的HTTP请求执行方法

### IntegrationTestSuite
所有测试套件的基类，提供：

- **自动设置和清理**: 每个测试的服务器初始化和清理
- **断言助手**: 常用的HTTP响应断言方法
- **异步操作等待**: 处理异步操作的工具方法

## 测试数据

测试使用以下类型的测试数据：

### 策略条件配置
- **传统策略**: 开空涨幅前10名，市值限制等
- **均线策略**: SMA均线，5/20周期交叉信号
- **均值回归策略**: 布林带配置，回归强度阈值
- **套利策略**: 三角套利，期现价差阈值
- **网格交易策略**: 网格上下限，网格级别

### 市场数据
- **测试货币对**: BTCUSDT
- **价格数据**: 50000.0 USDT
- **技术指标**: SMA、RSI等（根据需要）
- **排名数据**: 涨幅排名等

## 错误场景测试

集成测试涵盖以下错误场景：

1. **无效策略ID**: 404/500错误响应
2. **缺少必需参数**: 400错误响应
3. **配置验证失败**: 业务逻辑错误处理
4. **数据库错误**: 连接和查询错误
5. **并发访问**: 竞态条件和资源竞争
6. **超时处理**: 请求超时和取消

## 性能基准

集成测试包含性能基准测试：

- **并发请求处理**: 50个并发请求的平均响应时间
- **内存使用**: 测试期间的内存分配和GC
- **数据库查询**: SQL查询性能统计
- **组件初始化**: 策略组件的初始化时间

## 持续集成

这些测试设计用于CI/CD管道：

- **快速反馈**: 单元测试先运行，提供快速反馈
- **隔离执行**: 每个测试使用独立的数据库实例
- **资源清理**: 自动清理测试数据和资源
- **详细日志**: 失败时提供详细的错误信息和上下文

## 扩展测试

要添加新的集成测试：

1. **创建新的测试套件**: 继承`IntegrationTestSuite`
2. **实现测试方法**: 使用`Test*`前缀
3. **使用测试基础设施**: 利用`TestServer`和助手方法
4. **添加测试数据**: 根据需要创建测试策略和条件
5. **验证结果**: 使用断言验证HTTP响应和业务逻辑

## 故障排除

### 常见问题

1. **数据库连接错误**: 检查SQLite内存数据库初始化
2. **路由未找到**: 验证HTTP路径和方法
3. **组件未初始化**: 检查`TestServer`的初始化顺序
4. **并发测试失败**: 检查资源竞争和锁机制
5. **性能测试超时**: 调整超时时间或优化代码

### 调试技巧

1. **启用详细日志**: 设置日志级别为DEBUG
2. **检查HTTP响应**: 打印完整的响应体和状态码
3. **数据库查询**: 验证测试数据的创建和查询
4. **组件状态**: 检查策略路由器和工厂的状态
5. **竞态条件**: 使用`-race`标志检测数据竞争