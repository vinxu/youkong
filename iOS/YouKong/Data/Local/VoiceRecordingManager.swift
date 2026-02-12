import Foundation
import AVFoundation
import Combine

/// 语音录制管理器
@MainActor
class VoiceRecordingManager: NSObject, ObservableObject {
    // MARK: - Published Properties

    @Published var isRecording = false
    @Published var recordingDuration: TimeInterval = 0
    @Published var isCancelled = false
    @Published var hasPermission = false
    @Published var permissionDenied = false

    // MARK: - Private Properties

    private var audioRecorder: AVAudioRecorder?
    private var recordingURL: URL?
    private var durationTimer: Timer?
    private var startTime: Date?

    // MARK: - Constants

    private let minimumDuration: TimeInterval = 1.0  // 最短录音时长（秒）

    // MARK: - Init

    override init() {
        super.init()
        checkPermission()
    }

    // MARK: - Permission

    func checkPermission() {
        switch AVAudioSession.sharedInstance().recordPermission {
        case .granted:
            hasPermission = true
            permissionDenied = false
        case .denied:
            hasPermission = false
            permissionDenied = true
        case .undetermined:
            hasPermission = false
            permissionDenied = false
        @unknown default:
            hasPermission = false
        }
    }

    func requestPermission() async -> Bool {
        return await withCheckedContinuation { continuation in
            AVAudioSession.sharedInstance().requestRecordPermission { granted in
                Task { @MainActor in
                    self.hasPermission = granted
                    self.permissionDenied = !granted
                    continuation.resume(returning: granted)
                }
            }
        }
    }

    // MARK: - Audio Session

    /// 预热音频会话，减少录音启动延迟
    func prepareAudioSession() {
        do {
            let audioSession = AVAudioSession.sharedInstance()
            try audioSession.setCategory(.playAndRecord, mode: .default, options: [.defaultToSpeaker])
            try audioSession.setActive(true)
        } catch {
            print("🎤 [Voice] 预热音频会话失败: \(error)")
        }
    }

    // MARK: - Recording

    /// 开始录音
    /// - Returns: 录音文件 URL，如果失败返回 nil
    func startRecording() throws -> URL {
        guard hasPermission else {
            throw VoiceRecordingError.permissionDenied
        }

        // 配置音频会话
        let audioSession = AVAudioSession.sharedInstance()
        try audioSession.setCategory(.playAndRecord, mode: .default, options: [.defaultToSpeaker])
        try audioSession.setActive(true)

        // 创建临时文件
        let tempDir = FileManager.default.temporaryDirectory
        let fileName = "voice_\(UUID().uuidString).wav"
        let fileURL = tempDir.appendingPathComponent(fileName)

        // 录音设置（使用 Linear PCM / WAV 格式，阿里云 ASR 支持）
        let settings: [String: Any] = [
            AVFormatIDKey: Int(kAudioFormatLinearPCM),
            AVSampleRateKey: 16000,  // 16kHz，语音识别最佳
            AVNumberOfChannelsKey: 1,  // 单声道
            AVLinearPCMBitDepthKey: 16,  // 16位
            AVLinearPCMIsFloatKey: false,
            AVLinearPCMIsBigEndianKey: false
        ]

        audioRecorder = try AVAudioRecorder(url: fileURL, settings: settings)
        audioRecorder?.delegate = self

        guard audioRecorder?.record() == true else {
            throw VoiceRecordingError.recordingFailed
        }

        recordingURL = fileURL
        isRecording = true
        isCancelled = false
        startTime = Date()
        recordingDuration = 0

        // 启动计时器
        startDurationTimer()

        print("🎤 [Voice] 开始录音: \(fileURL.lastPathComponent)")

        return fileURL
    }

    /// 停止录音
    /// - Returns: 录音文件 URL 和时长，如果录音时间太短返回 nil
    func stopRecording() -> (url: URL, duration: TimeInterval)? {
        stopDurationTimer()
        audioRecorder?.stop()
        isRecording = false

        // 停用音频会话
        try? AVAudioSession.sharedInstance().setActive(false)

        guard !isCancelled else {
            print("🎤 [Voice] 录音已取消")
            cleanupRecording()
            return nil
        }

        guard let url = recordingURL else {
            print("🎤 [Voice] 没有录音文件")
            return nil
        }

        let duration = recordingDuration

        // 检查最短时长
        if duration < minimumDuration {
            print("🎤 [Voice] 录音时间太短: \(duration)s < \(minimumDuration)s")
            cleanupRecording()
            return nil
        }

        print("🎤 [Voice] 录音完成: \(duration)s")

        return (url, duration)
    }

    /// 取消录音
    func cancelRecording() {
        isCancelled = true
        stopDurationTimer()
        audioRecorder?.stop()
        isRecording = false

        // 停用音频会话
        try? AVAudioSession.sharedInstance().setActive(false)

        cleanupRecording()
        print("🎤 [Voice] 录音已取消")
    }

    /// 获取录音数据
    func getAudioData() -> Data? {
        guard let url = recordingURL else { return nil }
        return try? Data(contentsOf: url)
    }

    // MARK: - Private Methods

    private func startDurationTimer() {
        durationTimer = Timer.scheduledTimer(withTimeInterval: 0.1, repeats: true) { [weak self] _ in
            Task { @MainActor in
                guard let self = self, let startTime = self.startTime else { return }
                self.recordingDuration = Date().timeIntervalSince(startTime)
            }
        }
    }

    private func stopDurationTimer() {
        durationTimer?.invalidate()
        durationTimer = nil
    }

    private func cleanupRecording() {
        if let url = recordingURL {
            try? FileManager.default.removeItem(at: url)
        }
        recordingURL = nil
        recordingDuration = 0
        startTime = nil
        isCancelled = false
    }
}

// MARK: - AVAudioRecorderDelegate

extension VoiceRecordingManager: AVAudioRecorderDelegate {
    nonisolated func audioRecorderDidFinishRecording(_ recorder: AVAudioRecorder, successfully flag: Bool) {
        Task { @MainActor in
            if !flag {
                print("🎤 [Voice] 录音失败")
                cleanupRecording()
            }
        }
    }

    nonisolated func audioRecorderEncodeErrorDidOccur(_ recorder: AVAudioRecorder, error: Error?) {
        Task { @MainActor in
            print("🎤 [Voice] 录音编码错误: \(error?.localizedDescription ?? "unknown")")
            cleanupRecording()
        }
    }
}

// MARK: - Errors

enum VoiceRecordingError: LocalizedError {
    case permissionDenied
    case recordingFailed
    case tooShort

    var errorDescription: String? {
        switch self {
        case .permissionDenied:
            return "没有麦克风权限"
        case .recordingFailed:
            return "录音失败"
        case .tooShort:
            return "录音时间太短，请长按说话"
        }
    }
}
