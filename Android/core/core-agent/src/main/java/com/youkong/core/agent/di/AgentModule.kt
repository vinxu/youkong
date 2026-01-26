package com.youkong.core.agent.di

import com.youkong.core.agent.collector.LocationCollector
import com.youkong.core.agent.collector.LocationDataCollector
import com.youkong.core.agent.collector.ScreenDataCollector
import com.youkong.core.agent.collector.UsageStatsCollector
import dagger.Binds
import dagger.Module
import dagger.hilt.InstallIn
import dagger.hilt.components.SingletonComponent
import javax.inject.Singleton

/**
 * Agent 模块的依赖注入配置
 */
@Module
@InstallIn(SingletonComponent::class)
abstract class AgentModule {

    @Binds
    @Singleton
    abstract fun bindScreenDataCollector(
        impl: UsageStatsCollector
    ): ScreenDataCollector

    @Binds
    @Singleton
    abstract fun bindLocationDataCollector(
        impl: LocationCollector
    ): LocationDataCollector
}
