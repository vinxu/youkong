package model

// ContactMatchRequest 通讯录匹配请求
type ContactMatchRequest struct {
	PhoneHashes []string `json:"phone_hashes" binding:"required,min=1,max=1000"`
}

// ContactMatchResult 单个匹配结果
type ContactMatchResult struct {
	PhoneHash  string `json:"phone_hash"`
	User       *UserProfile `json:"user"`
	IsFriend   bool   `json:"is_friend"`   // 是否已是好友
}

// ContactMatchResponse 通讯录匹配响应
type ContactMatchResponse struct {
	Matches    []ContactMatchResult `json:"matches"`
	TotalFound int                  `json:"total_found"`
}

// BatchAddFriendsRequest 批量添加好友请求
type BatchAddFriendsRequest struct {
	UserIDs []string `json:"user_ids" binding:"required,min=1,max=100"`
}

// BatchAddFriendsResponse 批量添加好友响应
type BatchAddFriendsResponse struct {
	Added   int      `json:"added"`
	Skipped int      `json:"skipped"` // 已是好友，跳过
	UserIDs []string `json:"user_ids"`
}
