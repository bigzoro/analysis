#!/bin/bash

# 策略系统集成测试运行脚本
# 用于在CI/CD环境中运行完整的集成测试套件

set -e

echo "🚀 开始运行策略系统集成测试..."

# 设置Go环境
export GO111MODULE=on
export CGO_ENABLED=1

# 进入项目目录
cd "$(dirname "$0")/.."

# 颜色输出
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 日志函数
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# 检查依赖
check_dependencies() {
    log_info "检查测试依赖..."

    if ! command -v go &> /dev/null; then
        log_error "Go未安装或不在PATH中"
        exit 1
    fi

    # 检查Go版本
    GO_VERSION=$(go version | grep -oE 'go[0-9]+\.[0-9]+')
    log_info "Go版本: $GO_VERSION"

    # 检查是否有SQLite支持
    if ! go env CGO_ENABLED | grep -q "1"; then
        log_warning "CGO未启用，可能影响SQLite测试"
    fi
}

# 运行单元测试（作为集成测试的前提）
run_unit_tests() {
    log_info "运行单元测试..."

    if go test ./internal/server/strategy/router ./internal/server/strategy/factory \
              ./internal/server/strategy/traditional/execution \
              ./internal/server/strategy/moving_average/execution \
              -v -short; then
        log_success "单元测试通过"
    else
        log_error "单元测试失败，跳过集成测试"
        exit 1
    fi
}

# 运行集成测试套件
run_integration_tests() {
    log_info "运行集成测试套件..."

    local test_suites=(
        "TestStrategyExecutionSuite"
        "TestStrategyScanningSuite"
        "TestRouterFactorySuite"
        "TestEndToEndSuite"
    )

    local failed_suites=()

    for suite in "${test_suites[@]}"; do
        log_info "运行测试套件: $suite"

        if go test ./internal/server/strategy/integration/ -run "$suite" -v -timeout 30s; then
            log_success "测试套件 $suite 通过"
        else
            log_error "测试套件 $suite 失败"
            failed_suites+=("$suite")
        fi
    done

    # 报告失败的套件
    if [ ${#failed_suites[@]} -ne 0 ]; then
        log_error "以下测试套件失败:"
        for suite in "${failed_suites[@]}"; do
            echo "  - $suite"
        done
        exit 1
    fi
}

# 运行性能基准测试
run_performance_tests() {
    log_info "运行性能基准测试..."

    if go test ./internal/server/strategy/integration/ -run TestEndToEnd_Performance -bench=. -benchmem -v; then
        log_success "性能测试完成"
    else
        log_warning "性能测试失败，但不影响主要功能"
    fi
}

# 运行竞态检测
run_race_tests() {
    log_info "运行竞态检测..."

    if go test ./internal/server/strategy/integration/ -run TestStrategyExecution_ConcurrentRequests -race -v; then
        log_success "竞态检测通过"
    else
        log_warning "发现竞态条件，需要进一步调查"
    fi
}

# 生成测试覆盖率报告
generate_coverage_report() {
    log_info "生成测试覆盖率报告..."

    # 创建覆盖率目录
    mkdir -p coverage

    # 运行集成测试并生成覆盖率
    if go test ./internal/server/strategy/integration/ \
              -coverprofile=coverage/integration.out \
              -covermode=atomic \
              -v; then

        # 生成HTML报告
        go tool cover -html=coverage/integration.out -o coverage/integration.html

        # 显示覆盖率统计
        go tool cover -func=coverage/integration.out

        log_success "覆盖率报告生成: coverage/integration.html"
    else
        log_warning "覆盖率测试失败"
    fi
}

# 清理测试数据
cleanup() {
    log_info "清理测试数据..."

    # 清理覆盖率文件（如果需要）
    # rm -f coverage/integration.out

    log_success "清理完成"
}

# 主函数
main() {
    log_info "开始策略系统集成测试流程"

    # 陷阱：确保在脚本退出时运行清理
    trap cleanup EXIT

    # 执行测试步骤
    check_dependencies
    run_unit_tests
    run_integration_tests
    run_performance_tests
    run_race_tests
    generate_coverage_report

    log_success "🎉 所有集成测试完成！"
}

# 参数处理
case "${1:-}" in
    "unit")
        log_info "仅运行单元测试"
        check_dependencies
        run_unit_tests
        ;;
    "integration")
        log_info "仅运行集成测试"
        check_dependencies
        run_integration_tests
        ;;
    "performance")
        log_info "仅运行性能测试"
        check_dependencies
        run_performance_tests
        ;;
    "race")
        log_info "仅运行竞态检测"
        check_dependencies
        run_race_tests
        ;;
    "coverage")
        log_info "仅生成覆盖率报告"
        check_dependencies
        generate_coverage_report
        ;;
    "help"|"-h"|"--help")
        echo "策略系统集成测试脚本"
        echo ""
        echo "用法: $0 [选项]"
        echo ""
        echo "选项:"
        echo "  (无参数)    运行完整测试流程"
        echo "  unit        仅运行单元测试"
        echo "  integration 仅运行集成测试"
        echo "  performance 仅运行性能测试"
        echo "  race        仅运行竞态检测"
        echo "  coverage    仅生成覆盖率报告"
        echo "  help        显示此帮助信息"
        echo ""
        echo "环境变量:"
        echo "  GO111MODULE  Go模块模式 (默认: on)"
        echo "  CGO_ENABLED  CGO启用状态 (默认: 1)"
        ;;
    *)
        main
        ;;
esac