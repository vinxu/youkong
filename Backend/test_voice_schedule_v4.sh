#!/bin/bash

# V4 语音时刻表 API 测试脚本
# 测试新的简化架构：while 循环 + 工具调用

BASE_URL="${BASE_URL:-http://localhost:8080}"
TOKEN="${TOKEN:-your_token_here}"

echo "=========================================="
echo "V4 语音时刻表 API 测试"
echo "=========================================="
echo "Base URL: $BASE_URL"
echo ""

# 辅助函数：发送请求并显示 SSE 响应
send_request() {
    local text="$1"
    local session_id="${2:-}"

    echo ">>> 用户输入: $text"
    echo ""

    local body
    if [ -n "$session_id" ]; then
        body="{\"text\": \"$text\", \"session_id\": \"$session_id\"}"
    else
        body="{\"text\": \"$text\"}"
    fi

    curl -s -N \
        -H "Authorization: Bearer $TOKEN" \
        -H "Content-Type: application/json" \
        -d "$body" \
        "$BASE_URL/api/v1/agent/voice-schedule/v4/text" 2>&1 | while read -r line; do
        if [[ "$line" == data:* ]]; then
            echo "$line" | sed 's/^data: //'
        fi
    done
    echo ""
    echo "---"
    echo ""
}

# 测试用例 1: 创建时刻表
echo "========== 测试 1: 创建时刻表 =========="
send_request "明天下午3点到5点开会"

# 测试用例 2: 询问时间状态（应该用对话回答，不展示完整时刻表）
echo "========== 测试 2: 询问时间状态 =========="
send_request "两点以后什么状态"

# 测试用例 3: 更新当前状态
echo "========== 测试 3: 更新当前状态 =========="
send_request "我现在睡不着"

# 测试用例 4: 闲聊
echo "========== 测试 4: 闲聊 =========="
send_request "你好"

# 测试用例 5: 确认（需要先有 pending schedule）
echo "========== 测试 5: 创建并确认 =========="
echo "首先创建时刻表..."
send_request "明天上午9点到12点上班"
echo "然后确认..."
# 注意：需要传入相同的 session_id 才能确认
# 这里只是演示，实际需要从上一个响应中获取 session_id

echo ""
echo "=========================================="
echo "测试完成"
echo "=========================================="
echo ""
echo "使用方法："
echo "  1. 设置 TOKEN 环境变量（获取方法：登录后从响应中获取）"
echo "  2. 设置 BASE_URL 环境变量（默认 http://localhost:8080）"
echo "  3. 运行: ./test_voice_schedule_v4.sh"
echo ""
echo "示例："
echo "  TOKEN=eyJhbGciOiJIUzI1NiIs... ./test_voice_schedule_v4.sh"
