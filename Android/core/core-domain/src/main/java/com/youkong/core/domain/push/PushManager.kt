package com.youkong.core.domain.push

/**
 * 推送管理器接口
 * 定义推送相关的操作，具体实现由 app 模块提供
 */
interface PushManager {

    /**
     * 绑定用户账号到推送服务
     * 登录成功后调用，用于接收定向推送
     * @param userId 用户 ID
     */
    suspend fun bindAccount(userId: String)

    /**
     * 解绑用户账号
     * 退出登录时调用
     */
    suspend fun unbindAccount()
}
