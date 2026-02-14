package com.youkong.feature.auth.screen

import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
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
import androidx.compose.material3.Scaffold
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.remember
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.focus.FocusRequester
import androidx.compose.ui.focus.focusRequester
import androidx.compose.ui.graphics.SolidColor
import androidx.compose.ui.text.TextStyle
import androidx.compose.ui.text.font.FontFamily
import androidx.compose.ui.text.input.KeyboardType
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import androidx.hilt.navigation.compose.hiltViewModel
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import com.youkong.core.ui.component.cli.TerminalButton
import com.youkong.core.ui.component.cli.TerminalButtonStyle
import com.youkong.core.ui.theme.ASCII
import com.youkong.core.ui.theme.CLIColors
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
        kotlinx.coroutines.delay(100)
        try {
            focusRequester.requestFocus()
        } catch (_: IllegalStateException) {
            // 华为等设备布局时序不同，安全忽略
        }
    }

    LaunchedEffect(uiState.loginSuccess) {
        if (uiState.loginSuccess) {
            onLoginSuccess(uiState.isNewUser)
        }
    }

    Scaffold(
        containerColor = CLIColors.Background
    ) { innerPadding ->
        Column(
            modifier = Modifier
                .fillMaxSize()
                .background(CLIColors.Background)
                .padding(innerPadding)
                .padding(horizontal = YouKongTheme.spacing.xxl)
                .imePadding(),
            horizontalAlignment = Alignment.CenterHorizontally,
        ) {
            Spacer(modifier = Modifier.height(48.dp))

            // Back Button
            TerminalButton(
                text = "${ASCII.ARROW_LEFT} BACK",
                onClick = onBackClick,
                style = TerminalButtonStyle.GHOST,
                modifier = Modifier.fillMaxWidth(),
            )

            Spacer(modifier = Modifier.height(32.dp))

            // Header
            Text(
                text = "${ASCII.INFO} VERIFICATION_CODE_SENT_TO:",
                fontFamily = FontFamily.Monospace,
                fontSize = 13.sp,
                color = CLIColors.TextSecondary,
            )

            Spacer(modifier = Modifier.height(8.dp))

            Text(
                text = phone.replaceRange(3, 7, "****"),
                fontFamily = FontFamily.Monospace,
                fontSize = 18.sp,
                color = CLIColors.Green,
            )

            Spacer(modifier = Modifier.height(48.dp))

            // Verification Code Label
            Text(
                text = "ENTER_6_DIGIT_CODE:",
                fontFamily = FontFamily.Monospace,
                fontSize = 13.sp,
                color = CLIColors.TextSecondary,
                modifier = Modifier
                    .fillMaxWidth()
                    .padding(bottom = 8.dp)
            )

            // Verification Code Input
            BasicTextField(
                value = uiState.code,
                onValueChange = viewModel::onCodeChange,
                modifier = Modifier.focusRequester(focusRequester),
                keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Number),
                textStyle = TextStyle(
                    fontFamily = FontFamily.Monospace,
                    fontSize = 20.sp,
                    color = CLIColors.TextPrimary
                ),
                cursorBrush = SolidColor(CLIColors.Green),
                decorationBox = {
                    Row(
                        horizontalArrangement = Arrangement.spacedBy(8.dp),
                        modifier = Modifier.fillMaxWidth()
                    ) {
                        repeat(6) { index ->
                            CodeDigitBox(
                                digit = uiState.code.getOrNull(index)?.toString() ?: "",
                                isFocused = uiState.code.length == index,
                                modifier = Modifier.weight(1f)
                            )
                        }
                    }
                }
            )

            // Loading State
            if (uiState.isLoading) {
                Spacer(modifier = Modifier.height(16.dp))
                Text(
                    text = "${ASCII.ELLIPSIS} VERIFYING${ASCII.ELLIPSIS}",
                    fontFamily = FontFamily.Monospace,
                    fontSize = 12.sp,
                    color = CLIColors.Blue,
                )
            }

            // Error Message
            if (uiState.error != null) {
                Spacer(modifier = Modifier.height(16.dp))
                Text(
                    text = "${ASCII.CROSS} ${uiState.error}",
                    fontFamily = FontFamily.Monospace,
                    fontSize = 12.sp,
                    color = CLIColors.Red,
                )
            }

            Spacer(modifier = Modifier.height(32.dp))

            // Resend Button
            if (uiState.canResend) {
                TerminalButton(
                    text = "${ASCII.ARROW_RIGHT} RESEND_CODE",
                    onClick = viewModel::resendCode,
                    style = TerminalButtonStyle.SECONDARY,
                    modifier = Modifier.fillMaxWidth(),
                )
            } else {
                Text(
                    text = "${ASCII.BULLET} Wait ${uiState.countdown}s to resend",
                    fontFamily = FontFamily.Monospace,
                    fontSize = 12.sp,
                    color = CLIColors.TextWeak,
                )
            }
        }
    }
}

@Composable
private fun CodeDigitBox(
    digit: String,
    isFocused: Boolean,
    modifier: Modifier = Modifier,
) {
    Box(
        modifier = modifier
            .height(56.dp)
            .background(
                if (isFocused) CLIColors.Green.copy(alpha = 0.1f)
                else CLIColors.BackgroundSecondary
            )
            .border(
                width = if (isFocused) 2.dp else 1.dp,
                color = if (isFocused) CLIColors.Green else CLIColors.Border,
            )
            .padding(vertical = 12.dp),
        contentAlignment = Alignment.Center
    ) {
        if (digit.isEmpty() && isFocused) {
            // Show cursor when focused and empty
            Text(
                text = ASCII.CURSOR,
                fontFamily = FontFamily.Monospace,
                fontSize = 20.sp,
                color = CLIColors.Green,
            )
        } else {
            Text(
                text = digit,
                fontFamily = FontFamily.Monospace,
                fontSize = 20.sp,
                color = CLIColors.TextPrimary,
                textAlign = TextAlign.Center,
            )
        }
    }
}
