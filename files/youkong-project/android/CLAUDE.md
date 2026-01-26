# CLAUDE.md - Android 客户端

> ⚠️ **先阅读** `../CLAUDE.md` **了解共享定义**

---

## 我的职责

- 实现 Android 原生 Jetpack Compose 界面
- 调用后端 API
- 本地数据缓存 (Room)
- FCM 推送通知处理
- 微信 SDK 集成（登录/分享）

---

## 技术栈

```
Language:    Kotlin 1.9+
MinSDK:      26 (Android 8.0)
UI:          Jetpack Compose 1.5+
Architecture: MVVM + Clean Architecture
DI:          Hilt 2.48+
Network:     Retrofit + OkHttp
Storage:     Room 2.6+
Async:       Kotlin Coroutines + Flow
Image:       Coil 2.5+
Push:        Firebase Cloud Messaging
```

---

## 项目结构

```
app/src/main/java/com/youkong/
├── di/
│   ├── AppModule.kt
│   ├── NetworkModule.kt
│   └── DatabaseModule.kt
├── data/
│   ├── remote/
│   │   ├── api/
│   │   │   └── YouKongApi.kt
│   │   └── dto/
│   │       ├── AvailabilityDto.kt
│   │       ├── CircleDto.kt
│   │       └── UserDto.kt
│   ├── local/
│   │   ├── database/
│   │   │   └── YouKongDatabase.kt
│   │   ├── dao/
│   │   │   ├── AvailabilityDao.kt
│   │   │   └── CircleDao.kt
│   │   └── entity/
│   │       ├── AvailabilityEntity.kt
│   │       └── CircleEntity.kt
│   └── repository/
│       ├── AvailabilityRepositoryImpl.kt
│       └── CircleRepositoryImpl.kt
├── domain/
│   ├── model/
│   │   ├── User.kt
│   │   ├── Circle.kt
│   │   ├── Availability.kt
│   │   └── Message.kt
│   ├── repository/
│   │   ├── AvailabilityRepository.kt
│   │   └── CircleRepository.kt
│   └── usecase/
│       ├── PublishAvailabilityUseCase.kt
│       ├── GetFriendsAvailabilityUseCase.kt
│       ├── CreateCircleUseCase.kt
│       └── AIAnalyzeContactsUseCase.kt
├── presentation/
│   ├── navigation/
│   │   └── NavGraph.kt
│   ├── theme/
│   │   ├── Color.kt
│   │   ├── Type.kt
│   │   └── Theme.kt
│   ├── components/
│   │   ├── AvailabilityCard.kt
│   │   ├── CircleChip.kt
│   │   ├── PrimaryButton.kt
│   │   └── TimeRangePicker.kt
│   ├── home/
│   │   ├── HomeScreen.kt
│   │   └── HomeViewModel.kt
│   ├── publish/
│   │   ├── PublishScreen.kt
│   │   ├── PublishViewModel.kt
│   │   ├── TimeSelectionStep.kt
│   │   ├── LocationSelectionStep.kt
│   │   └── CircleSelectionStep.kt
│   ├── circles/
│   │   ├── CirclesScreen.kt
│   │   ├── CirclesViewModel.kt
│   │   └── CircleDetailScreen.kt
│   ├── chat/
│   │   ├── ChatScreen.kt
│   │   └── ChatViewModel.kt
│   └── profile/
│       ├── ProfileScreen.kt
│       └── ProfileViewModel.kt
└── YouKongApplication.kt
```

---

## 关键代码模板

### Retrofit API

```kotlin
// data/remote/api/YouKongApi.kt
interface YouKongApi {
    @GET("availabilities/friends")
    suspend fun getFriendsAvailabilities(
        @Query("page") page: Int = 1,
        @Query("size") size: Int = 20
    ): ApiResponse<PaginatedResponse<AvailabilityDto>>
    
    @POST("availabilities")
    suspend fun publishAvailability(
        @Body request: CreateAvailabilityRequest
    ): ApiResponse<AvailabilityDto>
    
    @DELETE("availabilities/{id}")
    suspend fun cancelAvailability(@Path("id") id: String): ApiResponse<Unit>
    
    @GET("circles")
    suspend fun getCircles(): ApiResponse<List<CircleDto>>
    
    @POST("circles")
    suspend fun createCircle(@Body request: CreateCircleRequest): ApiResponse<CircleDto>
    
    @POST("ai/analyze-contacts")
    suspend fun analyzeContacts(@Body request: AnalyzeContactsRequest): ApiResponse<AICircleSuggestions>
}
```

### ViewModel 模板

