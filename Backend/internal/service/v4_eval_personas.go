package service

// ========== V4 评估：用户画像定义 ==========

// EvalPersona 用户画像
type EvalPersona struct {
	ID          string // 画像标识 P1-P10
	Name        string // 画像名称
	Description string // 描述
	TestFocus   string // 测试重点

	// 模拟上下文（注入到 session 中）
	CurrentScheduleItems []EvalScheduleItem // 当前时刻表
	Summary              string             // 历史摘要（模拟已有对话上下文）
}

// EvalScheduleItem 评估用的时刻表条目
type EvalScheduleItem struct {
	StartTime string
	EndTime   string
	Emoji     string
	Status    string
}

// AllPersonas 返回全部 10 个用户画像
func AllPersonas() []EvalPersona {
	return []EvalPersona{
		personaP1(),
		personaP2(),
		personaP3(),
		personaP4(),
		personaP5(),
		personaP6(),
		personaP7(),
		personaP8(),
		personaP9(),
		personaP10(),
	}
}

// GetPersona 根据 ID 获取画像
func GetPersona(id string) *EvalPersona {
	for _, p := range AllPersonas() {
		if p.ID == id {
			return &p
		}
	}
	return nil
}

// ========== P1: 上班族小王 ==========
func personaP1() EvalPersona {
	return EvalPersona{
		ID:          "P1",
		Name:        "上班族小王",
		Description: "互联网公司 PM，9-6 工作，经常加班开会",
		TestFocus:   "日程创建、时间冲突处理",
		CurrentScheduleItems: []EvalScheduleItem{
			{StartTime: "09:00", EndTime: "10:00", Emoji: "💼", Status: "晨会"},
			{StartTime: "10:00", EndTime: "12:00", Emoji: "💻", Status: "写需求文档"},
			{StartTime: "12:00", EndTime: "13:00", Emoji: "🍽️", Status: "午餐"},
			{StartTime: "14:00", EndTime: "16:00", Emoji: "💼", Status: "项目评审"},
			{StartTime: "16:00", EndTime: "18:00", Emoji: "💻", Status: "处理邮件"},
		},
		Summary: "用户是互联网公司PM，工作日通常9点到18点上班。之前创建过明天的会议安排。",
	}
}

// ========== P2: 学生小李 ==========
func personaP2() EvalPersona {
	return EvalPersona{
		ID:          "P2",
		Name:        "学生小李",
		Description: "大三学生，课程灵活，周末兼职",
		TestFocus:   "模糊时间推理、全天安排",
		CurrentScheduleItems: []EvalScheduleItem{
			{StartTime: "08:00", EndTime: "10:00", Emoji: "📚", Status: "高数课"},
			{StartTime: "14:00", EndTime: "16:00", Emoji: "📚", Status: "英语课"},
		},
		Summary: "用户是大学生，课程不多，下午经常有空。周末在奶茶店兼职。",
	}
}

// ========== P3: 自由职业者阿明 ==========
func personaP3() EvalPersona {
	return EvalPersona{
		ID:          "P3",
		Name:        "自由职业者阿明",
		Description: "设计师，作息不规律，经常熬夜",
		TestFocus:   "跨日时间、非常规时间",
		CurrentScheduleItems: []EvalScheduleItem{
			{StartTime: "11:00", EndTime: "13:00", Emoji: "☕", Status: "起床+早午餐"},
			{StartTime: "15:00", EndTime: "19:00", Emoji: "🎨", Status: "设计项目"},
		},
		Summary: "用户是自由职业设计师，通常中午才起床，经常工作到凌晨。上次提到在赶一个设计稿。",
	}
}

