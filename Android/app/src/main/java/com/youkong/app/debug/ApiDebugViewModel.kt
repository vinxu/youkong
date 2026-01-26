package com.youkong.app.debug

import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import com.youkong.core.datastore.TokenManager
import com.youkong.core.network.api.AuthApi
import com.youkong.core.network.api.AvailabilityApi
import com.youkong.core.network.api.CircleApi
import com.youkong.core.network.api.UserApi
import com.youkong.core.network.model.CreateCircleRequest
import com.youkong.core.network.model.SendSmsRequest
import com.youkong.core.network.model.VerifySmsRequest
import dagger.hilt.android.lifecycle.HiltViewModel
import kotlinx.coroutines.launch
import javax.inject.Inject

@HiltViewModel
class ApiDebugViewModel @Inject constructor(
    private val authApi: AuthApi,
    private val userApi: UserApi,
    private val circleApi: CircleApi,
    private val availabilityApi: AvailabilityApi,
    private val tokenManager: TokenManager,
) : ViewModel() {

    // ===== 认证接口 =====

    fun sendSms(phone: String, callback: (String) -> Unit) {
        viewModelScope.launch {
            try {
                val response = authApi.sendSms(SendSmsRequest(phone))
                callback("code=${response.code}, message=${response.message}")
            } catch (e: Exception) {
                callback("ERROR: ${e.message}")
            }
        }
    }

    fun verifySms(phone: String, code: String, callback: (String) -> Unit) {
        viewModelScope.launch {
            try {
                val response = authApi.verifySms(VerifySmsRequest(phone, code))
                val data = response.data
                if (response.isSuccess && data != null) {
                    // 保存 Token
                    tokenManager.saveAccessToken(data.token)
                    callback("登录成功!\ntoken=${data.token.take(50)}...\nuser=${data.user.nickname}\nisNewUser=${data.isNewUser}")
                } else {
                    callback("code=${response.code}, message=${response.message}")
                }
            } catch (e: Exception) {
                callback("ERROR: ${e.message}")
            }
        }
    }

    // ===== 用户接口 =====

    fun getCurrentUser(callback: (String) -> Unit) {
        viewModelScope.launch {
            try {
                val response = userApi.getCurrentUser()
                val data = response.data
                if (response.isSuccess && data != null) {
                    callback("用户信息:\nid=${data.id}\nnickname=${data.nickname}\nphone=${data.phone}")
                } else {
                    callback("code=${response.code}, message=${response.message}")
                }
            } catch (e: Exception) {
                callback("ERROR: ${e.message}")
            }
        }
    }

    fun searchUsers(keyword: String, callback: (String) -> Unit) {
        viewModelScope.launch {
            try {
                val response = userApi.searchUsers(keyword)
                val data = response.data
                if (response.isSuccess && data != null) {
                    callback("搜索结果: ${data.size} 个用户\n${data.joinToString("\n") { "${it.nickname} (${it.id})" }}")
                } else {
                    callback("code=${response.code}, message=${response.message}")
                }
            } catch (e: Exception) {
                callback("ERROR: ${e.message}")
            }
        }
    }

    // ===== 圈子接口 =====

    fun getCircles(callback: (String) -> Unit) {
        viewModelScope.launch {
            try {
                val response = circleApi.getMyCircles()
                val data = response.data
                if (response.isSuccess && data != null) {
                    callback("圈子列表: ${data.size} 个\n${data.joinToString("\n") { "${it.emoji} ${it.name} (${it.id})" }}")
                } else {
                    callback("code=${response.code}, message=${response.message}")
                }
            } catch (e: Exception) {
                callback("ERROR: ${e.message}")
            }
        }
    }

    fun createCircle(callback: (String) -> Unit) {
        viewModelScope.launch {
            try {
                val request = CreateCircleRequest(
                    name = "测试圈子${System.currentTimeMillis() % 10000}",
                    emoji = "\uD83C\uDF89",
                    color = "#EC4899"
                )
                val response = circleApi.createCircle(request)
                val data = response.data
                if (response.isSuccess && data != null) {
                    callback("创建成功!\nid=${data.id}\nname=${data.name}")
                } else {
                    callback("code=${response.code}, message=${response.message}")
                }
            } catch (e: Exception) {
                callback("ERROR: ${e.message}")
            }
        }
    }

    // ===== 有空接口 =====

    fun getFriendsAvailabilities(callback: (String) -> Unit) {
        viewModelScope.launch {
            try {
                val response = availabilityApi.getFriendsAvailabilities()
                val data = response.data
                if (response.isSuccess && data != null) {
                    callback("好友有空: ${data.size} 条\n${data.joinToString("\n") { "${it.user.nickname}: ${it.startTime}" }}")
                } else {
                    callback("code=${response.code}, message=${response.message}")
                }
            } catch (e: Exception) {
                callback("ERROR: ${e.message}")
            }
        }
    }

    fun getMyAvailabilities(callback: (String) -> Unit) {
        viewModelScope.launch {
            try {
                val response = availabilityApi.getMyAvailabilities()
                val data = response.data
                if (response.isSuccess && data != null) {
                    callback("我的有空: ${data.size} 条\n${data.joinToString("\n") { "${it.startTime} - ${it.endTime}" }}")
                } else {
                    callback("code=${response.code}, message=${response.message}")
                }
            } catch (e: Exception) {
                callback("ERROR: ${e.message}")
            }
        }
    }

    // ===== 其他 =====

    fun clearToken() {
        viewModelScope.launch {
            tokenManager.clearTokens()
        }
    }
}
