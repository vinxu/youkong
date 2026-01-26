package com.youkong.core.agent.collector

import com.youkong.core.agent.model.LocalLocationData
import com.youkong.core.agent.model.LocalScreenData

/**
 * 数据收集器接口
 */
interface DataCollector<T> {
    /**
     * 是否有收集权限
     */
    suspend fun hasPermission(): Boolean

    /**
     * 收集数据
     */
    suspend fun collect(): T?
}

/**
 * 屏幕使用数据收集器接口
 */
interface ScreenDataCollector : DataCollector<LocalScreenData>

/**
 * 位置数据收集器接口
 */
interface LocationDataCollector : DataCollector<LocalLocationData>
