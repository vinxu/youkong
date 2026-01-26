# CLAUDE.md - iOS 客户端

> ⚠️ **先阅读** `../CLAUDE.md` **了解共享定义**

---

## 我的职责

- 实现 iOS 原生 SwiftUI 界面
- 调用后端 API
- 本地数据缓存 (Core Data)
- APNs 推送通知处理
- 微信 SDK 集成（登录/分享）

---

## 技术栈

```
Language:    Swift 5.9+
MinOS:       iOS 16.0
UI:          SwiftUI
Architecture: MVVM + Clean Architecture
DI:          Factory
Network:     URLSession + Async/Await
Storage:     Core Data / SwiftData
Image:       Kingfisher
Push:        APNs
```

---

## 项目结构

```
YouKong/
├── App/
│   ├── YouKongApp.swift
│   └── AppDelegate.swift
├── Core/
│   ├── DI/
│   │   └── Container.swift
│   ├── Extensions/
│   │   ├── Color+Ext.swift
│   │   ├── Date+Ext.swift
│   │   └── View+Ext.swift
│   └── Utils/
│       └── Formatters.swift
├── Data/
│   ├── Network/
│   │   ├── APIClient.swift
│   │   ├── APIEndpoint.swift
│   │   └── DTOs/
│   │       ├── AvailabilityDTO.swift
│   │       ├── CircleDTO.swift
│   │       └── UserDTO.swift
│   ├── Local/
│   │   ├── CoreDataStack.swift
│   │   └── Entities/
│   └── Repository/
│       ├── AvailabilityRepositoryImpl.swift
│       └── CircleRepositoryImpl.swift
├── Domain/
│   ├── Models/
│   │   ├── User.swift
│   │   ├── Circle.swift
│   │   ├── Availability.swift
│   │   └── Message.swift
│   ├── Repository/
│   │   ├── AvailabilityRepository.swift
│   │   └── CircleRepository.swift
│   └── UseCase/
│       ├── PublishAvailabilityUseCase.swift
│       ├── GetFriendsAvailabilityUseCase.swift
│       ├── CreateCircleUseCase.swift
│       └── AIAnalyzeContactsUseCase.swift
├── Presentation/
│   ├── Theme/
│   │   ├── Colors.swift
│   │   ├── Typography.swift
│   │   └── Spacing.swift
│   ├── Components/
│   │   ├── AvailabilityCard.swift
│   │   ├── CircleChip.swift
│   │   ├── PrimaryButton.swift
│   │   └── TimeRangePicker.swift
│   ├── Home/
│   │   ├── HomeView.swift
│   │   └── HomeViewModel.swift
│   ├── Publish/
│   │   ├── PublishView.swift
│   │   ├── PublishViewModel.swift
│   │   ├── TimeSelectionView.swift
│   │   ├── LocationSelectionView.swift
│   │   └── CircleSelectionView.swift
│   ├── Circles/
│   │   ├── CirclesView.swift
│   │   ├── CirclesViewModel.swift
│   │   └── CircleDetailView.swift
│   ├── Chat/
│   │   ├── ChatView.swift
│   │   └── ChatViewModel.swift
│   └── Profile/
│       ├── ProfileView.swift
│       └── ProfileViewModel.swift
└── Resources/
    ├── Assets.xcassets
    └── Localizable.strings
```

---

## 关键代码模板

### APIClient

```swift
// Data/Network/APIClient.swift
actor APIClient {
    static let shared = APIClient()
    private let baseURL = URL(string: "https://api.youkong.app/v1")!
    private var accessToken: String?
    
    func request<T: Decodable>(
        endpoint: APIEndpoint,
        responseType: T.Type
    ) async throws -> T {
        var request = URLRequest(url: baseURL.appendingPathComponent(endpoint.path))
        request.httpMethod = endpoint.method.rawValue
        request.setValue("application/json", forHTTPHeaderField: "Content-Type")
        
        if let token = accessToken {
            request.setValue("Bearer \(token)", forHTTPHeaderField: "Authorization")
        }
        
        if let body = endpoint.body {
            request.httpBody = try JSONEncoder().encode(body)
        }
        
        let (data, _) = try await URLSession.shared.data(for: request)
        let response = try JSONDecoder().decode(APIResponse<T>.self, from: data)
        return response.data
    }
}
```

