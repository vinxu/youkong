import Foundation
import Combine
import CoreLocation
import EventKit
import CoreMotion
import UIKit

// MARK: - Permission Manager

@MainActor
class PermissionManager: NSObject, ObservableObject {
    static let shared = PermissionManager()

    @Published var status = PermissionStatus.initial
    @Published var isChecking = false
    @Published var isAlwaysLocationGranted = false

    private let locationManager = CLLocationManager()
    private var locationContinuation: CheckedContinuation<Bool, Never>?
    private let eventStore = EKEventStore()

    override private init() {
        super.init()
        locationManager.delegate = self
    }

    // MARK: - Check All Permissions

    func checkAllPermissions() async {
        isChecking = true
        defer { isChecking = false }

        // 检查位置权限
        status.location = checkLocationPermission()
        isAlwaysLocationGranted = locationManager.authorizationStatus == .authorizedAlways

        // 检查日历权限
        status.calendar = checkCalendarPermission()

        // 检查运动权限
        status.motion = checkMotionPermission()
    }

    // MARK: - Location Permission

    private func checkLocationPermission() -> Bool {
        switch locationManager.authorizationStatus {
        case .authorizedAlways, .authorizedWhenInUse:
            return true
        default:
            return false
        }
    }

    func requestLocationPermission() async throws -> Bool {
        let currentStatus = locationManager.authorizationStatus

        switch currentStatus {
        case .notDetermined:
            // 请求权限
            return await withCheckedContinuation { continuation in
                self.locationContinuation = continuation
                self.locationManager.requestWhenInUseAuthorization()
            }
        case .authorizedAlways, .authorizedWhenInUse:
            status.location = true
            return true
        case .denied, .restricted:
            status.location = false
            return false
        @unknown default:
            return false
        }
    }

    // MARK: - Always Location Permission

    func requestAlwaysLocationPermission() async -> Bool {
        let currentStatus = locationManager.authorizationStatus

        switch currentStatus {
        case .authorizedAlways:
            isAlwaysLocationGranted = true
            return true
        case .authorizedWhenInUse:
            // Upgrade from whenInUse to always
            return await withCheckedContinuation { continuation in
                self.locationContinuation = continuation
                self.locationManager.requestAlwaysAuthorization()
            }
        default:
            return false
        }
    }

    // MARK: - Calendar Permission

    private func checkCalendarPermission() -> Bool {
        let authStatus = EKEventStore.authorizationStatus(for: .event)
        print("[Permission] Calendar status: \(authStatus.rawValue)")
        if #available(iOS 17.0, *) {
            return authStatus == .fullAccess || authStatus == .authorized
        } else {
            return authStatus == .authorized
        }
    }

    func requestCalendarPermission() async throws -> Bool {
        let currentStatus = EKEventStore.authorizationStatus(for: .event)
        print("[Permission] Requesting calendar, current status: \(currentStatus.rawValue)")

        // 如果已经被拒绝，需要引导用户去设置
        if currentStatus == .denied || currentStatus == .restricted {
            print("[Permission] Calendar denied, need to go to Settings")
            status.calendar = false
            if let url = URL(string: UIApplication.openSettingsURLString) {
                await UIApplication.shared.open(url)
            }
            return false
        }

        // 使用 completion handler 版本，因为 async 版本可能有问题
        return await withCheckedContinuation { continuation in
            if #available(iOS 17.0, *) {
                print("[Permission] Using iOS 17+ API with completion handler")
                eventStore.requestFullAccessToEvents { [weak self] granted, error in
                    print("[Permission] Calendar callback: granted=\(granted), error=\(String(describing: error))")
                    Task { @MainActor in
                        self?.status.calendar = granted
                        continuation.resume(returning: granted)
                    }
                }
            } else {
                print("[Permission] Using legacy API with completion handler")
                eventStore.requestAccess(to: .event) { [weak self] granted, error in
                    print("[Permission] Calendar callback: granted=\(granted), error=\(String(describing: error))")
                    Task { @MainActor in
                        self?.status.calendar = granted
                        continuation.resume(returning: granted)
                    }
                }
            }
        }
    }

    // MARK: - Motion Permission

    private func checkMotionPermission() -> Bool {
        guard CMMotionActivityManager.isActivityAvailable() else {
            print("[Permission] Motion not available on this device")
            return false
        }
        let authStatus = CMMotionActivityManager.authorizationStatus()
        print("[Permission] Motion status: \(authStatus.rawValue)")
        return authStatus == .authorized
    }

    func requestMotionPermission() async -> Bool {
        guard CMMotionActivityManager.isActivityAvailable() else {
            print("[Permission] Motion not available")
            status.motion = false
            return false
        }

        let currentStatus = CMMotionActivityManager.authorizationStatus()
        print("[Permission] Requesting motion, current status: \(currentStatus.rawValue)")

        // 如果已经被拒绝，需要引导用户去设置
        if currentStatus == .denied || currentStatus == .restricted {
            print("[Permission] Motion denied, need to go to Settings")
            status.motion = false
            await MainActor.run {
                if let url = URL(string: UIApplication.openSettingsURLString) {
                    UIApplication.shared.open(url)
                }
            }
            return false
        }

        let activityManager = CMMotionActivityManager()

        return await withCheckedContinuation { continuation in
            let now = Date()
            let oneHourAgo = now.addingTimeInterval(-3600)

            activityManager.queryActivityStarting(from: oneHourAgo, to: now, to: .main) { [weak self] _, error in
                Task { @MainActor in
                    if let error = error as NSError?,
                       error.code == Int(CMErrorMotionActivityNotAuthorized.rawValue) {
                        print("[Permission] Motion query denied")
                        self?.status.motion = false
                        continuation.resume(returning: false)
                    } else {
                        print("[Permission] Motion query succeeded")
                        self?.status.motion = true
                        continuation.resume(returning: true)
                    }
                }
            }
        }
    }

    // MARK: - Request All Permissions

    func requestAllPermissions() async {
        // 依次请求所有权限

        do {
            _ = try await requestLocationPermission()
        } catch {
            print("Location permission error: \(error)")
        }

        do {
            _ = try await requestCalendarPermission()
        } catch {
            print("Calendar permission error: \(error)")
        }

        _ = await requestMotionPermission()
    }
}

// MARK: - CLLocationManagerDelegate

extension PermissionManager: CLLocationManagerDelegate {
    nonisolated func locationManagerDidChangeAuthorization(_ manager: CLLocationManager) {
        Task { @MainActor in
            let granted = manager.authorizationStatus == .authorizedWhenInUse ||
                         manager.authorizationStatus == .authorizedAlways
            status.location = granted
            isAlwaysLocationGranted = manager.authorizationStatus == .authorizedAlways

            if let continuation = locationContinuation {
                locationContinuation = nil
                continuation.resume(returning: granted)
            }
        }
    }
}
