package com.youkong.core.ui.component.cli

import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.text.BasicTextField
import androidx.compose.foundation.text.KeyboardActions
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.SolidColor
import androidx.compose.ui.text.TextStyle
import androidx.compose.ui.text.font.FontFamily
import androidx.compose.ui.text.input.PasswordVisualTransformation
import androidx.compose.ui.text.input.VisualTransformation
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.youkong.core.ui.theme.ASCII
import com.youkong.core.ui.theme.CLIColors

/**
 * Terminal TextField - CLI 风格输入框（ASCII 边框）
 */
@Composable
fun TerminalTextField(
    value: String,
    onValueChange: (String) -> Unit,
    modifier: Modifier = Modifier,
    placeholder: String = "",
    isPassword: Boolean = false,
    keyboardOptions: KeyboardOptions = KeyboardOptions.Default,
    onSend: (() -> Unit)? = null,
) {
    Row(
        modifier = modifier
            .fillMaxWidth()
            .background(CLIColors.BackgroundSecondary)
            .border(width = 1.dp, color = CLIColors.Border)
            .padding(horizontal = 12.dp, vertical = 10.dp),
        verticalAlignment = Alignment.CenterVertically
    ) {
        // 提示符
        Text(
            text = ASCII.PROMPT,
            fontFamily = FontFamily.Monospace,
            fontSize = 14.sp,
            color = CLIColors.Green
        )

        Spacer(modifier = Modifier.width(8.dp))

        // 输入框
        BasicTextField(
            value = value,
            onValueChange = onValueChange,
            modifier = Modifier.weight(1f),
            textStyle = TextStyle(
                fontFamily = FontFamily.Monospace,
                fontSize = 14.sp,
                color = CLIColors.TextPrimary
            ),
            cursorBrush = SolidColor(CLIColors.Green),
            visualTransformation = if (isPassword) PasswordVisualTransformation() else VisualTransformation.None,
            keyboardOptions = keyboardOptions,
            keyboardActions = KeyboardActions(
                onSend = { onSend?.invoke() },
                onDone = { onSend?.invoke() }
            ),
            decorationBox = { innerTextField ->
                if (value.isEmpty()) {
                    Text(
                        text = placeholder,
                        fontFamily = FontFamily.Monospace,
                        fontSize = 14.sp,
                        color = CLIColors.TextWeak
                    )
                }
                innerTextField()
            }
        )
    }
}

/**
 * Labeled Terminal TextField - 带标签的输入框
 */
@Composable
fun LabeledTerminalTextField(
    label: String,
    value: String,
    onValueChange: (String) -> Unit,
    modifier: Modifier = Modifier,
    placeholder: String = "",
    isPassword: Boolean = false,
    keyboardOptions: KeyboardOptions = KeyboardOptions.Default,
) {
    Column(modifier = modifier) {
        // 标签
        Text(
            text = label,
            fontFamily = FontFamily.Monospace,
            fontSize = 13.sp,
            color = CLIColors.TextSecondary,
            modifier = Modifier.padding(bottom = 8.dp)
        )

        // 输入框
        TerminalTextField(
            value = value,
            onValueChange = onValueChange,
            placeholder = placeholder,
            isPassword = isPassword,
            keyboardOptions = keyboardOptions
        )
    }
}
