package com.youkong.feature.home.viewmodel

import androidx.lifecycle.ViewModel
import com.youkong.core.agent.collector.CalendarCollector
import com.youkong.core.agent.collector.DeviceStateCollector
import com.youkong.core.agent.collector.LocationCollector
import com.youkong.core.agent.collector.MovementCollector
import dagger.hilt.android.lifecycle.HiltViewModel
import javax.inject.Inject

/**
 * Onboarding ViewModel — 仅用于向 OnboardingScreen 提供 Hilt 注入的 Collector 实例
 */
@HiltViewModel
class OnboardingViewModel @Inject constructor(
    val locationCollector: LocationCollector,
    val movementCollector: MovementCollector,
    val calendarCollector: CalendarCollector,
    val deviceStateCollector: DeviceStateCollector
) : ViewModel()
