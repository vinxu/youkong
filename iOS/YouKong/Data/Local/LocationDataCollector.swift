import Foundation
import Combine
import CoreLocation

// MARK: - Location Data Collector

class LocationDataCollector: NSObject, ObservableObject {
    static let shared = LocationDataCollector()

    @Published private(set) var currentStatus: LocationStatus = .unknown
    @Published private(set) var isMonitoring = false

    private let locationManager = CLLocationManager()
    private let geocoder = CLGeocoder()
    private(set) var currentLocation: CLLocation?
    private var arrivalTime: Date?

    // 反向地理编码缓存
    private var lastGeocodedLocation: CLLocation?
    private var lastGeocodedPlaceName: String?
    private var lastGeocodedCity: String?

    // 学习到的地点（公开给调试视图）
    private(set) var homeLocation: CLLocation?
    private(set) var workLocation: CLLocation?

    private let homeLocationKey = "learnedHomeLocation"
    private let workLocationKey = "learnedWorkLocation"

    // 多夜确认制：候选 home 位置（Phase 3a）
    private let homeCandidateKey = "homeCandidateLocation"
    private let homeCandidateNightCountKey = "homeCandidateNightCount"
    private let homeCandidateLastDateKey = "homeCandidateLastDate"
    private let requiredNightsForHome = 3 // 连续 3 夜才确认为 home

    // 多日确认制：候选 work 位置
    private let workCandidateKey = "workCandidateLocation"
    private let workCandidateDayCountKey = "workCandidateDayCount"
    private let workCandidateLastDateKey = "workCandidateLastDate"
    private let requiredDaysForWork = 3 // 3 个工作日白天在同一位置才确认为 work

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

    var isLocationAuthorized: Bool {
        let status = locationManager.authorizationStatus
        return status == .authorizedAlways || status == .authorizedWhenInUse
    }

    func getCurrentStatus() -> LocationStatus {
        guard isLocationAuthorized else {
            return .unknown
        }
        return currentStatus
    }

