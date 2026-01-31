package com.youkong.feature.settings.screen

import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.lazy.rememberLazyListState
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material.icons.automirrored.filled.Send
import androidx.compose.material.icons.filled.PlayArrow
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.Scaffold
import androidx.compose.material3.Text
import androidx.compose.material3.TopAppBar
import androidx.compose.material3.TopAppBarDefaults
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.collectAsState
import androidx.compose.runtime.getValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.text.font.FontFamily
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import androidx.hilt.navigation.compose.hiltViewModel
import com.youkong.core.network.model.HolmesFullResult
import com.youkong.feature.settings.viewmodel.AgentDataViewModel
import com.youkong.feature.settings.viewmodel.CliLine

// Terminal 颜色方案 - 深色主题
private val TerminalBg = Color(0xFF0D1117)
private val TerminalBgAlt = Color(0xFF161B22)
private val TerminalGreen = Color(0xFF3FB950)
private val TerminalYellow = Color(0xFFD29922)
private val TerminalBlue = Color(0xFF58A6FF)
private val TerminalPurple = Color(0xFFBC8CFF)
private val TerminalCyan = Color(0xFF39C5CF)
private val TerminalOrange = Color(0xFFF0883E)
private val TerminalRed = Color(0xFFF85149)
private val TerminalWhite = Color(0xFFE6EDF3)
private val TerminalGray = Color(0xFF8B949E)
private val TerminalDimGray = Color(0xFF484F58)

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun AgentDataScreen(
    onBackClick: () -> Unit,
    viewModel: AgentDataViewModel = hiltViewModel(),
) {
    val uiState by viewModel.uiState.collectAsState()
    val listState = rememberLazyListState()

    // 自动滚动到底部
    LaunchedEffect(uiState.cliLines.size) {
        if (uiState.cliLines.isNotEmpty()) {
            listState.animateScrollToItem(uiState.cliLines.size - 1)
        }
    }

    Scaffold(
        topBar = {
            TopAppBar(
                title = {
                    Row(verticalAlignment = Alignment.CenterVertically) {
                        Text(
                            text = "🔍",
                            fontSize = 18.sp,
                        )
                        Spacer(modifier = Modifier.width(8.dp))
                        Text(
                            text = "Holmes Terminal",
                            fontFamily = FontFamily.Monospace,
                            fontSize = 16.sp,
                            fontWeight = FontWeight.Bold,
                            color = TerminalGreen,
                        )
                    }
                },
                navigationIcon = {
                    IconButton(onClick = onBackClick) {
                        Icon(
                            imageVector = Icons.AutoMirrored.Filled.ArrowBack,
                            contentDescription = "返回",
                            tint = TerminalWhite,
                        )
                    }
                },
                actions = {
                    // 上报按钮
                    IconButton(
                        onClick = { viewModel.uploadStatus() },
                        enabled = !uiState.isUploading && uiState.lastResult != null && !uiState.isRunning,
                    ) {
                        Icon(
                            imageVector = Icons.AutoMirrored.Filled.Send,
                            contentDescription = "上报状态",
                            tint = if (uiState.lastResult != null && !uiState.isUploading && !uiState.isRunning) {
                                TerminalYellow
                            } else {
                                TerminalDimGray
                            },
                            modifier = Modifier.size(24.dp),
                        )
                    }

                    // 运行分析按钮
                    IconButton(
                        onClick = { viewModel.refresh() },
                        enabled = !uiState.isRunning,
                    ) {
                        Icon(
                            imageVector = Icons.Default.PlayArrow,
                            contentDescription = "运行分析",
                            tint = if (uiState.isRunning) TerminalDimGray else TerminalGreen,
                            modifier = Modifier.size(28.dp),
                        )
                    }
                },
                colors = TopAppBarDefaults.topAppBarColors(
                    containerColor = TerminalBg,
                ),
            )
        },
        containerColor = TerminalBg,
    ) { paddingValues ->
        LazyColumn(
            state = listState,
            modifier = Modifier
                .fillMaxSize()
                .padding(paddingValues)
                .padding(horizontal = 12.dp, vertical = 8.dp),
            verticalArrangement = Arrangement.spacedBy(2.dp),
        ) {
            items(uiState.cliLines) { line ->
                CliLineItem(line)
            }

            item {
                Spacer(modifier = Modifier.height(120.dp))
            }
        }
    }
}

