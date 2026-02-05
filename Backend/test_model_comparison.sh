#!/bin/bash

# Qwen vs Kimi 2.5 模型对比测试脚本
# 使用方法: ./test_model_comparison.sh [选项]
#
# 选项:
#   --single <case_id> <provider>  运行单个测试
#   --category <category> <provider>  按分类运行测试
#   --full  运行完整对比测试
#   --list  列出所有测试用例
#   --report <json_file>  从 JSON 生成 Markdown 报告

set -e

# 配置
API_BASE="${API_BASE:-http://localhost:8080}"
AUTH_TOKEN="${AUTH_TOKEN:-}"  # 可选：如果 API 需要认证

# 颜色输出
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

print_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# 检查依赖
check_dependencies() {
    if ! command -v curl &> /dev/null; then
        print_error "curl 未安装"
        exit 1
    fi
    if ! command -v jq &> /dev/null; then
        print_warning "jq 未安装，JSON 输出可能不太美观"
    fi
}

# 构建请求头
build_headers() {
    local headers="-H 'Content-Type: application/json'"
    if [ -n "$AUTH_TOKEN" ]; then
        headers="$headers -H 'Authorization: Bearer $AUTH_TOKEN'"
    fi
    echo "$headers"
}

# 列出测试用例
list_cases() {
    print_info "获取测试用例列表..."

    local response
    response=$(curl -s "${API_BASE}/api/v1/agent/test/model/cases" \
        -H "Content-Type: application/json" \
        ${AUTH_TOKEN:+-H "Authorization: Bearer $AUTH_TOKEN"})

    if command -v jq &> /dev/null; then
        echo "$response" | jq '.data.cases[] | "\(.id). [\(.category)] \(.input) - \(.description)"'
    else
        echo "$response"
    fi
}

# 运行单个测试
run_single_test() {
    local case_id=$1
    local provider=$2

    if [ -z "$case_id" ] || [ -z "$provider" ]; then
        print_error "用法: ./test_model_comparison.sh --single <case_id> <provider>"
        print_info "provider 可选值: qwen, kimi"
        exit 1
    fi

    print_info "运行测试 #${case_id} (${provider})..."

    local response
    response=$(curl -s -X POST "${API_BASE}/api/v1/agent/test/model/single" \
        -H "Content-Type: application/json" \
        ${AUTH_TOKEN:+-H "Authorization: Bearer $AUTH_TOKEN"} \
        -d "{\"case_id\": ${case_id}, \"provider\": \"${provider}\"}")

    if command -v jq &> /dev/null; then
        echo "$response" | jq '.'
    else
        echo "$response"
    fi
}

# 按分类运行测试
run_category_test() {
    local category=$1
    local provider=$2

    if [ -z "$category" ] || [ -z "$provider" ]; then
        print_error "用法: ./test_model_comparison.sh --category <category> <provider>"
        print_info "分类: create_schedule, confirm, modify, query_schedule, update_status, chat, constraint"
        exit 1
    fi

    print_info "运行分类测试: ${category} (${provider})..."

    local response
    response=$(curl -s -X POST "${API_BASE}/api/v1/agent/test/model/category" \
        -H "Content-Type: application/json" \
        ${AUTH_TOKEN:+-H "Authorization: Bearer $AUTH_TOKEN"} \
        -d "{\"category\": \"${category}\", \"provider\": \"${provider}\"}")

    if command -v jq &> /dev/null; then
        echo "$response" | jq '.'
    else
        echo "$response"
    fi
}

