package com.youkong.feature.home.screen

import com.youkong.core.network.api.AgentApi
import com.youkong.core.network.api.OnboardingStatusOptionsRequest
import com.youkong.core.network.api.ProfileApi
import com.youkong.core.network.api.SimpleProfileRequest
import com.youkong.core.network.model.SelectStatusRequest
import com.youkong.core.network.model.StatusOption
import javax.inject.Inject
import javax.inject.Singleton

/**
 * 引导流程数据仓库
 */
@Singleton
class OnboardingRepository @Inject constructor(
    private val profileApi: ProfileApi,
    private val agentApi: AgentApi
) {

    /**
     * 获取引导流程的状态选项
     * @param profileType 用户角色类型
     * @param city 当前城市（可选，用于增强推理）
     */
    suspend fun getOnboardingStatusOptions(profileType: String, city: String? = null): List<StatusOption> {
        val response = profileApi.getOnboardingStatusOptions(
            OnboardingStatusOptionsRequest(profileType, city)
        )
        return response.data?.options?.map { StatusOption(it.emoji, it.status) } ?: emptyList()
    }

    /**
     * 保存简化的用户画像
     */
    suspend fun saveSimpleProfile(profileType: String, gender: String? = null, mbti: String? = null) {
        profileApi.saveSimpleProfile(SimpleProfileRequest(profileType, gender, mbti))
    }

    /**
     * 选择状态并记录
     */
    suspend fun selectStatus(request: SelectStatusRequest) {
        agentApi.selectStatus(request)
    }
}
