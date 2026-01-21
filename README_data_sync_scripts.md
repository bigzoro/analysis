# Data Sync 服务启动脚本使用说明

## 脚本文件

- `start_data_sync.sh` - 启动数据同步服务
- `stop_data_sync.sh` - 停止数据同步服务

## 环境变量配置

可以通过设置以下环境变量来自定义行为：

```bash
export DATA_SYNC_ACTION="start"           # 操作类型: start(启动服务), test-sync(测试同步), sync-once(单次同步), status(状态查询)
export DATA_SYNC_CONFIG_FILE="./sync_config.yaml"  # 同步服务配置文件路径
export DATA_SYNC_SYNCER="mysql"          # 指定特定的同步器（可选）
export CONFIG="./config.yaml"            # 主配置文件路径
```

## 使用示例

### 1. 启动服务（默认配置）
```bash
./start_data_sync.sh
```

### 2. 启动并指定同步器
```bash
DATA_SYNC_SYNCER="mysql" ./start_data_sync.sh
```

### 3. 测试同步器
```bash
DATA_SYNC_ACTION="test-sync" ./start_data_sync.sh
```

### 4. 单次同步
```bash
DATA_SYNC_ACTION="sync-once" ./start_data_sync.sh
```

### 5. 停止服务
```bash
./stop_data_sync.sh
```

## 日志文件

- 标准输出：`logs/data_sync.out`
- 错误输出：`logs/data_sync.err`
- 进程PID：`run/data_sync.pid`

## 注意事项

- 确保 `data_sync` 二进制文件存在且具有执行权限
- 脚本会在后台运行，不会阻塞终端
- 使用 `stop_data_sync.sh` 可以优雅地停止服务