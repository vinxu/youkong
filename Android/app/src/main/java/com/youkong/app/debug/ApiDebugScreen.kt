package com.youkong.app.debug

import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.Button
import androidx.compose.material3.ButtonDefaults
import androidx.compose.material3.Card
import androidx.compose.material3.CardDefaults
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Surface
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.text.font.FontFamily
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import androidx.hilt.navigation.compose.hiltViewModel
import kotlinx.coroutines.launch

@Composable
fun ApiDebugScreen(
    viewModel: ApiDebugViewModel = hiltViewModel(),
    modifier: Modifier = Modifier,
) {
    var phone by remember { mutableStateOf("13800000001") }
    var code by remember { mutableStateOf("111111") }
    var logText by remember { mutableStateOf("API 调试日志:\n") }
    val scope = rememberCoroutineScope()

    fun log(message: String) {
        logText += "\n$message"
    }

    Surface(
        modifier = modifier.fillMaxSize(),
        color = MaterialTheme.colorScheme.background
    ) {
        Column(
            modifier = Modifier
                .fillMaxSize()
                .padding(16.dp)
        ) {
            Text(
                text = "YouKong API 调试",
                style = MaterialTheme.typography.headlineMedium,
                color = MaterialTheme.colorScheme.primary
            )

            Spacer(modifier = Modifier.height(16.dp))

            // 输入区域
            Card(
                modifier = Modifier.fillMaxWidth(),
                colors = CardDefaults.cardColors(
                    containerColor = MaterialTheme.colorScheme.surfaceVariant
                )
            ) {
                Column(modifier = Modifier.padding(16.dp)) {
                    OutlinedTextField(
                        value = phone,
                        onValueChange = { phone = it },
                        label = { Text("手机号") },
                        modifier = Modifier.fillMaxWidth()
                    )
                    Spacer(modifier = Modifier.height(8.dp))
                    OutlinedTextField(
                        value = code,
                        onValueChange = { code = it },
                        label = { Text("验证码") },
                        modifier = Modifier.fillMaxWidth()
                    )
                }
            }

            Spacer(modifier = Modifier.height(16.dp))

            // 认证接口按钮
            Text(
                text = "认证接口",
                style = MaterialTheme.typography.titleMedium,
                modifier = Modifier.padding(bottom = 8.dp)
            )
            Row(
                horizontalArrangement = Arrangement.spacedBy(8.dp),
                modifier = Modifier.fillMaxWidth()
            ) {
                Button(
                    onClick = {
                        scope.launch {
                            log(">>> 发送验证码: $phone")
                            viewModel.sendSms(phone) { result ->
                                log("<<< $result")
                            }
                        }
                    },
                    modifier = Modifier.weight(1f)
                ) {
                    Text("发送验证码")
                }
                Button(
                    onClick = {
                        scope.launch {
                            log(">>> 验证登录: $phone, $code")
                            viewModel.verifySms(phone, code) { result ->
                                log("<<< $result")
                            }
                        }
                    },
                    modifier = Modifier.weight(1f)
                ) {
                    Text("验证登录")
                }
            }

            Spacer(modifier = Modifier.height(8.dp))

            // 用户接口按钮
            Text(
                text = "用户接口",
                style = MaterialTheme.typography.titleMedium,
                modifier = Modifier.padding(bottom = 8.dp)
            )
            Row(
                horizontalArrangement = Arrangement.spacedBy(8.dp),
                modifier = Modifier.fillMaxWidth()
            ) {
                Button(
                    onClick = {
                        scope.launch {
                            log(">>> 获取当前用户")
                            viewModel.getCurrentUser { result ->
                                log("<<< $result")
                            }
                        }
                    },
                    modifier = Modifier.weight(1f)
                ) {
                    Text("获取用户")
                }
                Button(
                    onClick = {
                        scope.launch {
                            log(">>> 搜索用户: test")
                            viewModel.searchUsers("test") { result ->
                                log("<<< $result")
                            }
                        }
                    },
                    modifier = Modifier.weight(1f)
                ) {
                    Text("搜索用户")
                }
            }

            Spacer(modifier = Modifier.height(8.dp))

            // 圈子接口按钮
            Text(
                text = "圈子接口",
                style = MaterialTheme.typography.titleMedium,
                modifier = Modifier.padding(bottom = 8.dp)
            )
            Row(
                horizontalArrangement = Arrangement.spacedBy(8.dp),
                modifier = Modifier.fillMaxWidth()
            ) {
                Button(
                    onClick = {
                        scope.launch {
                            log(">>> 获取圈子列表")
                            viewModel.getCircles { result ->
                                log("<<< $result")
                            }
                        }
                    },
                    modifier = Modifier.weight(1f)
                ) {
                    Text("获取圈子")
                }
                Button(
                    onClick = {
                        scope.launch {
                            log(">>> 创建圈子")
                            viewModel.createCircle { result ->
                                log("<<< $result")
                            }
                        }
                    },
                    modifier = Modifier.weight(1f)
                ) {
                    Text("创建圈子")
                }
            }

            Spacer(modifier = Modifier.height(8.dp))

            // 有空接口按钮
            Text(
                text = "有空接口",
                style = MaterialTheme.typography.titleMedium,
                modifier = Modifier.padding(bottom = 8.dp)
            )
            Row(
                horizontalArrangement = Arrangement.spacedBy(8.dp),
                modifier = Modifier.fillMaxWidth()
            ) {
                Button(
                    onClick = {
                        scope.launch {
                            log(">>> 获取好友有空")
                            viewModel.getFriendsAvailabilities { result ->
                                log("<<< $result")
                            }
                        }
                    },
                    modifier = Modifier.weight(1f)
                ) {
                    Text("好友有空")
                }
                Button(
                    onClick = {
                        scope.launch {
                            log(">>> 获取我的有空")
                            viewModel.getMyAvailabilities { result ->
                                log("<<< $result")
                            }
                        }
                    },
                    modifier = Modifier.weight(1f)
                ) {
                    Text("我的有空")
                }
            }

            Spacer(modifier = Modifier.height(8.dp))

            // 清除按钮
            Row(
                horizontalArrangement = Arrangement.spacedBy(8.dp),
                modifier = Modifier.fillMaxWidth()
            ) {
                Button(
                    onClick = { logText = "API 调试日志:\n" },
                    colors = ButtonDefaults.buttonColors(
                        containerColor = Color.Gray
                    ),
                    modifier = Modifier.weight(1f)
                ) {
                    Text("清除日志")
                }
                Button(
                    onClick = {
                        scope.launch {
                            viewModel.clearToken()
                            log(">>> Token 已清除")
                        }
                    },
                    colors = ButtonDefaults.buttonColors(
                        containerColor = Color.Red
                    ),
                    modifier = Modifier.weight(1f)
                ) {
                    Text("清除Token")
                }
            }

            Spacer(modifier = Modifier.height(16.dp))

            // 日志显示区域
            Card(
                modifier = Modifier
                    .fillMaxWidth()
                    .weight(1f),
                colors = CardDefaults.cardColors(
                    containerColor = Color(0xFF1E1E1E)
                )
            ) {
                Text(
                    text = logText,
                    color = Color(0xFF4EC9B0),
                    fontFamily = FontFamily.Monospace,
                    fontSize = 12.sp,
                    modifier = Modifier
                        .fillMaxSize()
                        .padding(12.dp)
                        .verticalScroll(rememberScrollState())
                )
            }
        }
    }
}