@Composable
private fun CliLineItem(line: CliLine) {
    when (line) {
        is CliLine.Command -> CommandLine(line.text)
        is CliLine.Output -> OutputLine(line.text)
        is CliLine.Success -> SuccessLine(line.text)
        is CliLine.Error -> ErrorLine(line.text)
        is CliLine.Phase -> PhaseLine(line.emoji, line.text)
        is CliLine.Clue -> ClueLine(line.text, line.isLast)
        is CliLine.Feature -> FeatureLine(line.text, line.isLast)
        is CliLine.Thinking -> ThinkingLine(line.text)
        is CliLine.Conclusion -> ConclusionLine(line.text, line.isLast)
        is CliLine.Result -> ResultBlock(line.result)
        is CliLine.Divider -> DividerLine()
    }
}

@Composable
private fun CommandLine(text: String) {
    Row(
        modifier = Modifier
            .fillMaxWidth()
            .padding(vertical = 4.dp),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        Text(
            text = "❯ ",
            fontFamily = FontFamily.Monospace,
            fontSize = 14.sp,
            color = TerminalGreen,
            fontWeight = FontWeight.Bold,
        )
        Text(
            text = text,
            fontFamily = FontFamily.Monospace,
            fontSize = 14.sp,
            color = TerminalYellow,
            fontWeight = FontWeight.Medium,
        )
    }
}

@Composable
private fun OutputLine(text: String) {
    Text(
        text = text,
        fontFamily = FontFamily.Monospace,
        fontSize = 13.sp,
        color = TerminalWhite,
        modifier = Modifier.padding(vertical = 1.dp),
    )
}

@Composable
private fun SuccessLine(text: String) {
    Text(
        text = text,
        fontFamily = FontFamily.Monospace,
        fontSize = 13.sp,
        color = TerminalGreen,
        modifier = Modifier.padding(vertical = 1.dp),
    )
}

@Composable
private fun ErrorLine(text: String) {
    Row(
        modifier = Modifier
            .fillMaxWidth()
            .clip(RoundedCornerShape(4.dp))
            .background(TerminalRed.copy(alpha = 0.15f))
            .padding(horizontal = 8.dp, vertical = 4.dp),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        Text(
            text = "✗ ",
            fontFamily = FontFamily.Monospace,
            fontSize = 13.sp,
            color = TerminalRed,
        )
        Text(
            text = text,
            fontFamily = FontFamily.Monospace,
            fontSize = 13.sp,
            color = TerminalRed,
        )
    }
}

@Composable
private fun PhaseLine(emoji: String, text: String) {
    Row(
        modifier = Modifier
            .fillMaxWidth()
            .padding(top = 8.dp, bottom = 4.dp),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        Text(
            text = emoji,
            fontSize = 14.sp,
        )
        Spacer(modifier = Modifier.width(8.dp))
        Text(
            text = text,
            fontFamily = FontFamily.Monospace,
            fontSize = 14.sp,
            color = TerminalCyan,
            fontWeight = FontWeight.Bold,
        )
    }
}

@Composable
private fun ClueLine(text: String, isLast: Boolean) {
    Row(
        modifier = Modifier
            .fillMaxWidth()
            .padding(start = 20.dp, top = 2.dp, bottom = 2.dp),
    ) {
        Text(
            text = if (isLast) "`-- " else "|-- ",
            fontFamily = FontFamily.Monospace,
            fontSize = 13.sp,
            color = TerminalDimGray,
        )
        Text(
            text = text,
            fontFamily = FontFamily.Monospace,
            fontSize = 13.sp,
            color = TerminalWhite,
        )
    }
}

@Composable
private fun FeatureLine(text: String, isLast: Boolean) {
    Row(
        modifier = Modifier
            .fillMaxWidth()
            .padding(start = 20.dp, top = 2.dp, bottom = 2.dp),
    ) {
        Text(
            text = if (isLast) "`-- " else "|-- ",
            fontFamily = FontFamily.Monospace,
            fontSize = 13.sp,
            color = TerminalDimGray,
        )
        Text(
            text = text,
            fontFamily = FontFamily.Monospace,
            fontSize = 13.sp,
            color = TerminalBlue,
        )
    }
}

@Composable
private fun ThinkingLine(text: String) {
    Row(
        modifier = Modifier
            .fillMaxWidth()
            .padding(start = 20.dp, top = 2.dp, bottom = 2.dp),
    ) {
        Text(
            text = "|  ",
            fontFamily = FontFamily.Monospace,
            fontSize = 13.sp,
            color = TerminalDimGray,
        )
        Text(
            text = text,
            fontFamily = FontFamily.Monospace,
            fontSize = 13.sp,
            color = TerminalPurple,
            fontWeight = FontWeight.Light,
        )
    }
}

@Composable
private fun ConclusionLine(text: String, isLast: Boolean) {
    Row(
        modifier = Modifier
            .fillMaxWidth()
            .padding(start = 20.dp, top = 2.dp, bottom = 2.dp),
    ) {
        Text(
            text = if (isLast) "`-- " else "|-- ",
            fontFamily = FontFamily.Monospace,
            fontSize = 13.sp,
            color = TerminalDimGray,
        )
        Text(
            text = text,
            fontFamily = FontFamily.Monospace,
            fontSize = 13.sp,
            color = TerminalOrange,
            fontWeight = FontWeight.Medium,
        )
    }
}

