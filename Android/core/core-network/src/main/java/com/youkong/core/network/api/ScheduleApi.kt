package com.youkong.core.network.api

import com.youkong.core.network.model.ApiResponse
import com.youkong.core.network.model.MyScheduleHistoryResponse
import com.youkong.core.network.model.UpdateScheduleItemRequest
import retrofit2.http.Body
import retrofit2.http.DELETE
import retrofit2.http.GET
import retrofit2.http.PUT
import retrofit2.http.Path
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

    /**
     * 更新时刻表条目（修改 emoji 和状态文字）
     */
    @PUT("agent/my-schedule/{date}/item")
    suspend fun updateScheduleItem(
        @Path("date") date: String,
        @Body request: UpdateScheduleItemRequest
    ): ApiResponse<Unit>

    /**
     * 删除时刻表条目
     */
    @DELETE("agent/my-schedule/{date}/item")
    suspend fun deleteScheduleItem(
        @Path("date") date: String,
        @Query("start_time") startTime: String,
        @Query("end_time") endTime: String
    ): ApiResponse<Unit>

    /**
     * 获取好友的时刻表（只含当前和未来条目）
     */
    @GET("agent/user/{userId}/schedule")
    suspend fun getUserSchedule(
        @Path("userId") userId: String
    ): ApiResponse<MyScheduleHistoryResponse>
}
