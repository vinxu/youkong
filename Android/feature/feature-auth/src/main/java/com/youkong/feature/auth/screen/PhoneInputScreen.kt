package com.youkong.feature.auth.screen

import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.imePadding
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.material3.Scaffold
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.text.font.FontFamily
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.input.KeyboardType
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import androidx.hilt.navigation.compose.hiltViewModel
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import com.youkong.core.ui.component.cli.LabeledTerminalTextField
import com.youkong.core.ui.component.cli.TerminalButton
import com.youkong.core.ui.component.cli.TerminalButtonStyle
import com.youkong.core.ui.theme.ASCII
import com.youkong.core.ui.theme.CLIColors
import com.youkong.core.ui.theme.YouKongTheme
import com.youkong.feature.auth.viewmodel.PhoneInputViewModel

@Composable
fun PhoneInputScreen(
    onNavigateToVerification: (String) -> Unit,
    viewModel: PhoneInputViewModel = hiltViewModel(),
) {
    val uiState by viewModel.uiState.collectAsStateWithLifecycle()

    LaunchedEffect(uiState.isSmsSent) {
        if (uiState.isSmsSent) {
            onNavigateToVerification(uiState.phone)
            viewModel.resetSmsSent()
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
            verticalArrangement = Arrangement.Center,
        ) {
            // ASCII Logo
            Text(
                text = """
                    ╦ ╦╔═╗╦ ╦╦╔═╔═╗╔╗╔╔═╗
                    ╚╦╝║ ║║ ║╠╩╗║ ║║║║║ ╦
                     ╩ ╚═╝╚═╝╩ ╩╚═╝╝╚╝╚═╝
                """.trimIndent(),
                fontFamily = FontFamily.Monospace,
                fontSize = 14.sp,
                fontWeight = FontWeight.Bold,
                color = CLIColors.Green,
                lineHeight = 16.sp,
                textAlign = TextAlign.Center,
            )

            Spacer(modifier = Modifier.height(16.dp))

            Text(
                text = "${ASCII.ARROW_RIGHT} 把「我想见你」变成「我有空，你看着办」",
                fontFamily = FontFamily.Monospace,
                fontSize = 13.sp,
                color = CLIColors.TextSecondary,
                textAlign = TextAlign.Center,
            )

            Spacer(modifier = Modifier.height(48.dp))

            // Phone Input
            LabeledTerminalTextField(
                label = "PHONE_NUMBER:",
                value = uiState.phone,
                onValueChange = viewModel::onPhoneChange,
                placeholder = "请输入手机号",
                keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Phone),
                modifier = Modifier.fillMaxWidth(),
            )

            // Error Message
            if (uiState.error != null) {
                Spacer(modifier = Modifier.height(12.dp))
                Text(
                    text = "${ASCII.CROSS} ${uiState.error}",
                    fontFamily = FontFamily.Monospace,
                    fontSize = 12.sp,
                    color = CLIColors.Red,
                )
            }

            Spacer(modifier = Modifier.height(24.dp))

            // Send SMS Button
            TerminalButton(
                text = if (uiState.isLoading) "[SENDING...]" else "[GET_VERIFICATION_CODE]",
                onClick = viewModel::sendSms,
                enabled = uiState.phone.length == 11 && !uiState.isLoading,
                style = TerminalButtonStyle.PRIMARY,
                modifier = Modifier.fillMaxWidth(),
            )

            Spacer(modifier = Modifier.height(24.dp))

            Text(
                text = "${ASCII.INFO} 登录即表示同意《用户协议》和《隐私政策》",
                fontFamily = FontFamily.Monospace,
                fontSize = 11.sp,
                color = CLIColors.TextWeak,
                textAlign = TextAlign.Center,
            )
        }
    }
}
