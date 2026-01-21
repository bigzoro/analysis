# Investment Scanner - 智能投资扫描器

Investment Scanner 是一个独立的命令行工具，用于执行智能投研、策略回测和表现验证等投资分析任务。它通过API调用与主服务器通信，不直接访问数据库。

## 功能特性

- **表现验证更新**: 直接管理推荐表现追踪调度器
- **策略回测**: 通过API执行历史策略回测分析
- **批量策略测试**: 批量测试推荐记录的策略表现
- **投资分析报告生成**: 生成详细的投资分析报告
- **定时任务**: 支持持续运行模式和调度器模式
- **智能调度**: 内置SmartScheduler进行工作负载管理
- **预计算处理**: PrecomputeProcessor处理推荐缓存预计算任务
- **轻量级部署**: 无需数据库连接，通过API工作

## 编译方式

### 使用PowerShell脚本（推荐）
```powershell
.\build_investment.ps1
```

### 手动编译
```bash
go build -o investment.exe cmd/investment/main.go
```

### 验证编译
```bash
# 查看帮助信息
investment.exe -h

# 测试单次运行（需要API服务器运行）
investment.exe -api=http://127.0.0.1:8010
```

## 服务管理

Investment Scanner 支持后台服务模式运行，提供专门的启动和停止脚本。

### Linux/macOS 环境

#### 启动服务
```bash
# 使用默认配置启动
./start_investment.sh

# 使用自定义配置启动
CONFIG=/path/to/config.yaml ./start_investment.sh

# 指定API地址和额外参数
INV_API=http://your-api-server:8010 INV_OPTS='-interval 600s -mode continuous' ./start_investment.sh
```

#### 停止服务
```bash
./stop_investment.sh
```

#### 查看日志
```bash
# 标准输出日志
tail -f logs/investment.out

# 错误日志
tail -f logs/investment.err
```

### Windows 环境

#### 启动服务
```powershell
# 使用默认配置启动
.\investment.exe -config config.yaml -api http://127.0.0.1:8010 -mode continuous

# 或创建批处理文件 start_investment.bat
@echo off
if not exist logs mkdir logs
if not exist run mkdir run
investment.exe -config config.yaml -api http://127.0.0.1:8010 -mode continuous > logs\investment.out 2> logs\investment.err
```

#### 停止服务
```powershell
# 通过任务管理器或进程管理器停止 investment.exe 进程
# 或使用 stop_investment.bat（需要实现PID管理）
```

### 服务管理环境变量

| 变量 | 默认值 | 说明 |
|------|--------|------|
| `CONFIG` | `./config.yaml` | 配置文件路径 |
| `INV_API` | `http://127.0.0.1:8010` | API服务器地址 |
| `INV_OPTS` | 空 | 额外命令行参数 |

### 后台运行说明

- **Linux/macOS**: 使用 `start_investment.sh` 脚本在后台运行，PID保存在 `run/investment.pid`
- **Windows**: 推荐使用计划任务或服务管理器实现后台运行
- 日志文件保存在 `logs/` 目录
- 运行状态保存在 `run/` 目录

## 使用方法

### 基本语法
```bash
investment.exe [选项]
```

### 运行模式

#### 1. 单次运行模式（默认）
执行一次完整的投资分析任务，包括表现数据更新和策略测试。

```bash
# 使用默认配置
investment.exe

# 指定配置文件
investment.exe -config=./config.yaml
```

#### 2. 持续运行模式
定期执行投资分析任务，适用于长期运行的服务。

```bash
# 每10分钟执行一次
investment.exe -mode=continuous -interval=10m

# 每1小时执行一次
investment.exe -mode=continuous -interval=1h
```

#### 3. 策略回测模式
对特定交易对执行历史策略回测。

```bash
# 买入持有策略回测
investment.exe -mode=backtest -symbol=BTCUSDT -strategy=buy_and_hold -start-date=2024-01-01 -end-date=2024-12-31

# 机器学习策略回测
investment.exe -mode=backtest -symbol=ETHUSDT -strategy=ml_prediction -start-date=2024-06-01 -end-date=2024-12-31
```

