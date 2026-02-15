package com.youkong.core.agent.collector

import android.app.AppOpsManager
import android.app.usage.UsageStatsManager
import android.content.Context
import android.os.PowerManager
import android.os.Process
import android.util.Log
import com.youkong.core.network.model.ScreenDataRequest
import dagger.hilt.android.qualifiers.ApplicationContext
import javax.inject.Inject
import javax.inject.Singleton

private const val TAG = "ScreenUsageCollector"

@Singleton
class ScreenUsageCollector @Inject constructor(
    @ApplicationContext private val context: Context,
) {
    private val usageStatsManager: UsageStatsManager by lazy {
        context.getSystemService(Context.USAGE_STATS_SERVICE) as UsageStatsManager
    }

    private val powerManager: PowerManager by lazy {
        context.getSystemService(Context.POWER_SERVICE) as PowerManager
    }

    fun hasPermission(): Boolean {
        return try {
            val appOps = context.getSystemService(Context.APP_OPS_SERVICE) as AppOpsManager
            val mode = appOps.unsafeCheckOpNoThrow(
                AppOpsManager.OPSTR_GET_USAGE_STATS,
                Process.myUid(),
                context.packageName,
            )
            mode == AppOpsManager.MODE_ALLOWED
        } catch (e: Exception) {
            false
        }
    }

    fun collect(): ScreenDataRequest? {
        if (!hasPermission()) {
            Log.d(TAG, "无 USAGE_STATS 权限")
            return null
        }

        val now = System.currentTimeMillis()
        val isScreenOn = powerManager.isInteractive

        // 查询最近 30 分钟的使用记录
        val stats = try {
            usageStatsManager.queryUsageStats(
                UsageStatsManager.INTERVAL_BEST,
                now - 30 * 60 * 1000,
                now,
            )
        } catch (e: Exception) {
            Log.w(TAG, "查询 UsageStats 失败: ${e.message}")
            return null
        }

        if (stats.isNullOrEmpty()) {
            Log.d(TAG, "无使用记录")
            return ScreenDataRequest(
                isActive = isScreenOn,
                activityType = "idle",
                sessionDurationMinutes = 0,
                lastActiveMinutesAgo = if (isScreenOn) 0 else 5,
            )
        }

        // 找到最近使用的 app
        val recentApp = stats
            .filter { it.totalTimeInForeground > 0 }
            .maxByOrNull { it.lastTimeUsed }

        val lastActiveMinutesAgo = if (recentApp != null) {
            ((now - recentApp.lastTimeUsed) / 60000).toInt().coerceAtLeast(0)
        } else {
            5
        }

        // 屏幕是否活跃：屏幕亮着 且 最近1分钟内有app使用
        val isActive = isScreenOn && lastActiveMinutesAgo <= 1

        // 分类最近使用的 app（仅屏幕活跃时才分类，否则 idle）
        val activityType = if (isActive && recentApp != null) {
            classifyApp(recentApp.packageName)
        } else {
            "idle"
        }

        // 计算当前会话时长（最近 app 的前台时长，分钟）
        val sessionMinutes = if (recentApp != null && lastActiveMinutesAgo <= 1) {
            (recentApp.totalTimeInForeground / 60000).toInt().coerceAtLeast(1)
        } else {
            0
        }

        Log.d(TAG, "collect: active=$isActive, type=$activityType, session=${sessionMinutes}min, " +
                "lastActive=${lastActiveMinutesAgo}min, app=${recentApp?.packageName}")

        return ScreenDataRequest(
            isActive = isActive,
            activityType = activityType,
            sessionDurationMinutes = sessionMinutes,
            lastActiveMinutesAgo = lastActiveMinutesAgo,
        )
    }

    companion object {
        // 社交/通讯类
        private val COMMUNICATION_APPS = setOf(
            "com.tencent.mm",           // 微信
            "com.tencent.mobileqq",     // QQ
            "com.tencent.tim",          // TIM
            "com.alibaba.android.rimet", // 钉钉
            "com.ss.android.lark",      // 飞书
            "com.tencent.wework",       // 企业微信
            "com.whatsapp",             // WhatsApp
            "org.telegram.messenger",   // Telegram
            "com.facebook.orca",        // Messenger
            "com.immomo.momo",          // 陌陌
            "com.sina.weibo",           // 微博
            "com.xiaomi.channel",       // 小米通话
        )

        // 娱乐类
        private val ENTERTAINMENT_APPS = setOf(
            "com.ss.android.ugc.aweme",   // 抖音
            "com.kuaishou.nebula",        // 快手
            "tv.danmaku.bili",            // B站
            "com.youku.phone",            // 优酷
            "com.tencent.qqlive",         // 腾讯视频
            "com.qiyi.video",             // 爱奇艺
            "com.netease.cloudmusic",     // 网易云音乐
            "com.kugou.android",          // 酷狗
            "com.tencent.qqmusic",        // QQ音乐
            "com.spotify.music",          // Spotify
            "com.zhihu.android",          // 知乎
            "com.ss.android.article.news", // 今日头条
            "com.tencent.news",           // 腾讯新闻
            "com.jingdong.app.reader",    // 京东读书
            "com.dragon.read",            // 番茄小说
            "com.tencent.tmgp.sgame",     // 王者荣耀
            "com.miHoYo.Yuanshen",        // 原神
            "com.tencent.ig",             // PUBG
            "com.supercell.clashofclans", // COC
            "com.xiaomi.gamecenter",      // 小米游戏
        )

        // 生产力/工作类
        private val PRODUCTIVITY_APPS = setOf(
            "com.microsoft.office.outlook", // Outlook
            "com.google.android.gm",        // Gmail
            "com.microsoft.teams",          // Teams
            "com.alibaba.android.rimet",    // 钉钉 (同时算通讯+生产力)
            "com.ss.android.lark",          // 飞书
            "com.tencent.wework",           // 企业微信
            "com.microsoft.office.word",    // Word
            "com.microsoft.office.excel",   // Excel
            "com.google.android.apps.docs", // Google Docs
            "com.kingsoft.moffice_pro",     // WPS
            "com.github.android",           // GitHub
            "com.notion.id",               // Notion
            "com.todoist",                 // Todoist
            "com.evernote",                // Evernote
        )

        // 购物类（归为 entertainment）
        private val SHOPPING_APPS = setOf(
            "com.taobao.taobao",     // 淘宝
            "com.jingdong.app.mall", // 京东
            "com.xunmeng.pinduoduo", // 拼多多
            "com.tmall.wireless",    // 天猫
            "com.meituan",           // 美团
            "com.sankuai.meituan.takeoutnew", // 美团外卖
            "me.ele",                // 饿了么
        )

        fun classifyApp(packageName: String): String {
            return when {
                PRODUCTIVITY_APPS.contains(packageName) -> "productivity"
                COMMUNICATION_APPS.contains(packageName) -> "communication"
                ENTERTAINMENT_APPS.contains(packageName) -> "entertainment"
                SHOPPING_APPS.contains(packageName) -> "entertainment"
                // 系统应用和其他
                packageName.startsWith("com.android.") -> "idle"
                packageName.startsWith("com.google.android.") -> "idle"
                packageName.startsWith("com.huawei.") -> "idle"
                packageName.startsWith("com.miui.") -> "idle"
                packageName.startsWith("com.oppo.") -> "idle"
                packageName.startsWith("com.vivo.") -> "idle"
                packageName.startsWith("com.youkong.") -> "idle" // 自己的 app 不计入
                else -> "other" // 未知 app 不预设分类
            }
        }
    }
}
