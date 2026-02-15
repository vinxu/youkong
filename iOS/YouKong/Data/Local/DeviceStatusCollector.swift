import Foundation
import UIKit
import AVFoundation
import Network
import NetworkExtension

// MARK: - Device Status

struct DeviceStatus {
    let batteryLevel: Float           // 0-1
    let batteryState: BatteryState
    let isLowPowerMode: Bool
    let isFocusModeOn: Bool           // 勿扰/专注模式
    let isHeadphonesConnected: Bool
    let networkType: NetworkType
    let screenBrightness: Float       // 0-1
    let bluetoothDeviceType: String?  // headphones/car/speaker
    var wifiSSID: String?             // WiFi 名称

    var isCharging: Bool {
        batteryState == .charging || batteryState == .full
    }

    static let unknown = DeviceStatus(
        batteryLevel: 0,
        batteryState: .unknown,
        isLowPowerMode: false,
        isFocusModeOn: false,
        isHeadphonesConnected: false,
        networkType: .unknown,
        screenBrightness: 0.5,
        bluetoothDeviceType: nil,
        wifiSSID: nil
    )
}

enum BatteryState: String {
    case unknown
    case unplugged   // 未充电
    case charging    // 充电中
    case full        // 已充满
}

enum NetworkType: String {
    case wifi
    case cellular
    case none
    case unknown
}

// MARK: - Device Status Collector

class DeviceStatusCollector: ObservableObject {
    static let shared = DeviceStatusCollector()

    @Published private(set) var currentStatus: DeviceStatus = .unknown

    private let networkMonitor = NWPathMonitor()
    private let networkQueue = DispatchQueue(label: "NetworkMonitor")
    private var currentNetworkType: NetworkType = .unknown

    private init() {
        setupBatteryMonitoring()
        setupNetworkMonitoring()
        updateStatus()
    }

    deinit {
        UIDevice.current.isBatteryMonitoringEnabled = false
        networkMonitor.cancel()
    }

    // MARK: - Setup

    private func setupBatteryMonitoring() {
        UIDevice.current.isBatteryMonitoringEnabled = true

        NotificationCenter.default.addObserver(
            self,
            selector: #selector(batteryStateDidChange),
            name: UIDevice.batteryStateDidChangeNotification,
            object: nil
        )

        NotificationCenter.default.addObserver(
            self,
            selector: #selector(batteryLevelDidChange),
            name: UIDevice.batteryLevelDidChangeNotification,
            object: nil
        )

        NotificationCenter.default.addObserver(
            self,
            selector: #selector(powerStateDidChange),
            name: NSNotification.Name.NSProcessInfoPowerStateDidChange,
            object: nil
        )
    }

    private func setupNetworkMonitoring() {
        networkMonitor.pathUpdateHandler = { [weak self] path in
            DispatchQueue.main.async {
                if path.status == .satisfied {
                    if path.usesInterfaceType(.wifi) {
                        self?.currentNetworkType = .wifi
                    } else if path.usesInterfaceType(.cellular) {
                        self?.currentNetworkType = .cellular
                    } else {
                        self?.currentNetworkType = .unknown
                    }
                } else {
                    self?.currentNetworkType = .none
                }
                self?.updateStatus()
            }
        }
        networkMonitor.start(queue: networkQueue)
    }

    // MARK: - Public Methods

    /// 开始监控设备状态
    func startMonitoring() {
        // DeviceStatusCollector 在 init 时已经设置好所有监听器
        // 这里只需要确保状态是最新的
        updateStatus()
        print("📱 [DeviceStatus] 启动监控")
    }

    /// 停止监控
    func stopMonitoring() {
        // DeviceStatusCollector 的监听器是持久的，不需要停止
        // 但保留此方法以保持接口一致性
        print("📱 [DeviceStatus] 停止监控（监听器保持活跃）")
    }

    // MARK: - Notifications

    @objc private func batteryStateDidChange() {
        updateStatus()
    }

    @objc private func batteryLevelDidChange() {
        updateStatus()
    }

    @objc private func powerStateDidChange() {
        updateStatus()
    }

