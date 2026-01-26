import Foundation
import Combine
import CoreLocation

// MARK: - Location Data Collector

class LocationDataCollector: NSObject, ObservableObject {
    static let shared = LocationDataCollector()

    @Published private(set) var currentStatus: LocationStatus = .unknown
    @Published private(set) var isMonitoring = false

    private let locationManager = CLLocationManager()
    private var currentLocation: CLLocation?
    private var arrivalTime: Date?

    // 学习到的地点
    private var homeLocation: CLLocation?
    private var workLocation: CLLocation?

    private let homeLocationKey = "learnedHomeLocation"
    private let workLocationKey = "learnedWorkLocation"

    override private init() {
        super.init()
        locationManager.delegate = self
        locationManager.desiredAccuracy = kCLLocationAccuracyHundredMeters
        locationManager.distanceFilter = 100 // 100米移动才更新
        loadLearnedPlaces()
    }

    // MARK: - Public Methods

    func startMonitoring() {
        guard CLLocationManager.locationServicesEnabled() else { return }

        isMonitoring = true
        locationManager.startUpdatingLocation()
        arrivalTime = Date()
    }

    func stopMonitoring() {
        isMonitoring = false
        locationManager.stopUpdatingLocation()
    }

    func getCurrentStatus() -> LocationStatus {
        return currentStatus
    }

    // MARK: - Place Classification

    func classifyLocation(_ location: CLLocation) -> PlaceType {
        // 检查是否在家
        if let home = homeLocation {
            let distance = location.distance(from: home)
            if distance < 200 { // 200米内视为在家
                return .home
            }
        }

        // 检查是否在公司
        if let work = workLocation {
            let distance = location.distance(from: work)
            if distance < 200 {
                return .work
            }
        }

        // 检查移动速度判断是否在路上
        if location.speed > 2.0 { // 大于 2m/s 视为在移动
            return .transit
        }

        // 默认视为休闲场所
        return .leisure
    }

    // MARK: - Learn Places

    /// 学习家的位置（通常在晚上10点到早上7点停留的地方）
    func learnHomeLocation() {
        guard let location = currentLocation else { return }

        let hour = Calendar.current.component(.hour, from: Date())
        if hour >= 22 || hour <= 7 {
            homeLocation = location
            saveLearnedPlace(location, forKey: homeLocationKey)
        }
    }

    /// 学习公司位置（通常在工作日白天停留的地方）
    func learnWorkLocation() {
        guard let location = currentLocation else { return }

        let calendar = Calendar.current
        let weekday = calendar.component(.weekday, from: Date())
        let hour = calendar.component(.hour, from: Date())

        // 周一到周五，上午9点到下午6点
        if weekday >= 2 && weekday <= 6 && hour >= 9 && hour <= 18 {
            // 确保不是家的位置
            if let home = homeLocation, location.distance(from: home) < 500 {
                return
            }
            workLocation = location
            saveLearnedPlace(location, forKey: workLocationKey)
        }
    }

    // MARK: - Persistence

    private func saveLearnedPlace(_ location: CLLocation, forKey key: String) {
        let data: [String: Double] = [
            "latitude": location.coordinate.latitude,
            "longitude": location.coordinate.longitude
        ]
        UserDefaults.standard.set(data, forKey: key)
    }

    private func loadLearnedPlaces() {
        if let homeData = UserDefaults.standard.dictionary(forKey: homeLocationKey) as? [String: Double],
           let lat = homeData["latitude"],
           let lng = homeData["longitude"] {
            homeLocation = CLLocation(latitude: lat, longitude: lng)
        }

        if let workData = UserDefaults.standard.dictionary(forKey: workLocationKey) as? [String: Double],
           let lat = workData["latitude"],
           let lng = workData["longitude"] {
            workLocation = CLLocation(latitude: lat, longitude: lng)
        }
    }

    // MARK: - Update Status

    private func updateStatus(with location: CLLocation) {
        let placeType = classifyLocation(location)

        // 如果地点类型变化了，重置到达时间
        if placeType != currentStatus.placeType {
            arrivalTime = Date()
        }

        let atPlaceSinceMinutes = Int(Date().timeIntervalSince(arrivalTime ?? Date()) / 60)

        currentStatus = LocationStatus(
            placeType: placeType,
            atPlaceSinceMinutes: atPlaceSinceMinutes
        )

        currentLocation = location

        // 尝试学习地点
        learnHomeLocation()
        learnWorkLocation()
    }
}

// MARK: - CLLocationManagerDelegate

extension LocationDataCollector: CLLocationManagerDelegate {
    func locationManager(_ manager: CLLocationManager, didUpdateLocations locations: [CLLocation]) {
        guard let location = locations.last else { return }
        updateStatus(with: location)
    }

    func locationManager(_ manager: CLLocationManager, didFailWithError error: Error) {
        print("Location error: \(error.localizedDescription)")
    }
}