# 运行完整对比测试
run_full_comparison() {
    print_info "开始完整模型对比测试..."
    print_warning "这可能需要较长时间（约 10-30 分钟）"

    local start_time=$(date +%s)

    local response
    response=$(curl -s -X POST "${API_BASE}/api/v1/agent/test/model/comparison" \
        -H "Content-Type: application/json" \
        ${AUTH_TOKEN:+-H "Authorization: Bearer $AUTH_TOKEN"} \
        --max-time 1800)  # 30 分钟超时

    local end_time=$(date +%s)
    local duration=$((end_time - start_time))

    print_success "测试完成，耗时 ${duration} 秒"

    # 保存 JSON 结果
    local timestamp=$(date +%Y%m%d_%H%M%S)
    local json_file="model_comparison_${timestamp}.json"
    echo "$response" > "$json_file"
    print_info "JSON 结果已保存到: $json_file"

    # 显示摘要
    if command -v jq &> /dev/null; then
        echo ""
        echo "========== 测试摘要 =========="
        echo "Qwen 综合得分: $(echo "$response" | jq '.data.qwen_summary.overall_score')"
        echo "Kimi 综合得分: $(echo "$response" | jq '.data.kimi_summary.overall_score')"
        echo "推荐: $(echo "$response" | jq -r '.data.recommendation')"
    fi

    # 生成 Markdown 报告
    print_info "生成 Markdown 报告..."
    local report_response
    report_response=$(curl -s -X POST "${API_BASE}/api/v1/agent/test/model/report" \
        -H "Content-Type: application/json" \
        ${AUTH_TOKEN:+-H "Authorization: Bearer $AUTH_TOKEN"} \
        -d "{\"report\": $(echo "$response" | jq '.data')}")

    local md_file="MODEL_COMPARISON_REPORT_${timestamp}.md"
    echo "$report_response" > "$md_file"
    print_success "Markdown 报告已保存到: $md_file"
}

# 从 JSON 生成报告
generate_report() {
    local json_file=$1

    if [ -z "$json_file" ] || [ ! -f "$json_file" ]; then
        print_error "用法: ./test_model_comparison.sh --report <json_file>"
        exit 1
    fi

    print_info "从 $json_file 生成报告..."

    local report_data
    report_data=$(cat "$json_file" | jq '.data // .')

    local response
    response=$(curl -s -X POST "${API_BASE}/api/v1/agent/test/model/report" \
        -H "Content-Type: application/json" \
        ${AUTH_TOKEN:+-H "Authorization: Bearer $AUTH_TOKEN"} \
        -d "{\"report\": ${report_data}}")

    local md_file="${json_file%.json}.md"
    echo "$response" > "$md_file"
    print_success "报告已保存到: $md_file"
}

# 快速测试（验证 API 是否正常）
quick_test() {
    print_info "运行快速测试..."

    # 测试 Qwen
    print_info "测试 Qwen (case #1)..."
    run_single_test 1 "qwen"

    echo ""

    # 测试 Kimi
    print_info "测试 Kimi (case #1)..."
    run_single_test 1 "kimi"
}

# 显示帮助
show_help() {
    echo "Qwen vs Kimi 2.5 模型对比测试脚本"
    echo ""
    echo "用法: ./test_model_comparison.sh [选项]"
    echo ""
    echo "选项:"
    echo "  --list                          列出所有测试用例"
    echo "  --single <case_id> <provider>   运行单个测试"
    echo "  --category <category> <provider> 按分类运行测试"
    echo "  --full                          运行完整对比测试"
    echo "  --quick                         快速测试（验证 API）"
    echo "  --report <json_file>            从 JSON 生成 Markdown 报告"
    echo "  --help                          显示帮助"
    echo ""
    echo "Provider: qwen, kimi"
    echo ""
    echo "分类:"
    echo "  create_schedule  时刻表创建"
    echo "  confirm         确认操作"
    echo "  modify          修改操作"
    echo "  query_schedule  查询时刻表"
    echo "  update_status   更新状态"
    echo "  chat            闲聊/边界"
    echo "  constraint      约束输出验证"
    echo ""
    echo "环境变量:"
    echo "  API_BASE     API 地址 (默认: http://localhost:8080)"
    echo "  AUTH_TOKEN   认证 Token (可选)"
    echo ""
    echo "示例:"
    echo "  ./test_model_comparison.sh --list"
    echo "  ./test_model_comparison.sh --single 1 qwen"
    echo "  ./test_model_comparison.sh --category create_schedule kimi"
    echo "  ./test_model_comparison.sh --full"
}

# 主函数
main() {
    check_dependencies

    case "$1" in
        --list)
            list_cases
            ;;
        --single)
            run_single_test "$2" "$3"
            ;;
        --category)
            run_category_test "$2" "$3"
            ;;
        --full)
            run_full_comparison
            ;;
        --quick)
            quick_test
            ;;
        --report)
            generate_report "$2"
            ;;
        --help|-h|"")
            show_help
            ;;
        *)
            print_error "未知选项: $1"
            show_help
            exit 1
            ;;
    esac
}

main "$@"