### ViewModel 模板

```swift
// Presentation/Home/HomeViewModel.swift
@MainActor
class HomeViewModel: ObservableObject {
    @Published var todayAvailabilities: [AvailabilityWithUser] = []
    @Published var laterAvailabilities: [AvailabilityWithUser] = []
    @Published var myAvailability: Availability?
    @Published var isLoading = false
    @Published var error: Error?
    
    private let getAvailabilitiesUseCase: GetFriendsAvailabilityUseCase
    
    init(getAvailabilitiesUseCase: GetFriendsAvailabilityUseCase = .init()) {
        self.getAvailabilitiesUseCase = getAvailabilitiesUseCase
        Task { await loadData() }
    }
    
    func loadData() async {
        isLoading = true
        defer { isLoading = false }
        
        do {
            let all = try await getAvailabilitiesUseCase.execute()
            let today = Calendar.current.startOfDay(for: Date())
            let tomorrow = Calendar.current.date(byAdding: .day, value: 1, to: today)!
            
            todayAvailabilities = all.filter { $0.startTime >= today && $0.startTime < tomorrow }
            laterAvailabilities = all.filter { $0.startTime >= tomorrow }
        } catch {
            self.error = error
        }
    }
}
```

### SwiftUI View 模板

```swift
// Presentation/Home/HomeView.swift
struct HomeView: View {
    @StateObject private var viewModel = HomeViewModel()
    @State private var showPublishSheet = false
    
    var body: some View {
        NavigationStack {
            ZStack(alignment: .bottomTrailing) {
                ScrollView {
                    LazyVStack(spacing: Spacing.lg) {
                        // 我的有空状态
                        if let my = viewModel.myAvailability {
                            MyAvailabilityCard(availability: my)
                        }
                        
                        // 今天有空的朋友
                        Section("🟢 今天有空的朋友") {
                            ForEach(viewModel.todayAvailabilities) { item in
                                AvailabilityCard(availability: item)
                            }
                        }
                        
                        // 之后有空
                        Section("🟡 之后有空") {
                            ForEach(viewModel.laterAvailabilities) { item in
                                AvailabilityCard(availability: item, dimmed: true)
                            }
                        }
                    }
                    .padding()
                }
                .refreshable { await viewModel.loadData() }
                
                // FAB
                Button(action: { showPublishSheet = true }) {
                    Image(systemName: "plus")
                        .font(.title2.weight(.semibold))
                        .foregroundColor(.white)
                        .frame(width: 56, height: 56)
                        .background(Color.primary)
                        .clipShape(Circle())
                }
                .padding()
            }
            .navigationTitle("有空")
        }
        .sheet(isPresented: $showPublishSheet) {
            PublishView()
        }
    }
}
```

---

## 色彩定义

```swift
// Presentation/Theme/Colors.swift
extension Color {
    static let primary = Color(hex: "10B981")
    static let primaryDark = Color(hex: "059669")
    static let secondary = Color(hex: "14B8A6")
    
    static let gray50 = Color(hex: "F9FAFB")
    static let gray100 = Color(hex: "F3F4F6")
    static let gray200 = Color(hex: "E5E7EB")
    static let gray400 = Color(hex: "9CA3AF")
    static let gray500 = Color(hex: "6B7280")
    static let gray600 = Color(hex: "4B5563")
    static let gray800 = Color(hex: "1F2937")
}
```

---

## 待处理事件

检查 `../shared/events/` 目录获取待实现的功能。

---

*版本: 1.0.0*