#### 4. 策略测试模式
对单个推荐记录执行策略测试。

```bash
investment.exe -mode=strategy -performance-id=123
```

#### 5. 报告生成模式
生成详细的投资分析报告。

```bash
# 生成汇总报告
investment.exe -mode=report -performance-id=123 -report-type=summary

# 生成详细报告
investment.exe -mode=report -performance-id=123 -report-type=detailed

# 生成对比报告
investment.exe -mode=report -performance-id=123 -report-type=comparison
```

#### 6. 调度器模式（新增）
启动内置的调度器，包括PerformanceTracker和SmartScheduler。

```bash
# 启动调度器模式（推荐替代API内置调度器）
investment.exe -mode=scheduler
```

## 命令行选项

| 选项 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| `-config` | string | `./config.yaml` | 配置文件路径 |
| `-mode` | string | `once` | 运行模式: `once`, `continuous`, `backtest`, `strategy`, `report`, `scheduler` |
| `-interval` | duration | `10m` | 持续模式下的运行间隔 |
| `-performance-id` | string | 空 | 单个表现验证ID（策略测试和报告模式使用） |
| `-symbol` | string | 空 | 交易对符号（回测模式使用） |
| `-strategy` | string | `buy_and_hold` | 回测策略类型 |
| `-start-date` | string | 空 | 回测开始日期 (YYYY-MM-DD) |
| `-end-date` | string | 空 | 回测结束日期 (YYYY-MM-DD) |
| `-report-type` | string | `summary` | 报告类型: `summary`, `detailed`, `comparison` |
| `-output` | string | 空 | 报告输出路径 |

## 支持的策略类型

- `buy_and_hold`: 买入持有策略
- `ml_prediction`: 机器学习预测策略
- `ensemble`: 集成学习策略

## 配置要求

需要有效的配置文件（主要用于代理设置），参考 `config.yaml` 文件。

## API依赖

Investment Scanner需要与运行中的API服务器通信，确保以下API端点可用：

- `POST /recommendations/performance/batch-update` - 批量更新验证记录
- `POST /recommendations/performance/batch-strategy-test` - 批量策略测试
- `POST /recommendations/performance/{id}/strategy-test` - 单个策略测试
- `POST /recommendations/performance/{id}/report` - 生成投资分析报告

## 输出示例

### 回测结果输出
```
=== 回测结果 ===
策略: buy_and_hold
交易对: BTCUSDT
时间范围: 2024-01-01 -> 2024-12-31
总收益率: 45.67%
年化收益率: 12.34%
胜率: 55.23%
最大回撤: -15.42%
夏普比率: 1.23
总交易次数: 245
```

### 策略测试结果输出
```
=== 策略测试结果 ===
记录ID: 123
币种: BTCUSDT
推荐价格: 45000.00000000
入场价格: 45000.00000000
出场价格: 46500.00000000
策略收益: 3.33%
持有时间: 1440 分钟
退出原因: profit
```

## 日志说明

程序会输出详细的运行日志，包括：
- 任务开始/结束时间
- 处理的记录数量
- 成功/失败统计
- 详细的错误信息

## 注意事项

1. **调度器模式**: 使用 `-mode=scheduler` 启动内置调度器，替代API服务内置的调度功能
2. 确保数据库连接正常
3. 大量数据处理时可能需要较长时间
4. 建议在非交易高峰期运行批量任务
5. 定期检查日志文件，确保程序正常运行
6. **架构变更**: PerformanceTracker和SmartScheduler已从API服务移动到investment服务

## 故障排除

### 数据库连接失败
检查配置文件中的数据库连接参数是否正确。

### 策略测试失败
检查推荐记录是否存在，以及是否有有效的价格数据。

### 回测数据不足
检查指定的时间范围内是否有足够的历史数据。

### 调度器启动失败
检查是否已有其他investment进程在运行调度器模式，避免冲突。

## 开发说明

如需添加新的投资分析功能：

1. 在 `analysis` 包中实现新的分析逻辑
2. 在 `main.go` 中添加新的命令行选项
3. 在相应的处理函数中调用新的分析功能
4. 更新此README文档
