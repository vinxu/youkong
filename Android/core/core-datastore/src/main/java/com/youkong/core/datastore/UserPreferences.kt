package com.youkong.core.datastore

import android.content.Context
import androidx.datastore.core.DataStore
import androidx.datastore.preferences.core.Preferences
import androidx.datastore.preferences.core.booleanPreferencesKey
import androidx.datastore.preferences.core.edit
import androidx.datastore.preferences.core.stringPreferencesKey
import androidx.datastore.preferences.preferencesDataStore
import dagger.hilt.android.qualifiers.ApplicationContext
import kotlinx.coroutines.flow.Flow
import kotlinx.coroutines.flow.map
import javax.inject.Inject
import javax.inject.Singleton

private val Context.userDataStore: DataStore<Preferences> by preferencesDataStore(name = "user_prefs")

@Singleton
class UserPreferences @Inject constructor(
    @ApplicationContext private val context: Context,
) {
    private object PreferencesKeys {
        val USER_ID = stringPreferencesKey("user_id")
        val USER_PHONE = stringPreferencesKey("user_phone")
        val USER_NICKNAME = stringPreferencesKey("user_nickname")
        val USER_AVATAR = stringPreferencesKey("user_avatar")
        val IS_NEW_USER = booleanPreferencesKey("is_new_user")
        val ONBOARDING_COMPLETED = booleanPreferencesKey("onboarding_completed")
    }

    val userId: Flow<String?> = context.userDataStore.data.map { preferences ->
        preferences[PreferencesKeys.USER_ID]
    }

    val userNickname: Flow<String?> = context.userDataStore.data.map { preferences ->
        preferences[PreferencesKeys.USER_NICKNAME]
    }

    val isNewUser: Flow<Boolean> = context.userDataStore.data.map { preferences ->
        preferences[PreferencesKeys.IS_NEW_USER] ?: false
    }

    val onboardingCompleted: Flow<Boolean> = context.userDataStore.data.map { preferences ->
        preferences[PreferencesKeys.ONBOARDING_COMPLETED] ?: false
    }

    suspend fun saveUser(
        id: String,
        phone: String,
        nickname: String,
        avatar: String?,
        isNewUser: Boolean,
    ) {
        context.userDataStore.edit { preferences ->
            preferences[PreferencesKeys.USER_ID] = id
            preferences[PreferencesKeys.USER_PHONE] = phone
            preferences[PreferencesKeys.USER_NICKNAME] = nickname
            if (avatar != null) {
                preferences[PreferencesKeys.USER_AVATAR] = avatar
            }
            preferences[PreferencesKeys.IS_NEW_USER] = isNewUser
        }
    }

    suspend fun updateNickname(nickname: String) {
        context.userDataStore.edit { preferences ->
            preferences[PreferencesKeys.USER_NICKNAME] = nickname
        }
    }

    suspend fun updateAvatar(avatar: String) {
        context.userDataStore.edit { preferences ->
            preferences[PreferencesKeys.USER_AVATAR] = avatar
        }
    }

    suspend fun setOnboardingCompleted() {
        context.userDataStore.edit { preferences ->
            preferences[PreferencesKeys.ONBOARDING_COMPLETED] = true
            preferences[PreferencesKeys.IS_NEW_USER] = false
        }
    }

    suspend fun clear() {
        context.userDataStore.edit { preferences ->
            preferences.clear()
        }
    }
}
