package com.youkong.core.permission.di

import dagger.Module
import dagger.hilt.InstallIn
import dagger.hilt.components.SingletonComponent

/**
 * 权限模块的依赖注入配置
 * PermissionManager 使用 @Singleton 和 @Inject constructor，无需手动提供
 */
@Module
@InstallIn(SingletonComponent::class)
object PermissionModule
