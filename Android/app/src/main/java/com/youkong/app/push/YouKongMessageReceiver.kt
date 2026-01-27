package com.youkong.app.push

import android.content.Context
import android.content.Intent
import android.util.Log
import androidx.core.net.toUri
import com.tencent.android.tpush.XGPushBaseReceiver
import com.tencent.android.tpush.XGPushClickedResult
import com.tencent.android.tpush.XGPushRegisterResult
import com.tencent.android.tpush.XGPushShowedResult
import com.tencent.android.tpush.XGPushTextMessage
import com.youkong.app.MainActivity
import org.json.JSONObject

/**
 * TPNS 推送消息接收器
 * 处理通知点击事件，解析 conversation_id 跳转到聊天页面
 */
class YouKongMessageReceiver : XGPushBaseReceiver() {

    companion object {
        private const val TAG = "YouKongMessageReceiver"
    }

    /**
     * 通知点击回调
     * 解析自定义参数，跳转到对应页面
     */
    override fun onNotificationClickedResult(
        context: Context,
        message: XGPushClickedResult?
    ) {
        if (message == null) {
            Log.w(TAG, "收到空的点击消息")
            openMainActivity(context)
            return
        }

        Log.d(TAG, "通知被点击: ${message.title}, customContent: ${message.customContent}")

        val customContent = message.customContent
        if (customContent.isNullOrEmpty()) {
            openMainActivity(context)
            return
        }

        try {
            val json = JSONObject(customContent)
            val conversationId = json.optString("conversation_id")

            if (conversationId.isNotEmpty()) {
                // 跳转到聊天页面
                openChatScreen(context, conversationId)
            } else {
                // 没有 conversation_id，打开主页
                openMainActivity(context)
            }
        } catch (e: Exception) {
            Log.e(TAG, "解析 customContent 失败", e)
            openMainActivity(context)
        }
    }

    /**
     * 打开聊天页面
     */
    private fun openChatScreen(context: Context, userId: String) {
        val intent = Intent(
            Intent.ACTION_VIEW,
            "youkong://chat/$userId".toUri()
        ).apply {
            flags = Intent.FLAG_ACTIVITY_NEW_TASK or Intent.FLAG_ACTIVITY_CLEAR_TOP
        }
        context.startActivity(intent)
    }

    /**
     * 打开主页面
     */
    private fun openMainActivity(context: Context) {
        val intent = Intent(context, MainActivity::class.java).apply {
            flags = Intent.FLAG_ACTIVITY_NEW_TASK or Intent.FLAG_ACTIVITY_CLEAR_TOP
        }
        context.startActivity(intent)
    }

    /**
     * 注册结果回调
     */
    override fun onRegisterResult(
        context: Context,
        errorCode: Int,
        message: XGPushRegisterResult?
    ) {
        if (errorCode == XGPushBaseReceiver.SUCCESS) {
            Log.d(TAG, "注册成功，token: ${message?.token}")
        } else {
            Log.e(TAG, "注册失败，errorCode: $errorCode")
        }
    }

    /**
     * 透传消息回调
     */
    override fun onTextMessage(context: Context, message: XGPushTextMessage?) {
        Log.d(TAG, "收到透传消息: ${message?.content}")
    }

    /**
     * 通知展示回调
     */
    override fun onNotificationShowedResult(
        context: Context,
        result: XGPushShowedResult?
    ) {
        Log.d(TAG, "通知已展示: ${result?.title}")
    }

    override fun onUnregisterResult(context: Context, errorCode: Int) {
        Log.d(TAG, "注销结果，errorCode: $errorCode")
    }

    override fun onSetTagResult(context: Context, errorCode: Int, tagName: String?) {
        Log.d(TAG, "设置标签结果，errorCode: $errorCode, tag: $tagName")
    }

    override fun onDeleteTagResult(context: Context, errorCode: Int, tagName: String?) {
        Log.d(TAG, "删除标签结果，errorCode: $errorCode, tag: $tagName")
    }

    override fun onSetAccountResult(context: Context, errorCode: Int, account: String?) {
        Log.d(TAG, "设置账号结果，errorCode: $errorCode, account: $account")
    }

    override fun onDeleteAccountResult(context: Context, errorCode: Int, account: String?) {
        Log.d(TAG, "删除账号结果，errorCode: $errorCode, account: $account")
    }

    override fun onSetAttributeResult(
        context: Context,
        errorCode: Int,
        key: String?
    ) {
        Log.d(TAG, "设置属性结果，errorCode: $errorCode, key: $key")
    }

    override fun onQueryTagsResult(
        context: Context,
        errorCode: Int,
        data: String?,
        operateName: String?
    ) {
        Log.d(TAG, "查询标签结果，errorCode: $errorCode")
    }

    override fun onDeleteAttributeResult(
        context: Context,
        errorCode: Int,
        key: String?
    ) {
        Log.d(TAG, "删除属性结果，errorCode: $errorCode, key: $key")
    }
}
