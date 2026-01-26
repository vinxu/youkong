package com.youkong.feature.auth.screen

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.imePadding
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.text.BasicTextField
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Scaffold
import androidx.compose.material3.Surface
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.remember
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.focus.FocusRequester
import androidx.compose.ui.focus.focusRequester
import androidx.compose.ui.text.input.KeyboardType
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.unit.dp
import androidx.hilt.navigation.compose.hiltViewModel
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import com.youkong.core.ui.component.YouKongLoading
import com.youkong.core.ui.component.YouKongTextButton
import com.youkong.core.ui.component.YouKongTopBar
import com.youkong.core.ui.theme.Border
import com.youkong.core.ui.theme.Error
import com.youkong.core.ui.theme.Primary
import com.youkong.core.ui.theme.TextPrimary
import com.youkong.core.ui.theme.TextSecondary
import com.youkong.core.ui.theme.YouKongShapes
import com.youkong.core.ui.theme.YouKongTheme
import com.youkong.feature.auth.viewmodel.SmsVerificationViewModel

@Composable
fun SmsVerificationScreen(
    phone: String,
    onBackClick: () -> Unit,
    onLoginSuccess: (Boolean) -> Unit,
    viewModel: SmsVerificationViewModel = hiltViewModel(),
) {
    val uiState by viewModel.uiState.collectAsStateWithLifecycle()
    val focusRequester = remember { FocusRequester() }

    LaunchedEffect(Unit) {
        focusRequester.requestFocus()
    }

    LaunchedEffect(uiState.loginSuccess) {
        if (uiState.loginSuccess) {
            onLoginSuccess(uiState.isNewUser)
        }
    }

    Scaffold(
        topBar = {
            YouKongTopBar(
                title = "验证码",
                onBackClick = onBackClick,
            )
        }
    ) { innerPadding ->
        if (uiState.isLoading) {
            YouKongLoading(message = "验证中...")
        } else {
            Column(
                modifier = Modifier
                    .fillMaxSize()
                    .padding(innerPadding)
                    .padding(horizontal = YouKongTheme.spacing.xxl)
                    .imePadding(),
                horizontalAlignment = Alignment.CenterHorizontally,
            ) {
                Spacer(modifier = Modifier.height(48.dp))

                Text(
                    text = "验证码已发送至",
                    style = MaterialTheme.typography.bodyMedium,
                    color = TextSecondary,
                )

                Spacer(modifier = Modifier.height(8.dp))

                Text(
                    text = phone.replaceRange(3, 7, "****"),
                    style = MaterialTheme.typography.titleLarge,
                    color = TextPrimary,
                )

                Spacer(modifier = Modifier.height(48.dp))

                // 验证码输入框
                BasicTextField(
                    value = uiState.code,
                    onValueChange = viewModel::onCodeChange,
                    modifier = Modifier.focusRequester(focusRequester),
                    keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Number),
                    decorationBox = {
                        Row(
                            horizontalArrangement = Arrangement.spacedBy(8.dp),
                        ) {
                            repeat(6) { index ->
                                CodeDigitBox(
                                    digit = uiState.code.getOrNull(index)?.toString() ?: "",
                                    isFocused = uiState.code.length == index,
                                )
                            }
                        }
                    }
                )

                if (uiState.error != null) {
                    Spacer(modifier = Modifier.height(16.dp))
                    Text(
                        text = uiState.error!!,
                        style = MaterialTheme.typography.bodySmall,
                        color = Error,
                    )
                }

                Spacer(modifier = Modifier.height(32.dp))

                if (uiState.canResend) {
                    YouKongTextButton(
                        text = "重新发送验证码",
                        onClick = viewModel::resendCode,
                    )
                } else {
                    Text(
                        text = "${uiState.countdown}秒后可重新发送",
                        style = MaterialTheme.typography.bodyMedium,
                        color = TextSecondary,
                    )
                }
            }
        }
    }
}

@Composable
private fun CodeDigitBox(
    digit: String,
    isFocused: Boolean,
) {
    Surface(
        modifier = Modifier.size(48.dp),
        shape = YouKongShapes.medium,
        color = if (isFocused) Primary.copy(alpha = 0.1f) else MaterialTheme.colorScheme.surface,
        border = androidx.compose.foundation.BorderStroke(
            width = if (isFocused) 2.dp else 1.dp,
            color = if (isFocused) Primary else Border,
        ),
    ) {
        Text(
            text = digit,
            style = MaterialTheme.typography.headlineMedium,
            color = TextPrimary,
            textAlign = TextAlign.Center,
            modifier = Modifier
                .fillMaxSize()
                .padding(top = 8.dp),
        )
    }
}