@Composable
private fun ResultBlock(fullResult: HolmesFullResult) {
    val result = fullResult.result
    val probability = result?.probability ?: 0
    val confidence = result?.confidence ?: "low"
    val summary = result?.summary ?: ""
    val available = result?.available ?: false

    val probabilityColor = when {
        probability >= 80 -> Color(0xFF3FB950) // 绿色
        probability >= 60 -> Color(0xFF56D364) // 浅绿
        probability >= 40 -> Color(0xFFD29922) // 黄色
        probability >= 20 -> Color(0xFFF0883E) // 橙色
        else -> Color(0xFFF85149) // 红色
    }

    // 根据状态选择 emoji
    val emoji = when {
        probability >= 70 -> "✅"
        probability >= 40 -> "🤔"
        else -> "🚫"
    }

    val status = if (available) "Available" else "Busy"

    Column(
        modifier = Modifier
            .fillMaxWidth()
            .padding(vertical = 8.dp)
            .clip(RoundedCornerShape(8.dp))
            .background(TerminalBgAlt)
            .padding(16.dp),
    ) {
        // 头部：emoji + 状态
        Row(
            verticalAlignment = Alignment.CenterVertically,
        ) {
            Text(
                text = emoji,
                fontSize = 40.sp,
            )
            Spacer(modifier = Modifier.width(16.dp))
            Column {
                Text(
                    text = status,
                    fontFamily = FontFamily.Monospace,
                    fontSize = 20.sp,
                    fontWeight = FontWeight.Bold,
                    color = if (available) TerminalGreen else TerminalOrange,
                )
                Spacer(modifier = Modifier.height(4.dp))
                Row(verticalAlignment = Alignment.CenterVertically) {
                    Text(
                        text = "probability:",
                        fontFamily = FontFamily.Monospace,
                        fontSize = 12.sp,
                        color = TerminalGray,
                    )
                    Spacer(modifier = Modifier.width(8.dp))
                    Box(
                        modifier = Modifier
                            .clip(RoundedCornerShape(4.dp))
                            .background(probabilityColor)
                            .padding(horizontal = 10.dp, vertical = 3.dp),
                    ) {
                        Text(
                            text = "$probability%",
                            fontFamily = FontFamily.Monospace,
                            fontSize = 14.sp,
                            fontWeight = FontWeight.Bold,
                            color = Color.White,
                        )
                    }
                    Spacer(modifier = Modifier.width(12.dp))
                    Text(
                        text = "confidence: $confidence",
                        fontFamily = FontFamily.Monospace,
                        fontSize = 12.sp,
                        color = TerminalGray,
                    )
                }
            }
        }

        Spacer(modifier = Modifier.height(12.dp))

        // 摘要
        if (summary.isNotEmpty()) {
            Text(
                text = "// $summary",
                fontFamily = FontFamily.Monospace,
                fontSize = 13.sp,
                color = TerminalGray,
            )
        }

        // 特征信息
        fullResult.features?.let { features ->
            Spacer(modifier = Modifier.height(8.dp))
            Column {
                features.timePeriod?.let {
                    FeatureItem("time", it)
                }
                features.locationType?.let {
                    FeatureItem("location", it)
                }
                features.activity?.let {
                    FeatureItem("activity", it)
                }
            }
        }

        // 推理结论
        fullResult.reasoning?.conclusion?.let { conclusion ->
            if (conclusion.isNotEmpty()) {
                Spacer(modifier = Modifier.height(8.dp))
                Text(
                    text = "/* $conclusion */",
                    fontFamily = FontFamily.Monospace,
                    fontSize = 12.sp,
                    color = TerminalDimGray,
                    lineHeight = 18.sp,
                )
            }
        }
    }
}

@Composable
private fun FeatureItem(key: String, value: String) {
    Row(
        modifier = Modifier.padding(vertical = 1.dp),
    ) {
        Text(
            text = "$key: ",
            fontFamily = FontFamily.Monospace,
            fontSize = 12.sp,
            color = TerminalBlue,
        )
        Text(
            text = value,
            fontFamily = FontFamily.Monospace,
            fontSize = 12.sp,
            color = TerminalWhite,
        )
    }
}

@Composable
private fun DividerLine() {
    Text(
        text = "-".repeat(44),
        fontFamily = FontFamily.Monospace,
        fontSize = 12.sp,
        color = TerminalDimGray.copy(alpha = 0.5f),
        modifier = Modifier.padding(vertical = 4.dp),
    )
}
