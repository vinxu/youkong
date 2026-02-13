package service

import "math/rand"

// ========== 压缩时间模拟测试：对话模板 ==========

// ConversationTemplate 对话模板
type ConversationTemplate struct {
	ID       string   // "simple_set", "multi_round", "complex_day"
	Name     string   // "简单设定", "多轮修改", "复杂行程"
	Messages []string // 按顺序发送的消息

	// 验证期望
	ExpectedItemCount int      // 期望生成的 schedule item 数量
	ExpectedTimes     []string // 期望的时间段（HH:MM 格式），按顺序
	ExpectedKeywords  []string // 期望 item status 包含的关键词（任一匹配即可）
}

// AllConversationTemplates 返回所有对话模板（按场景分类）
func AllConversationTemplates() []ConversationTemplate {
	templates := make([]ConversationTemplate, 0, len(simpleTemplates)+len(multiRoundTemplates)+len(complexTemplates))
	templates = append(templates, simpleTemplates...)
	templates = append(templates, multiRoundTemplates...)
	templates = append(templates, complexTemplates...)
	return templates
}

// RandomConversationTemplates 每类随机取 1 个，返回 3 个模板
func RandomConversationTemplates() []ConversationTemplate {
	result := make([]ConversationTemplate, 0, 3)
	if len(simpleTemplates) > 0 {
		result = append(result, simpleTemplates[rand.Intn(len(simpleTemplates))])
	}
	if len(multiRoundTemplates) > 0 {
		result = append(result, multiRoundTemplates[rand.Intn(len(multiRoundTemplates))])
	}
	if len(complexTemplates) > 0 {
		result = append(result, complexTemplates[rand.Intn(len(complexTemplates))])
	}
	return result
}

// ========== 简单设定模板 ==========

var simpleTemplates = []ConversationTemplate{
	{
		ID: "simple_meeting", Name: "简单-下午开会",
		Messages:          []string{"下午两点到五点开会"},
		ExpectedItemCount: 1,
		ExpectedTimes:     []string{"14:00"},
		ExpectedKeywords:  []string{"开会", "会议"},
	},
	{
		ID: "simple_home", Name: "简单-晚上看剧",
		Messages:          []string{"晚上在家看剧"},
		ExpectedItemCount: 1,
		ExpectedTimes:     []string{"19:00", "20:00"},
		ExpectedKeywords:  []string{"看剧", "追剧", "刷剧"},
	},
	{
		ID: "simple_exercise", Name: "简单-上午跑步",
		Messages:          []string{"明天上午去跑步"},
		ExpectedItemCount: 1,
		ExpectedTimes:     []string{"08:00", "09:00", "07:00"},
		ExpectedKeywords:  []string{"跑步", "运动", "锻炼"},
	},
	{
		ID: "simple_work", Name: "简单-上午办公",
		Messages:          []string{"上午九点到十二点在公司办公"},
		ExpectedItemCount: 1,
		ExpectedTimes:     []string{"09:00"},
		ExpectedKeywords:  []string{"办公", "工作", "上班"},
	},
	{
		ID: "simple_lunch", Name: "简单-中午吃饭",
		Messages:          []string{"中午十二点和朋友吃饭"},
		ExpectedItemCount: 1,
		ExpectedTimes:     []string{"12:00"},
		ExpectedKeywords:  []string{"吃饭", "午餐", "聚餐"},
	},
}

// ========== 多轮修改模板 ==========

var multiRoundTemplates = []ConversationTemplate{
	{
		ID: "multi_change_time", Name: "多轮-修改时间",
		Messages: []string{
			"下午三点到六点去健身",
			"改成四点开始吧",
		},
		ExpectedItemCount: 1,
		ExpectedTimes:     []string{"16:00"},
		ExpectedKeywords:  []string{"健身", "运动", "锻炼", "gym"},
	},
	{
		ID: "multi_add_item", Name: "多轮-追加行程",
		Messages: []string{
			"上午十点开会",
			"下午两点再加一个客户拜访",
		},
		ExpectedItemCount: 2,
		ExpectedTimes:     []string{"10:00", "14:00"},
		ExpectedKeywords:  []string{"开会", "拜访", "客户"},
	},
	{
		ID: "multi_cancel_item", Name: "多轮-取消行程",
		Messages: []string{
			"晚上七点和老王吃饭，九点去看电影",
			"电影不去了，取消掉",
		},
		ExpectedItemCount: 1,
		ExpectedTimes:     []string{"19:00"},
		ExpectedKeywords:  []string{"吃饭"},
	},
}

// ========== 复杂行程模板 ==========

var complexTemplates = []ConversationTemplate{
	{
		ID: "complex_full_day", Name: "复杂-全天安排",
		Messages: []string{
			"明天的安排是：早上9点开会，中午和同事吃饭，下午3点见客户，晚上回家休息",
		},
		ExpectedItemCount: 4,
		ExpectedTimes:     []string{"09:00", "12:00", "15:00", "18:00", "19:00"},
		ExpectedKeywords:  []string{"开会", "吃饭", "客户", "休息"},
	},
	{
		ID: "complex_weekend", Name: "复杂-周末安排",
		Messages: []string{
			"周末安排：上午去健身房锻炼，中午在外面吃饭，下午在家休息看书，晚上和朋友聚会",
		},
		ExpectedItemCount: 4,
		ExpectedTimes:     []string{"09:00", "10:00", "12:00", "14:00", "19:00", "20:00"},
		ExpectedKeywords:  []string{"健身", "锻炼", "吃饭", "看书", "休息", "聚会"},
	},
	{
		ID: "complex_business", Name: "复杂-商务日程",
		Messages: []string{
			"明天出差，早上八点的飞机，十一点到了去客户公司开会，下午三点回酒店，晚上七点商务晚宴",
		},
		ExpectedItemCount: 4,
		ExpectedTimes:     []string{"08:00", "11:00", "15:00", "19:00"},
		ExpectedKeywords:  []string{"飞机", "出差", "开会", "客户", "酒店", "晚宴", "商务"},
	},
}
