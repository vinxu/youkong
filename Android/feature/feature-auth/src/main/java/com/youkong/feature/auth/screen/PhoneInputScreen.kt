package com.youkong.feature.auth.screen

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.imePadding
import androidx.compose.foundation.layout.padding
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Scaffold
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.unit.dp
import androidx.hilt.navigation.compose.hiltViewModel
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import com.youkong.core.ui.component.YouKongButton
import com.youkong.core.ui.component.YouKongPhoneTextField
import com.youkong.core.ui.theme.Primary
import com.youkong.core.ui.theme.TextSecondary
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

    Scaffold { innerPadding ->
        Column(
            modifier = Modifier
                .fillMaxSize()
                .padding(innerPadding)
                .padding(horizontal = YouKongTheme.spacing.xxl)
                .imePadding(),
            horizontalAlignment = Alignment.CenterHorizontally,
            verticalArrangement = Arrangement.Center,
        ) {
            // Logo
            Text(
                text = "有空",
                style = MaterialTheme.typography.displayMedium,
                color = Primary,
            )

            Spacer(modifier = Modifier.height(8.dp))

            Text(
                text = "把「我想见你」变成「我有空，你看着办」",
                style = MaterialTheme.typography.bodyMedium,
                color = TextSecondary,
                textAlign = TextAlign.Center,
            )

            Spacer(modifier = Modifier.height(48.dp))

            YouKongPhoneTextField(
                value = uiState.phone,
                onValueChange = viewModel::onPhoneChange,
                error = uiState.error,
                enabled = !uiState.isLoading,
                onImeAction = viewModel::sendSms,
                modifier = Modifier.fillMaxWidth(),
            )

            Spacer(modifier = Modifier.height(24.dp))

            YouKongButton(
                text = "获取验证码",
                onClick = viewModel::sendSms,
                enabled = uiState.phone.length == 11,
                loading = uiState.isLoading,
                modifier = Modifier.fillMaxWidth(),
            )

            Spacer(modifier = Modifier.height(16.dp))

            Text(
                text = "登录即表示同意《用户协议》和《隐私政策》",
                style = MaterialTheme.typography.bodySmall,
                color = TextSecondary,
                textAlign = TextAlign.Center,
            )
        }
    }
}