    /// 刷新当前位置（请求一次性位置更新）
    func refreshLocation() async -> CLLocation? {
        locationManager.requestLocation()
        // 等待一小段时间让位置更新回调触发
        try? await Task.sleep(nanoseconds: 1_000_000_000) // 1秒
        return currentLocation
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

    /// 学习家的位置（多夜确认制：连续 3 夜在同一位置才确认为 home）
    func learnHomeLocation() {
        guard let location = currentLocation else { return }

        let hour = Calendar.current.component(.hour, from: Date())
        guard hour >= 22 || hour <= 7 else { return }

        // 如果仍在已确认的 home 200m 范围内，不触发学习
        if let home = homeLocation, location.distance(from: home) < 200 {
            return
        }

        let today = Calendar.current.startOfDay(for: Date())
        let todayStr = ISO8601DateFormatter().string(from: today)

        // 加载候选位置
        let candidateLocation = loadCandidateLocation()
        let candidateNightCount = UserDefaults.standard.integer(forKey: homeCandidateNightCountKey)
        let candidateLastDate = UserDefaults.standard.string(forKey: homeCandidateLastDateKey) ?? ""

        // 今天已经记录过，跳过（每天只计一次）
        if candidateLastDate == todayStr {
            return
        }

        if let candidate = candidateLocation, location.distance(from: candidate) < 200 {
            // 在候选位置 200m 范围内 → 累加过夜计数
            let newCount = candidateNightCount + 1
            UserDefaults.standard.set(newCount, forKey: homeCandidateNightCountKey)
            UserDefaults.standard.set(todayStr, forKey: homeCandidateLastDateKey)

            if newCount >= requiredNightsForHome {
                // 达到阈值 → 确认为 home
                homeLocation = candidate
                saveLearnedPlace(candidate, forKey: homeLocationKey)
                // 清除候选
                UserDefaults.standard.removeObject(forKey: homeCandidateKey)
                UserDefaults.standard.removeObject(forKey: homeCandidateNightCountKey)
                UserDefaults.standard.removeObject(forKey: homeCandidateLastDateKey)
            }
        } else {
            // 新位置 → 重置候选
            saveCandidateLocation(location)
            UserDefaults.standard.set(1, forKey: homeCandidateNightCountKey)
            UserDefaults.standard.set(todayStr, forKey: homeCandidateLastDateKey)
        }
    }

    private func saveCandidateLocation(_ location: CLLocation) {
        let data: [String: Double] = [
            "latitude": location.coordinate.latitude,
            "longitude": location.coordinate.longitude
        ]
        UserDefaults.standard.set(data, forKey: homeCandidateKey)
    }

    private func loadCandidateLocation() -> CLLocation? {
        guard let data = UserDefaults.standard.dictionary(forKey: homeCandidateKey) as? [String: Double],
              let lat = data["latitude"],
              let lng = data["longitude"] else {
            return nil
        }
        return CLLocation(latitude: lat, longitude: lng)
    }

    /// 学习公司位置（多日确认制：连续 3 个工作日白天在同一位置才确认为 work）
    func learnWorkLocation() {
        guard let location = currentLocation else { return }

        let calendar = Calendar.current
        let weekday = calendar.component(.weekday, from: Date())
        let hour = calendar.component(.hour, from: Date())

        // 周一到周五，上午9点到下午6点
        guard weekday >= 2 && weekday <= 6 && hour >= 9 && hour <= 18 else { return }

        // 确保不是家的位置
        if let home = homeLocation, location.distance(from: home) < 500 {
            return
        }

        // 如果已在确认的 work 200m 范围内，不触发学习
        if let work = workLocation, location.distance(from: work) < 200 {
            return
        }

        let today = calendar.startOfDay(for: Date())
        let todayStr = ISO8601DateFormatter().string(from: today)

        // 今天已经记录过，跳过（每天只计一次）
        let candidateLastDate = UserDefaults.standard.string(forKey: workCandidateLastDateKey) ?? ""
        if candidateLastDate == todayStr {
            return
        }

        // 加载候选位置
        let candidateLocation = loadWorkCandidateLocation()
        let candidateDayCount = UserDefaults.standard.integer(forKey: workCandidateDayCountKey)

        if let candidate = candidateLocation, location.distance(from: candidate) < 300 {
            // 在候选位置 300m 范围内 → 累加天数
            let newCount = candidateDayCount + 1
            UserDefaults.standard.set(newCount, forKey: workCandidateDayCountKey)
            UserDefaults.standard.set(todayStr, forKey: workCandidateLastDateKey)

            if newCount >= requiredDaysForWork {
                // 达到阈值 → 确认为 work
                workLocation = candidate
                saveLearnedPlace(candidate, forKey: workLocationKey)
                // 清除候选
                UserDefaults.standard.removeObject(forKey: workCandidateKey)
                UserDefaults.standard.removeObject(forKey: workCandidateDayCountKey)
                UserDefaults.standard.removeObject(forKey: workCandidateLastDateKey)
            }
        } else {
            // 新位置 → 重置候选
            saveWorkCandidateLocation(location)
            UserDefaults.standard.set(1, forKey: workCandidateDayCountKey)
            UserDefaults.standard.set(todayStr, forKey: workCandidateLastDateKey)
        }
    }

    private func saveWorkCandidateLocation(_ location: CLLocation) {
        let data: [String: Double] = [
            "latitude": location.coordinate.latitude,
            "longitude": location.coordinate.longitude
        ]
        UserDefaults.standard.set(data, forKey: workCandidateKey)
    }

    private func loadWorkCandidateLocation() -> CLLocation? {
        guard let data = UserDefaults.standard.dictionary(forKey: workCandidateKey) as? [String: Double],
              let lat = data["latitude"],
              let lng = data["longitude"] else {
            return nil
        }
        return CLLocation(latitude: lat, longitude: lng)
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

        // 迁移：旧版本的 work 位置不可靠（单次就学习），清除并要求重新学习
        let workMigrationKey = "workLocationMigratedV2"
        if !UserDefaults.standard.bool(forKey: workMigrationKey) {
            UserDefaults.standard.removeObject(forKey: workLocationKey)
            UserDefaults.standard.removeObject(forKey: workCandidateKey)
            UserDefaults.standard.removeObject(forKey: workCandidateDayCountKey)
            UserDefaults.standard.removeObject(forKey: workCandidateLastDateKey)
            UserDefaults.standard.set(true, forKey: workMigrationKey)
            workLocation = nil
            return
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

        // 更新状态（先用缓存的地点名称）
        currentStatus = LocationStatus(
            placeType: placeType,
            atPlaceSinceMinutes: atPlaceSinceMinutes,
            placeName: lastGeocodedPlaceName,
            latitude: location.coordinate.latitude,
            longitude: location.coordinate.longitude,
            city: lastGeocodedCity
        )

        currentLocation = location

        // 尝试学习地点
        learnHomeLocation()
        learnWorkLocation()

        // 异步获取地点名称（避免频繁调用）
        reverseGeocodeIfNeeded(location)
    }

    // MARK: - Reverse Geocoding

    private func reverseGeocodeIfNeeded(_ location: CLLocation) {
        // 如果距离上次地理编码位置超过 200 米，才重新编码
        if let lastLocation = lastGeocodedLocation {
            let distance = location.distance(from: lastLocation)
            if distance < 200 {
                return
            }
        }

        lastGeocodedLocation = location

        geocoder.reverseGeocodeLocation(location) { [weak self] placemarks, error in
            guard let self = self, let placemark = placemarks?.first else {
                return
            }

            // 构建地点名称
            var placeName = ""
            if let name = placemark.name {
                placeName = name
            } else if let thoroughfare = placemark.thoroughfare {
                placeName = thoroughfare
                if let subThoroughfare = placemark.subThoroughfare {
                    placeName = "\(thoroughfare)\(subThoroughfare)"
                }
            } else if let locality = placemark.locality {
                placeName = locality
            }

            DispatchQueue.main.async {
                self.lastGeocodedPlaceName = placeName.isEmpty ? nil : placeName
                self.lastGeocodedCity = placemark.locality

                // 更新状态包含地点名称
                self.currentStatus = LocationStatus(
                    placeType: self.currentStatus.placeType,
                    atPlaceSinceMinutes: self.currentStatus.atPlaceSinceMinutes,
                    placeName: self.lastGeocodedPlaceName,
                    latitude: location.coordinate.latitude,
                    longitude: location.coordinate.longitude,
                    city: self.lastGeocodedCity
                )
            }
        }
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
