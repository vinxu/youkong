package com.youkong.feature.home.screen

import androidx.compose.foundation.background
import androidx.compose.foundation.layout.*
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.lazy.rememberLazyListState
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.text.font.FontFamily
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import androidx.hilt.navigation.compose.hiltViewModel
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import com.youkong.feature.home.viewmodel.LineType
import com.youkong.feature.home.viewmodel.OutputLine
import com.youkong.feature.home.viewmodel.StatusAnalysisViewModel

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun StatusAnalysisScreen(
    onComplete: () -> Unit,
    onDismiss: () -> Unit,
    viewModel: StatusAnalysisViewModel = hiltViewModel()
) {
    val uiState by viewModel.uiState.collectAsStateWithLifecycle()
    val listState = rememberLazyListState()

    // 自动滚动到底部
    LaunchedEffect(uiState.outputLines.size) {
        if (uiState.outputLines.isNotEmpty()) {
            listState.animateScrollToItem(uiState.outputLines.size - 1)
        }
    }

    // 启动分析
    LaunchedEffect(Unit) {
        viewModel.startAnalysis()
    }

    Scaffold(
        topBar = {
            TopAppBar(
                title = {
                    Text(
                        text = "状态分析",
                        color = Color.Cyan
                    )
                },
                actions = {
                    if (uiState.analysisCompleted) {
                        TextButton(onClick = {
                            onComplete()
                            onDismiss()
                        }) {
                            Text("完成", color = Color.Cyan)
                        }
                    } else if (!uiState.isAnalyzing) {
                        TextButton(onClick = onDismiss) {
                            Text("关闭", color = Color.White)
                        }
                    }
                },
                colors = TopAppBarDefaults.topAppBarColors(
                    containerColor = Color.Black
                )
            )
        },
        containerColor = Color.Black
    ) { paddingValues ->
        LazyColumn(
            state = listState,
            modifier = Modifier
                .fillMaxSize()
                .padding(paddingValues)
                .padding(16.dp),
            verticalArrangement = Arrangement.spacedBy(4.dp)
        ) {
            items(uiState.outputLines) { line ->
                OutputLineItem(line = line)
            }
        }
    }
}

@Composable
private fun OutputLineItem(line: OutputLine) {
    Text(
        text = line.text,
        fontFamily = FontFamily.Monospace,
        fontSize = when (line.type) {
            LineType.TITLE -> 20.sp
            LineType.PHASE -> 16.sp
            LineType.RESULT -> 20.sp
            LineType.CONCLUSION -> 14.sp
            else -> 13.sp
        },
        fontWeight = when (line.type) {
            LineType.TITLE -> FontWeight.Bold
            LineType.PHASE -> FontWeight.SemiBold
            LineType.RESULT -> FontWeight.Bold
            LineType.CONCLUSION -> FontWeight.Medium
            else -> FontWeight.Normal
        },
        color = when (line.type) {
            LineType.TITLE -> Color.Cyan
            LineType.PHASE -> Color.Yellow
            LineType.CLUE -> Color.Green
            LineType.THINKING -> Color.Gray
            LineType.CONCLUSION -> Color(0xFFFF9800) // Orange
            LineType.RESULT -> Color.Cyan
            LineType.ERROR -> Color.Red
            LineType.NORMAL -> Color.White
        },
        lineHeight = 20.sp,
        modifier = Modifier.fillMaxWidth()
    )
}