    // MARK: - Update Status

    func updateStatus() {
        let device = UIDevice.current

        // 电池状态
        let batteryState: BatteryState
        switch device.batteryState {
        case .unknown:
            batteryState = .unknown
        case .unplugged:
            batteryState = .unplugged
        case .charging:
            batteryState = .charging
        case .full:
            batteryState = .full
        @unknown default:
            batteryState = .unknown
        }

        // 低电量模式
        let isLowPowerMode = ProcessInfo.processInfo.isLowPowerModeEnabled

        // 专注模式/勿扰模式 - 通过检查通知设置来推断
        let isFocusModeOn = checkFocusMode()

        // 耳机连接状态
        let isHeadphonesConnected = checkHeadphonesConnected()

        // 屏幕亮度
        let screenBrightness = UIScreen.main.brightness

        // 蓝牙设备类型
        let bluetoothDeviceType = getBluetoothDeviceType()

        currentStatus = DeviceStatus(
            batteryLevel: device.batteryLevel,
            batteryState: batteryState,
            isLowPowerMode: isLowPowerMode,
            isFocusModeOn: isFocusModeOn,
            isHeadphonesConnected: isHeadphonesConnected,
            networkType: currentNetworkType,
            screenBrightness: Float(screenBrightness),
            bluetoothDeviceType: bluetoothDeviceType,
            wifiSSID: currentStatus.wifiSSID  // 保留上次的 WiFi SSID（异步更新）
        )
    }

    // MARK: - Check Focus Mode

    private func checkFocusMode() -> Bool {
        // iOS 没有直接 API 检查勿扰模式
        // 但可以通过一些间接方式推断（这里简化处理）
        // 实际上需要用户授权通知权限后才能更准确判断
        return false
    }

    // MARK: - Check Headphones

    private func checkHeadphonesConnected() -> Bool {
        let audioSession = AVAudioSession.sharedInstance()
        let currentRoute = audioSession.currentRoute

        for output in currentRoute.outputs {
            let portType = output.portType
            if portType == .headphones ||
               portType == .bluetoothA2DP ||
               portType == .bluetoothHFP ||
               portType == .bluetoothLE ||
               portType == .airPlay {
                return true
            }
        }
        return false
    }

    // MARK: - Public Methods

    func getCurrentStatus() -> DeviceStatus {
        updateStatus()
        return currentStatus
    }

    // MARK: - Bluetooth Device Type

    private func getBluetoothDeviceType() -> String? {
        let audioSession = AVAudioSession.sharedInstance()
        let currentRoute = audioSession.currentRoute

        for output in currentRoute.outputs {
            switch output.portType {
            case .bluetoothA2DP, .bluetoothHFP, .bluetoothLE:
                let name = output.portName.lowercased()
                if name.contains("car") || name.contains("vehicle") || name.contains("carplay") {
                    return "car"
                }
                return "headphones"
            case .airPlay:
                return "speaker"
            case .headphones:
                return "headphones"
            default:
                continue
            }
        }
        return nil
    }

    // MARK: - WiFi SSID

    func fetchWiFiSSID() async -> String? {
        await withCheckedContinuation { continuation in
            NEHotspotNetwork.fetchCurrent { network in
                continuation.resume(returning: network?.ssid)
            }
        }
    }

    /// 异步获取完整状态（包含 WiFi SSID）
    func getCurrentStatusAsync() async -> DeviceStatus {
        updateStatus()
        let ssid = await fetchWiFiSSID()
        var status = currentStatus
        status.wifiSSID = ssid
        return status
    }

    // 转换为可上报的字典格式
    func toReportData() -> [String: Any] {
        let status = currentStatus
        return [
            "battery_level": Int(status.batteryLevel * 100),
            "battery_state": status.batteryState.rawValue,
            "is_charging": status.isCharging,
            "is_low_power_mode": status.isLowPowerMode,
            "is_focus_mode_on": status.isFocusModeOn,
            "is_headphones_connected": status.isHeadphonesConnected,
            "network_type": status.networkType.rawValue,
            "screen_brightness": Int(status.screenBrightness * 100)
        ]
    }
}
