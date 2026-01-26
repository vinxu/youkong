import Foundation
import Combine
import CoreLocation
import Contacts

// MARK: - Permission Manager

@MainActor
class PermissionManager: NSObject, ObservableObject {
    static let shared = PermissionManager()

    @Published var status = PermissionStatus.initial
    @Published var isChecking = false

    private let locationManager = CLLocationManager()
    private var locationContinuation: CheckedContinuation<Bool, Never>?

    override private init() {
        super.init()
        locationManager.delegate = self
    }

    // MARK: - Check All Permissions

    func checkAllPermissions() async {
        isChecking = true
        defer { isChecking = false }

        // 检查屏幕使用时间权限
        status.screenTime = await checkScreenTimePermission()

        // 检查位置权限
        status.location = checkLocationPermission()

        // 检查通讯录权限
        status.contacts = checkContactsPermission()
    }

    // MARK: - Screen Time Permission

    /// 检查屏幕使用时间权限
    private func checkScreenTimePermission() async -> Bool {
        let collector = ScreenDataCollector.shared
        await collector.checkAuthorization()
        return collector.isAuthorized
    }

    func requestScreenTimePermission() async throws -> Bool {
        let collector = ScreenDataCollector.shared
        let granted = await collector.requestAuthorization()
        status.screenTime = granted
        return granted
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

    // MARK: - Contacts Permission

    private func checkContactsPermission() -> Bool {
        let status = CNContactStore.authorizationStatus(for: .contacts)
        return status == .authorized
    }

    func requestContactsPermission() async throws -> Bool {
        let store = CNContactStore()

        do {
            let granted = try await store.requestAccess(for: .contacts)
            status.contacts = granted
            return granted
        } catch {
            status.contacts = false
            throw error
        }
    }

    // MARK: - Request All Permissions

    func requestAllPermissions() async {
        // 依次请求所有权限
        do {
            _ = try await requestScreenTimePermission()
        } catch {
            print("Screen time permission error: \(error)")
        }

        do {
            _ = try await requestLocationPermission()
        } catch {
            print("Location permission error: \(error)")
        }

        do {
            _ = try await requestContactsPermission()
        } catch {
            print("Contacts permission error: \(error)")
        }
    }
}

// MARK: - CLLocationManagerDelegate

extension PermissionManager: CLLocationManagerDelegate {
    nonisolated func locationManagerDidChangeAuthorization(_ manager: CLLocationManager) {
        Task { @MainActor in
            let granted = manager.authorizationStatus == .authorizedWhenInUse ||
                         manager.authorizationStatus == .authorizedAlways
            status.location = granted

            if let continuation = locationContinuation {
                locationContinuation = nil
                continuation.resume(returning: granted)
            }
        }
    }
}
