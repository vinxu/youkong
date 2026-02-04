package com.youkong.core.network.api

import com.youkong.core.network.model.ApiResponse
import com.youkong.core.network.model.MyScheduleHistoryResponse
import retrofit2.http.GET
import retrofit2.http.Query

/**
 * 时刻表 API 接口
 */
interface ScheduleApi {
    /**
     * 获取我的状态时刻表历史（分页）
     *
     * @param limit 每页数量，默认 20
     * @param beforeDate 分页游标，获取此日期之前的记录（格式: yyyy-MM-dd）
     */
    @GET("agent/my-schedule/history")
    suspend fun getMyScheduleHistory(
        @Query("limit") limit: Int = 20,
        @Query("before_date") beforeDate: String? = null
    ): ApiResponse<MyScheduleHistoryResponse>
}
