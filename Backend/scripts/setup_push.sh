#!/bin/bash
# 推送通知功能服务器配置脚本
# 在服务器上执行: bash setup_push.sh

set -e

echo "=== 推送通知功能配置 ==="

# 1. 创建 keys 目录
echo "[1/4] 创建密钥目录..."
mkdir -p /opt/youkong/config/keys
chmod 700 /opt/youkong/config/keys

# 2. 执行数据库迁移
echo "[2/4] 执行数据库迁移..."
sudo mysql youkong << 'EOF'
-- 设备推送令牌表
CREATE TABLE IF NOT EXISTS device_tokens (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    user_id VARCHAR(36) NOT NULL,
    token VARCHAR(512) NOT NULL,
    platform ENUM('ios', 'android') NOT NULL,
    device_name VARCHAR(100),
    app_version VARCHAR(20),
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,

    UNIQUE KEY uk_token (token),
    INDEX idx_user_id (user_id),
    INDEX idx_platform (platform),
    INDEX idx_user_active (user_id, is_active),
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
EOF
echo "数据库迁移完成"

# 3. 添加推送配置到 .env
echo "[3/4] 更新环境变量配置..."
if ! grep -q "APNS_KEY_PATH" /opt/youkong/.env; then
    cat >> /opt/youkong/.env << 'EOF'

# Apple APNs (iOS 推送通知)
APNS_KEY_PATH=./config/keys/AuthKey_JKFD384UQ9.p8
APNS_KEY_ID=JKFD384UQ9
APNS_TEAM_ID=KQW6UNZE8J
APNS_BUNDLE_ID=com.YouKong
APNS_PRODUCTION=true

# 腾讯云 TPNS (Android 推送通知)
TPNS_ACCESS_ID=1580021617
TPNS_ACCESS_KEY=ABS41NO8GE9Z
TPNS_SECRET_KEY=e023d353613f35d296903b43eeddc850
EOF
    echo "环境变量已添加"
else
    echo "环境变量已存在，跳过"
fi

# 4. 提示上传密钥文件
echo "[4/4] 密钥文件配置..."
if [ ! -f /opt/youkong/config/keys/AuthKey_JKFD384UQ9.p8 ]; then
    echo ""
    echo "!!! 请手动上传 APNs 密钥文件 !!!"
    echo "本地文件: ~/Documents/AuthKey_JKFD384UQ9.p8"
    echo "目标路径: /opt/youkong/config/keys/AuthKey_JKFD384UQ9.p8"
    echo ""
    echo "上传命令 (在本地执行):"
    echo "scp ~/Documents/AuthKey_JKFD384UQ9.p8 root@49.232.13.41:/opt/youkong/config/keys/"
    echo ""
else
    echo "密钥文件已存在"
fi

echo ""
echo "=== 配置完成 ==="
echo "重启服务: systemctl restart youkong"
echo "查看日志: journalctl -u youkong -f"
