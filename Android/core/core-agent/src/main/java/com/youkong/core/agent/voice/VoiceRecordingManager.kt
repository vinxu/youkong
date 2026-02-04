package com.youkong.core.agent.voice

import android.Manifest
import android.content.Context
import android.content.pm.PackageManager
import android.media.AudioFormat
import android.media.AudioRecord
import android.media.MediaRecorder
import android.util.Log
import androidx.core.content.ContextCompat
import dagger.hilt.android.qualifiers.ApplicationContext
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.withContext
import java.io.ByteArrayOutputStream
import java.io.File
import java.io.FileOutputStream
import java.io.RandomAccessFile
import java.util.Timer
import java.util.TimerTask
import javax.inject.Inject
import javax.inject.Singleton

/**
 * 语音录制管理器
 *
 * 支持 WAV 16kHz 单声道格式录制，兼容后端 ASR
 */
@Singleton
class VoiceRecordingManager @Inject constructor(
    @ApplicationContext private val context: Context
) {
    companion object {
        private const val TAG = "VoiceRecording"
        private const val SAMPLE_RATE = 16000
        private const val CHANNEL_CONFIG = AudioFormat.CHANNEL_IN_MONO
        private const val AUDIO_FORMAT = AudioFormat.ENCODING_PCM_16BIT
        private const val MINIMUM_DURATION_MS = 1000L  // 最短录音时长（毫秒）
    }

    // 状态流
    private val _isRecording = MutableStateFlow(false)
    val isRecording: StateFlow<Boolean> = _isRecording.asStateFlow()

    private val _recordingDuration = MutableStateFlow(0L)  // 毫秒
    val recordingDuration: StateFlow<Long> = _recordingDuration.asStateFlow()

    private val _isCancelled = MutableStateFlow(false)
    val isCancelled: StateFlow<Boolean> = _isCancelled.asStateFlow()

    // 录制相关
    private var audioRecord: AudioRecord? = null
    private var recordingThread: Thread? = null
    private var audioDataOutputStream: ByteArrayOutputStream? = null
    private var startTime: Long = 0
    private var durationTimer: Timer? = null

    /**
     * 检查是否有录音权限
     */
    val hasPermission: Boolean
        get() = ContextCompat.checkSelfPermission(
            context,
            Manifest.permission.RECORD_AUDIO
        ) == PackageManager.PERMISSION_GRANTED

    /**
     * 开始录音
     * @return 是否成功开始
     */
    fun startRecording(): Boolean {
        if (!hasPermission) {
            Log.e(TAG, "没有录音权限")
            return false
        }

        if (_isRecording.value) {
            Log.w(TAG, "已经在录音中")
            return false
        }

        try {
            val bufferSize = AudioRecord.getMinBufferSize(SAMPLE_RATE, CHANNEL_CONFIG, AUDIO_FORMAT)
            if (bufferSize == AudioRecord.ERROR || bufferSize == AudioRecord.ERROR_BAD_VALUE) {
                Log.e(TAG, "获取缓冲区大小失败")
                return false
            }

            audioRecord = AudioRecord(
                MediaRecorder.AudioSource.MIC,
                SAMPLE_RATE,
                CHANNEL_CONFIG,
                AUDIO_FORMAT,
                bufferSize
            )

            if (audioRecord?.state != AudioRecord.STATE_INITIALIZED) {
                Log.e(TAG, "AudioRecord 初始化失败")
                audioRecord?.release()
                audioRecord = null
                return false
            }

            audioDataOutputStream = ByteArrayOutputStream()
            _isRecording.value = true
            _isCancelled.value = false
            startTime = System.currentTimeMillis()
            _recordingDuration.value = 0

            // 启动录音线程
            audioRecord?.startRecording()
            recordingThread = Thread {
                val buffer = ByteArray(bufferSize)
                while (_isRecording.value && !_isCancelled.value) {
                    val bytesRead = audioRecord?.read(buffer, 0, buffer.size) ?: -1
                    if (bytesRead > 0) {
                        audioDataOutputStream?.write(buffer, 0, bytesRead)
                    }
                }
            }.apply { start() }

            // 启动计时器
            startDurationTimer()

            Log.d(TAG, "🎤 开始录音")
            return true
        } catch (e: SecurityException) {
            Log.e(TAG, "录音权限被拒绝: ${e.message}")
            return false
        } catch (e: Exception) {
            Log.e(TAG, "开始录音失败: ${e.message}")
            cleanup()
            return false
        }
    }

    /**
     * 停止录音并获取音频数据
     * @return 音频文件和时长，如果录音太短或取消则返回 null
     */
    suspend fun stopRecording(): RecordingResult? = withContext(Dispatchers.IO) {
        if (!_isRecording.value) {
            return@withContext null
        }

        stopDurationTimer()
        _isRecording.value = false

        // 等待录音线程结束
        recordingThread?.join(1000)

        audioRecord?.stop()
        audioRecord?.release()
        audioRecord = null

        if (_isCancelled.value) {
            Log.d(TAG, "🎤 录音已取消")
            cleanup()
            return@withContext null
        }

        val duration = _recordingDuration.value
        if (duration < MINIMUM_DURATION_MS) {
            Log.d(TAG, "🎤 录音时间太短: ${duration}ms < ${MINIMUM_DURATION_MS}ms")
            cleanup()
            return@withContext null
        }

        val pcmData = audioDataOutputStream?.toByteArray()
        if (pcmData == null || pcmData.isEmpty()) {
            Log.e(TAG, "没有录音数据")
            cleanup()
            return@withContext null
        }

        // 转换为 WAV 格式
        val wavData = pcmToWav(pcmData)

        // 保存到临时文件
        val tempFile = File(context.cacheDir, "voice_${System.currentTimeMillis()}.wav")
        FileOutputStream(tempFile).use { it.write(wavData) }

        cleanup()

        Log.d(TAG, "🎤 录音完成: ${duration}ms, ${wavData.size} bytes")

        RecordingResult(
            file = tempFile,
            data = wavData,
            durationMs = duration
        )
    }

    /**
     * 取消录音
     */
    fun cancelRecording() {
        if (_isRecording.value) {
            _isCancelled.value = true
            _isRecording.value = false

            stopDurationTimer()
            recordingThread?.interrupt()

            audioRecord?.stop()
            audioRecord?.release()
            audioRecord = null

            cleanup()

            Log.d(TAG, "🎤 录音已取消")
        }
    }

    /**
     * 设置取消状态（用于 UI 反馈上滑取消区域）
     */
    fun setCancelling(cancelling: Boolean) {
        _isCancelled.value = cancelling
    }

    private fun startDurationTimer() {
        durationTimer = Timer().apply {
            scheduleAtFixedRate(object : TimerTask() {
                override fun run() {
                    if (_isRecording.value) {
                        _recordingDuration.value = System.currentTimeMillis() - startTime
                    }
                }
            }, 0, 100)  // 每 100ms 更新一次
        }
    }

    private fun stopDurationTimer() {
        durationTimer?.cancel()
        durationTimer = null
    }

    private fun cleanup() {
        audioDataOutputStream?.close()
        audioDataOutputStream = null
        recordingThread = null
    }

    /**
     * 将 PCM 数据转换为 WAV 格式
     */
    private fun pcmToWav(pcmData: ByteArray): ByteArray {
        val totalAudioLen = pcmData.size
        val totalDataLen = totalAudioLen + 36
        val byteRate = SAMPLE_RATE * 1 * 16 / 8  // 单声道, 16位

        val header = ByteArray(44)

        // RIFF header
        header[0] = 'R'.code.toByte()
        header[1] = 'I'.code.toByte()
        header[2] = 'F'.code.toByte()
        header[3] = 'F'.code.toByte()
        header[4] = (totalDataLen and 0xff).toByte()
        header[5] = ((totalDataLen shr 8) and 0xff).toByte()
        header[6] = ((totalDataLen shr 16) and 0xff).toByte()
        header[7] = ((totalDataLen shr 24) and 0xff).toByte()

        // WAVE header
        header[8] = 'W'.code.toByte()
        header[9] = 'A'.code.toByte()
        header[10] = 'V'.code.toByte()
        header[11] = 'E'.code.toByte()

        // fmt subchunk
        header[12] = 'f'.code.toByte()
        header[13] = 'm'.code.toByte()
        header[14] = 't'.code.toByte()
        header[15] = ' '.code.toByte()
        header[16] = 16  // Subchunk1Size (16 for PCM)
        header[17] = 0
        header[18] = 0
        header[19] = 0
        header[20] = 1  // AudioFormat (1 for PCM)
        header[21] = 0
        header[22] = 1  // NumChannels (1 for mono)
        header[23] = 0
        header[24] = (SAMPLE_RATE and 0xff).toByte()
        header[25] = ((SAMPLE_RATE shr 8) and 0xff).toByte()
        header[26] = ((SAMPLE_RATE shr 16) and 0xff).toByte()
        header[27] = ((SAMPLE_RATE shr 24) and 0xff).toByte()
        header[28] = (byteRate and 0xff).toByte()
        header[29] = ((byteRate shr 8) and 0xff).toByte()
        header[30] = ((byteRate shr 16) and 0xff).toByte()
        header[31] = ((byteRate shr 24) and 0xff).toByte()
        header[32] = 2  // BlockAlign (NumChannels * BitsPerSample / 8)
        header[33] = 0
        header[34] = 16  // BitsPerSample
        header[35] = 0

        // data subchunk
        header[36] = 'd'.code.toByte()
        header[37] = 'a'.code.toByte()
        header[38] = 't'.code.toByte()
        header[39] = 'a'.code.toByte()
        header[40] = (totalAudioLen and 0xff).toByte()
        header[41] = ((totalAudioLen shr 8) and 0xff).toByte()
        header[42] = ((totalAudioLen shr 16) and 0xff).toByte()
        header[43] = ((totalAudioLen shr 24) and 0xff).toByte()

        return header + pcmData
    }
}

/**
 * 录音结果
 */
data class RecordingResult(
    val file: File,
    val data: ByteArray,
    val durationMs: Long
) {
    val durationSeconds: Float get() = durationMs / 1000f

    override fun equals(other: Any?): Boolean {
        if (this === other) return true
        if (javaClass != other?.javaClass) return false
        other as RecordingResult
        return file == other.file && durationMs == other.durationMs
    }

    override fun hashCode(): Int {
        var result = file.hashCode()
        result = 31 * result + durationMs.hashCode()
        return result
    }
}