// ========== P4: 宝妈小张 ==========
func personaP4() EvalPersona {
	return EvalPersona{
		ID:          "P4",
		Name:        "宝妈小张",
		Description: "全职妈妈，围绕孩子安排时间",
		TestFocus:   "碎片化时间、频繁修改",
		CurrentScheduleItems: []EvalScheduleItem{
			{StartTime: "07:00", EndTime: "08:00", Emoji: "🍼", Status: "送孩子上学"},
			{StartTime: "08:30", EndTime: "11:30", Emoji: "🏠", Status: "打扫做家务"},
			{StartTime: "11:30", EndTime: "12:30", Emoji: "🍽️", Status: "准备午饭"},
			{StartTime: "15:00", EndTime: "16:00", Emoji: "🏫", Status: "接孩子放学"},
			{StartTime: "16:30", EndTime: "18:00", Emoji: "📚", Status: "辅导作业"},
		},
		Summary: "用户是全职妈妈，时间围绕孩子安排。上次修改过接孩子的时间从15:30改到15:00。",
	}
}

// ========== P5: 社交达人小美 ==========
func personaP5() EvalPersona {
	return EvalPersona{
		ID:          "P5",
		Name:        "社交达人小美",
		Description: "人脉广，频繁约人吃饭聚会",
		TestFocus:   "好友查询、批量邀请",
		CurrentScheduleItems: []EvalScheduleItem{
			{StartTime: "10:00", EndTime: "12:00", Emoji: "💼", Status: "工作"},
			{StartTime: "12:00", EndTime: "13:30", Emoji: "🍽️", Status: "和同事午餐"},
			{StartTime: "19:00", EndTime: "21:00", Emoji: "🎉", Status: "约朋友聚餐"},
		},
		Summary: "用户社交活跃，经常约朋友吃饭聚会。上次查询了有空的朋友，其中小红和小明都有空。",
	}
}

// ========== P6: 极简用户老赵 ==========
func personaP6() EvalPersona {
	return EvalPersona{
		ID:          "P6",
		Name:        "极简用户老赵",
		Description: "不喜欢废话，一句话说完",
		TestFocus:   "极短输入解析",
		CurrentScheduleItems: []EvalScheduleItem{
			{StartTime: "09:00", EndTime: "17:00", Emoji: "💼", Status: "上班"},
		},
		Summary: "",
	}
}

// ========== P7: 话唠用户小陈 ==========
func personaP7() EvalPersona {
	return EvalPersona{
		ID:          "P7",
		Name:        "话唠用户小陈",
		Description: "描述啰嗦，一段话包含多个意图",
		TestFocus:   "复合意图拆解",
		CurrentScheduleItems: []EvalScheduleItem{
			{StartTime: "09:00", EndTime: "12:00", Emoji: "💼", Status: "开会"},
			{StartTime: "13:00", EndTime: "17:00", Emoji: "💻", Status: "写代码"},
		},
		Summary: "用户表达比较冗长，经常一段话里混合多个请求。之前在讨论明天的安排。",
	}
}

// ========== P8: 方言口语用户 ==========
func personaP8() EvalPersona {
	return EvalPersona{
		ID:          "P8",
		Name:        "方言口语用户",
		Description: "带口语化、网络用语",
		TestFocus:   "非标准表达理解",
		CurrentScheduleItems: []EvalScheduleItem{
			{StartTime: "10:00", EndTime: "12:00", Emoji: "💼", Status: "搬砖"},
			{StartTime: "14:00", EndTime: "18:00", Emoji: "💼", Status: "继续搬砖"},
		},
		Summary: "",
	}
}

// ========== P9: 犹豫反复用户 ==========
func personaP9() EvalPersona {
	return EvalPersona{
		ID:          "P9",
		Name:        "犹豫反复用户",
		Description: "频繁修改、取消、重来",
		TestFocus:   "状态管理、撤销",
		CurrentScheduleItems: []EvalScheduleItem{
			{StartTime: "14:00", EndTime: "16:00", Emoji: "💼", Status: "开会"},
		},
		Summary: "用户经常改变主意。之前创建了下午开会的安排，又想改时间，改了两次。",
	}
}

// ========== P10: 压力测试用户 ==========
func personaP10() EvalPersona {
	return EvalPersona{
		ID:          "P10",
		Name:        "压力测试用户",
		Description: "恶意输入、注入攻击、超长文本",
		TestFocus:   "安全鲁棒性",
		CurrentScheduleItems: []EvalScheduleItem{},
		Summary:              "",
	}
}