```kotlin
// presentation/home/HomeViewModel.kt
@HiltViewModel
class HomeViewModel @Inject constructor(
    private val getAvailabilitiesUseCase: GetFriendsAvailabilityUseCase,
    private val cancelAvailabilityUseCase: CancelAvailabilityUseCase
) : ViewModel() {
    
    private val _uiState = MutableStateFlow(HomeUiState())
    val uiState: StateFlow<HomeUiState> = _uiState.asStateFlow()
    
    init {
        loadData()
    }
    
    fun loadData() {
        viewModelScope.launch {
            _uiState.update { it.copy(isLoading = true) }
            
            try {
                val all = getAvailabilitiesUseCase()
                val today = Calendar.getInstance().apply {
                    set(Calendar.HOUR_OF_DAY, 0)
                    set(Calendar.MINUTE, 0)
                }.time
                val tomorrow = Calendar.getInstance().apply {
                    add(Calendar.DAY_OF_MONTH, 1)
                    set(Calendar.HOUR_OF_DAY, 0)
                    set(Calendar.MINUTE, 0)
                }.time
                
                _uiState.update {
                    it.copy(
                        isLoading = false,
                        todayAvailabilities = all.filter { a -> a.startTime >= today && a.startTime < tomorrow },
                        laterAvailabilities = all.filter { a -> a.startTime >= tomorrow }
                    )
                }
            } catch (e: Exception) {
                _uiState.update { it.copy(isLoading = false, error = e.message) }
            }
        }
    }
}

data class HomeUiState(
    val isLoading: Boolean = false,
    val myAvailability: Availability? = null,
    val todayAvailabilities: List<AvailabilityWithUser> = emptyList(),
    val laterAvailabilities: List<AvailabilityWithUser> = emptyList(),
    val error: String? = null
)
```

### Compose UI 模板

```kotlin
// presentation/home/HomeScreen.kt
@Composable
fun HomeScreen(
    viewModel: HomeViewModel = hiltViewModel(),
    onNavigateToPublish: () -> Unit,
    onNavigateToChat: (String) -> Unit
) {
    val uiState by viewModel.uiState.collectAsStateWithLifecycle()
    
    Scaffold(
        floatingActionButton = {
            FloatingActionButton(
                onClick = onNavigateToPublish,
                containerColor = Primary
            ) {
                Icon(Icons.Default.Add, contentDescription = "发布有空")
            }
        }
    ) { paddingValues ->
        LazyColumn(
            modifier = Modifier.padding(paddingValues),
            contentPadding = PaddingValues(16.dp),
            verticalArrangement = Arrangement.spacedBy(12.dp)
        ) {
            // 我的有空状态
            uiState.myAvailability?.let { availability ->
                item {
                    MyAvailabilityCard(
                        availability = availability,
                        onCancel = { viewModel.cancelAvailability() }
                    )
                }
            }
            
            // 今天有空的朋友
            item {
                Text("🟢 今天有空的朋友", style = MaterialTheme.typography.titleMedium)
            }
            items(uiState.todayAvailabilities) { availability ->
                AvailabilityCard(
                    availability = availability,
                    onInvite = { onNavigateToChat(availability.user.id) }
                )
            }
            
            // 之后有空
            item {
                Text("🟡 之后有空", style = MaterialTheme.typography.titleMedium)
            }
            items(uiState.laterAvailabilities) { availability ->
                AvailabilityCard(
                    availability = availability,
                    dimmed = true,
                    onInvite = { onNavigateToChat(availability.user.id) }
                )
            }
        }
    }
}
```

---

## 色彩定义

```kotlin
// presentation/theme/Color.kt
val Primary = Color(0xFF10B981)
val PrimaryDark = Color(0xFF059669)
val Secondary = Color(0xFF14B8A6)

val Gray50 = Color(0xFFF9FAFB)
val Gray100 = Color(0xFFF3F4F6)
val Gray200 = Color(0xFFE5E7EB)
val Gray400 = Color(0xFF9CA3AF)
val Gray500 = Color(0xFF6B7280)
val Gray600 = Color(0xFF4B5563)
val Gray800 = Color(0xFF1F2937)

val CirclePink = Color(0xFFEC4899)
val CircleOrange = Color(0xFFF97316)
val CircleBlue = Color(0xFF3B82F6)
val CircleGreen = Color(0xFF22C55E)
val CirclePurple = Color(0xFF8B5CF6)
```

---

## 依赖配置

```kotlin
// build.gradle.kts (app)
dependencies {
    // Compose
    implementation(platform("androidx.compose:compose-bom:2024.01.00"))
    implementation("androidx.compose.ui:ui")
    implementation("androidx.compose.material3:material3")
    
    // Hilt
    implementation("com.google.dagger:hilt-android:2.48")
    kapt("com.google.dagger:hilt-compiler:2.48")
    
    // Retrofit
    implementation("com.squareup.retrofit2:retrofit:2.9.0")
    implementation("com.squareup.retrofit2:converter-gson:2.9.0")
    
    // Room
    implementation("androidx.room:room-runtime:2.6.0")
    implementation("androidx.room:room-ktx:2.6.0")
    kapt("androidx.room:room-compiler:2.6.0")
    
    // Coil
    implementation("io.coil-kt:coil-compose:2.5.0")
    
    // Firebase
    implementation(platform("com.google.firebase:firebase-bom:32.7.0"))
    implementation("com.google.firebase:firebase-messaging-ktx")
}
```

---

## 待处理事件

检查 `../shared/events/` 目录获取待实现的功能。

---

*版本: 1.0.0*
