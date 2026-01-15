package contentcreator

// HotTopic 热点话题
type HotTopic struct {
	Rank      string `json:"rank"`
	Title     string `json:"title"`
	Link      string `json:"link"`
	Hot       string `json:"hot"`        // 热度值
	Source    string `json:"source"`     // 来源
	Timestamp string `json:"timestamp"`  // 时间戳
}

// ContentVersion 内容版本
type ContentVersion struct {
	Style   string `json:"style"`   // stanley, defou, combo
	Content string `json:"content"` // 内容正文
	Hooks   []string `json:"hooks"` // 开头钩子选项
	Score   Score  `json:"score"`   // 潜力评估
}

// Score 内容评分
type Score struct {
	Curiosity     int    `json:"curiosity"`     // 好奇心
	Resonance     int    `json:"resonance"`     // 共鸣度
	Clarity       int    `json:"clarity"`       // 清晰度
	Shareability  int    `json:"shareability"`  // 转发价值
	Total         int    `json:"total"`         // 总分
	Reasoning     string `json:"reasoning"`     // 评分理由
}

// VerificationResult 验证结果
type VerificationResult struct {
	OriginalScore   int      `json:"original_score"`   // 原始得分
	OptimizedScore  int      `json:"optimized_score"`  // 优化后得分
	Improvements    []string `json:"improvements"`     // 改进点
	OptimizedContent string  `json:"optimized_content"` // 优化后内容
	DetailedScores  DetailedScore `json:"detailed_scores"` // 详细评分
}

// DetailedScore 详细评分
type DetailedScore struct {
	Curiosity   ScoreItem `json:"curiosity"`
	Emotion     ScoreItem `json:"emotion"`
	Value       ScoreItem `json:"value"`
	Timeliness  ScoreItem `json:"timeliness"`
	Rhythm      ScoreItem `json:"rhythm"`
	Novelty     ScoreItem `json:"novelty"`
}

// ScoreItem 单项评分
type ScoreItem struct {
	Score    int    `json:"score"`    // 得分
	MaxScore int    `json:"max_score"` // 满分
	Weight   float64 `json:"weight"`   // 权重
	Analysis string `json:"analysis"` // 分析
	Issues   string `json:"issues"`   // 问题
}

// GenerateRequest 生成请求
type GenerateRequest struct {
	Topic      string   `json:"topic"`       // 话题
	RawContent string   `json:"raw_content"` // 原始内容
	Style      string   `json:"style"`       // 风格：stanley, defou, combo, all
	Sources    []HotTopic `json:"sources"`   // 热点来源
}

// TrendAnalysis 热点分析
type TrendAnalysis struct {
	Topics          []HotTopic         `json:"topics"`           // 热点列表
	Recommendations []TopicRecommendation `json:"recommendations"` // 推荐选题
	Summary         string             `json:"summary"`          // 总结
}

// TopicRecommendation 选题推荐
type TopicRecommendation struct {
	Topic       string `json:"topic"`        // 话题
	Reason      string `json:"reason"`       // 推荐理由
	Potential   int    `json:"potential"`    // 潜力评分
	Angle       string `json:"angle"`        // 切入角度
	Source      string `json:"source"`       // 来源
}
