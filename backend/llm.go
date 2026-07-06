package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

// DashScope API 配置
const (
	dashScopeDefaultAPIURL  = "https://llm-26cli7e69esmbtok.cn-beijing.maas.aliyuncs.com/api/v1/services/aigc/multimodal-generation/generation"
	dashScopeGenerationPath = "/services/aigc/multimodal-generation/generation"
	defaultModel            = "qwen3.7-plus"
	fallbackDashScopeAPIKey = "sk-8ab77da79b894ba6beb61c9190c74602"
)

// DashScope 请求结构（原生 HTTP 调用格式）
type DashScopeRequest struct {
	Model      string              `json:"model"`
	Input      DashScopeInput      `json:"input"`
	Parameters DashScopeParameters `json:"parameters,omitempty"`
}

type DashScopeInput struct {
	Messages []DashScopeMessage `json:"messages"`
}

type DashScopeMessage struct {
	Role    string `json:"role"`
	Content any    `json:"content"`
}

type DashScopeContentPart struct {
	Text string `json:"text,omitempty"`
}

type DashScopeParameters struct {
	ResultFormat        string                   `json:"result_format,omitempty"`
	Temperature         float64                  `json:"temperature,omitempty"`
	MaxCompletionTokens int                      `json:"max_completion_tokens,omitempty"`
	ResponseFormat      *DashScopeResponseFormat `json:"response_format,omitempty"`
	EnableThinking      *bool                    `json:"enable_thinking,omitempty"`
}

type DashScopeResponseFormat struct {
	Type string `json:"type"`
}

type DashScopeStatusCode int

func (c *DashScopeStatusCode) UnmarshalJSON(data []byte) error {
	data = bytes.TrimSpace(data)
	if len(data) == 0 || bytes.Equal(data, []byte("null")) {
		*c = 0
		return nil
	}
	var code int
	if err := json.Unmarshal(data, &code); err == nil {
		*c = DashScopeStatusCode(code)
		return nil
	}
	var text string
	if err := json.Unmarshal(data, &text); err != nil {
		return err
	}
	if strings.TrimSpace(text) == "" {
		*c = 0
		return nil
	}
	parsed, err := strconv.Atoi(text)
	if err != nil {
		return err
	}
	*c = DashScopeStatusCode(parsed)
	return nil
}

// DashScope 响应结构（原生 HTTP 调用格式）
type DashScopeResponse struct {
	StatusCode DashScopeStatusCode `json:"status_code"`
	RequestID  string              `json:"request_id"`
	Code       string              `json:"code"`
	Message    string              `json:"message"`
	Output     struct {
		Text         string `json:"text,omitempty"`
		FinishReason string `json:"finish_reason,omitempty"`
		Choices      []struct {
			FinishReason string                   `json:"finish_reason"`
			Message      DashScopeResponseMessage `json:"message"`
		} `json:"choices,omitempty"`
	} `json:"output"`
	Usage struct {
		InputTokens  int `json:"input_tokens"`
		OutputTokens int `json:"output_tokens"`
		TotalTokens  int `json:"total_tokens"`
	} `json:"usage,omitempty"`
}

type DashScopeResponseMessage struct {
	Role             string          `json:"role"`
	Content          json.RawMessage `json:"content"`
	ReasoningContent string          `json:"reasoning_content,omitempty"`
}

// GrammarCheckResponse 语用失误检查响应结构 (LLM 返回的 JSON)
type GrammarCheckResponse struct {
	HasError    bool   `json:"has_error"`
	Suggestion  string `json:"suggestion"`
	Explanation string `json:"explanation"`
	ErrorType   string `json:"error_type"`
	// 新增字段: LLM 返回的详细分析
	Impoliteness               bool   `json:"impoliteness"`
	LinguisticPragmaticFailure bool   `json:"linguistic_pragmatic_failure"`
	SocialPragmaticFailure     bool   `json:"social_pragmatic_failure"`
	OverallEvaluation          string `json:"overall_evaluation"` // "good" / "improvable" / "problematic"
}

// mapErrorType 根据 LLM 返回的详细分析结果映射到5种错误类型
func mapErrorType(result *GrammarCheckResponse) string {
	evaluation := strings.ToLower(strings.TrimSpace(result.OverallEvaluation))
	hasPragma := result.LinguisticPragmaticFailure
	hasSocio := result.SocialPragmaticFailure

	switch evaluation {
	case "problematic":
		// 严重情况
		if hasPragma && hasSocio {
			return ErrorTypeBothFailure // 语言语用失误和社会语用失误
		} else if hasSocio {
			return ErrorTypeSevereSociopragmatic // 严重社会语用失误
		} else {
			return ErrorTypeSeverePragmalinguistic // 严重语言语用失误
		}
	case "improvable":
		// 可改进情况
		if hasSocio {
			return ErrorTypeSociopragmatic // 社会语用失误
		} else {
			return ErrorTypePragmalinguistic // 语用语言失误
		}
	default:
		// 兜底: 根据失误类型判断
		if hasPragma && hasSocio {
			return ErrorTypeBothFailure
		} else if hasSocio {
			return ErrorTypeSociopragmatic
		} else {
			return ErrorTypePragmalinguistic
		}
	}
}

// buildCombinedPrompt 构建 PFChat 语用检测提示词。
func buildCombinedPrompt(history []Message, current Message, sender User, receiver User, session ConversationSession) string {
	var sb strings.Builder

	msgLang := detectMessageLanguage(current.Content)
	isChineseMsg := msgLang == "zh"
	targetLang := getLanguageFullName(session.TargetLanguage)
	suggestionLang := getSuggestionLanguageName(msgLang, session.TargetLanguage, isChineseMsg)
	analysisTarget := getAnalysisTargetName(current.Role, isChineseMsg)
	explanationCountry := sender.Country
	if current.Role == ErrorSourceLLM {
		explanationCountry = receiver.Country
	}
	explanationLang := getCountryLanguageName(explanationCountry, isChineseMsg)

	if isChineseMsg {
		sb.WriteString("你是 PFChat 项目中的跨文化语用学评估器。\n")
		sb.WriteString("PFChat 是一个语言学习聊天练习系统：用户与 LLM Bot 围绕指定关系和主题对话，系统只记录会影响交际效果的语用问题。\n")
		sb.WriteString(fmt.Sprintf("你的任务是判断“%s”在本次会话上下文中是否存在语用失误，并给出可直接展示在语用问题记录页的 JSON。\n\n", analysisTarget))

		sb.WriteString("会话上下文:\n")
		sb.WriteString(fmt.Sprintf("- 会话模式: %s\n", getSessionModePrompt(session.Mode, true)))
		sb.WriteString(fmt.Sprintf("- 目标/练习语言: %s\n", targetLang))
		sb.WriteString(fmt.Sprintf("- 对话关系: %s\n", session.RelationshipType))
		sb.WriteString(fmt.Sprintf("- 对话主题: %s\n", session.Topic))
		sb.WriteString(fmt.Sprintf("- 发送者国家/地区: %s\n", getCountryName(sender.Country)))
		sb.WriteString(fmt.Sprintf("- 当前分析对象: %s\n", analysisTarget))
		sb.WriteString(fmt.Sprintf("- 接收者语言背景: %s\n\n", getCountryName(receiver.Country)))

		sb.WriteString("判定标准:\n")
		sb.WriteString("- 只检测语用问题，不做普通语法、拼写、词汇或风格润色；只有这些问题改变礼貌、意图或关系处理时才标记。\n")
		sb.WriteString("- 根据对话关系判断得体性：陌生人、师生、同事/商务关系通常需要更高礼貌度；朋友、同学关系可更自然直接。\n")
		sb.WriteString("- 根据主题判断场景期待：学术、商务、旅行、文化交流和日常闲聊的表达规范不同。\n")
		sb.WriteString("- 不要因为学习者表达不够地道就标记错误；只有可能造成冒犯、误解、请求/拒绝/感谢/道歉不当或关系失衡时才标记。\n")
		sb.WriteString("- 只评价当前分析对象，不把历史消息中的问题归因到当前消息。\n")
		sb.WriteString("- llm_l2 模式中，LLM Bot 会适当模拟二语学习者的语用问题；如果当前分析对象是 LLM Bot 回复，请识别其故意触发的语用失误。\n")
		sb.WriteString("- user_l2 模式中，重点评估用户用目标语言与母语者交流时的语用得体性。\n\n")

		sb.WriteString("字段含义:\n")
		sb.WriteString("- impoliteness: 当前消息是否明显不礼貌、冒犯、命令感过强或缺少必要缓和。\n")
		sb.WriteString("- linguistic_pragmatic_failure: 语言形式选择导致语用功能不当，例如请求、拒绝、道歉、感谢、称呼、缓和语或礼貌策略不合适。\n")
		sb.WriteString("- social_pragmatic_failure: 对社会关系、身份距离、权力差异、文化规范或场景期待判断不当。\n")
		sb.WriteString("- overall_evaluation: good 表示无明显问题；improvable 表示轻中度不合适但通常可修正；problematic 表示很可能冒犯或导致交际失败。\n\n")
	} else {
		sb.WriteString("You are the cross-cultural pragmatics evaluator inside PFChat.\n")
		sb.WriteString("PFChat is a language-learning chat practice system where a user and an LLM bot talk within a selected relationship and topic. The app only records pragmatic issues that affect communicative success.\n")
		sb.WriteString(fmt.Sprintf("Your task is to judge whether the %s has a pragmatic failure in this session context, then return JSON that can be shown directly in the pragmatic issue record page.\n\n", analysisTarget))

		sb.WriteString("Session context:\n")
		sb.WriteString(fmt.Sprintf("- Session mode: %s\n", getSessionModePrompt(session.Mode, false)))
		sb.WriteString(fmt.Sprintf("- Target/practice language: %s\n", targetLang))
		sb.WriteString(fmt.Sprintf("- Relationship: %s\n", session.RelationshipType))
		sb.WriteString(fmt.Sprintf("- Topic: %s\n", session.Topic))
		sb.WriteString(fmt.Sprintf("- Sender country/region: %s\n", getCountryName(sender.Country)))
		sb.WriteString(fmt.Sprintf("- Current analysis target: %s\n", analysisTarget))
		sb.WriteString(fmt.Sprintf("- Receiver language background: %s\n\n", getCountryName(receiver.Country)))

		sb.WriteString("Evaluation rules:\n")
		sb.WriteString("- Detect pragmatic failures only, not ordinary grammar, spelling, vocabulary, or style issues unless they change politeness, intent, or relationship management.\n")
		sb.WriteString("- Judge appropriateness by relationship: strangers, teacher-student, colleague/business contexts usually require more politeness; friends and classmates can be more direct.\n")
		sb.WriteString("- Judge by topic: academic, business, travel, cultural exchange, and daily chat contexts have different expectations.\n")
		sb.WriteString("- Do not flag a message merely because it is non-native or not idiomatic; flag only likely offense, misunderstanding, inappropriate requests/refusals/thanks/apologies, or relationship mismatch.\n")
		sb.WriteString("- Evaluate only the current analysis target; do not attribute issues in previous messages to the current message.\n")
		sb.WriteString("- In llm_l2 mode, the LLM bot may appropriately simulate L2 pragmatic problems; if the current analysis target is an LLM Bot reply, identify the intentionally triggered pragmatic failure.\n")
		sb.WriteString("- In user_l2 mode, focus on whether the user's target-language message is pragmatically appropriate for a native-speaker interlocutor.\n\n")

		sb.WriteString("Field meanings:\n")
		sb.WriteString("- impoliteness: whether the current message is clearly rude, offensive, too commanding, or lacks necessary mitigation.\n")
		sb.WriteString("- linguistic_pragmatic_failure: whether the wording/form choice makes the speech act pragmatically inappropriate, such as requests, refusals, apologies, thanks, address terms, mitigation, or politeness strategies.\n")
		sb.WriteString("- social_pragmatic_failure: whether the message misjudges social relationship, distance, power, cultural norms, or situational expectations.\n")
		sb.WriteString("- overall_evaluation: good means no clear issue; improvable means mild/moderate inappropriateness; problematic means likely offense or communication breakdown.\n\n")
	}

	if len(history) > 0 {
		if isChineseMsg {
			sb.WriteString("最近聊天记录（从旧到新）:\n")
		} else {
			sb.WriteString("Recent chat history (oldest first):\n")
		}
		for i := len(history) - 1; i >= 0; i-- {
			msg := history[i]
			var senderName string
			if msg.Role == "user" || msg.Sender.ID == sender.ID {
				if isChineseMsg {
					senderName = "用户"
				} else {
					senderName = "User"
				}
			} else {
				senderName = "LLM Bot"
			}
			sb.WriteString(fmt.Sprintf("[%s]: %s\n", senderName, msg.Content))
		}
		sb.WriteString("\n")
	}

	if isChineseMsg {
		sb.WriteString(fmt.Sprintf("当前待分析的%s: %q\n\n", analysisTarget, current.Content))
	} else {
		sb.WriteString(fmt.Sprintf("Current %s to analyze: %q\n\n", analysisTarget, current.Content))
	}

	if isChineseMsg {
		sb.WriteString("请以以下 JSON 格式分析（不要包含 markdown 代码块，只输出纯 JSON）:\n")
	} else {
		sb.WriteString("Analyze and respond in the following JSON format (no markdown code blocks, pure JSON only):\n")
	}

	sb.WriteString("{\n")
	sb.WriteString("  \"has_error\": true/false,\n")
	sb.WriteString("  \"impoliteness\": true/false,\n")
	sb.WriteString("  \"linguistic_pragmatic_failure\": true/false,\n")
	sb.WriteString("  \"social_pragmatic_failure\": true/false,\n")
	sb.WriteString("  \"overall_evaluation\": \"good\"/\"improvable\"/\"problematic\",\n")
	sb.WriteString("  \"error_type\": \"语用语言失误/社会语用失误/严重语用语言失误/严重社会语用失误/语用语言失误和社会语用失误\",\n")
	sb.WriteString("  \"suggestion\": \"single revised sentence in the SAME language as the original message\",\n")
	sb.WriteString("  \"explanation\": \"brief reason and improvement advice\"\n")
	sb.WriteString("}\n\n")

	if isChineseMsg {
		sb.WriteString("如果没有语用失误，返回:\n")
		sb.WriteString("{\"has_error\": false, \"impoliteness\": false, \"linguistic_pragmatic_failure\": false, \"social_pragmatic_failure\": false, \"overall_evaluation\": \"good\", \"error_type\": \"\", \"suggestion\": \"\", \"explanation\": \"\"}\n\n")
		sb.WriteString(fmt.Sprintf("输出语言要求: suggestion 必须使用%s，并且只包含一个最佳修改句；explanation 必须使用%s，说明为什么它是语用问题以及如何改进。\n", suggestionLang, explanationLang))
		sb.WriteString("如果 has_error 为 false，suggestion 和 explanation 必须为空字符串。\n")
	} else {
		sb.WriteString("If no pragmatic failure is detected, return:\n")
		sb.WriteString("{\"has_error\": false, \"impoliteness\": false, \"linguistic_pragmatic_failure\": false, \"social_pragmatic_failure\": false, \"overall_evaluation\": \"good\", \"error_type\": \"\", \"suggestion\": \"\", \"explanation\": \"\"}\n\n")
		sb.WriteString(fmt.Sprintf("Output language requirements: suggestion must be in %s and contain only one best revised sentence; explanation must be in %s and explain why it is a pragmatic issue and how to improve it.\n", suggestionLang, explanationLang))
		sb.WriteString("If has_error is false, suggestion and explanation must be empty strings.\n")
	}

	return sb.String()
}

func getAnalysisTargetName(role string, isChinese bool) string {
	if role == ErrorSourceLLM {
		if isChinese {
			return "LLM Bot 当前回复"
		}
		return "LLM Bot reply"
	}
	if isChinese {
		return "用户当前消息"
	}
	return "user message"
}
func getSessionModePrompt(mode string, isChinese bool) string {
	switch mode {
	case ModeUserL2:
		if isChinese {
			return "user_l2，用户使用目标语言练习，LLM Bot 作为目标语言母语者自然回应"
		}
		return "user_l2, the user practices in the target language and the LLM bot replies as a native speaker"
	case ModeLLML2:
		if isChinese {
			return "llm_l2，LLM Bot 使用目标语言模拟二语学习者，用户通常作为母语者参与对话"
		}
		return "llm_l2, the LLM bot simulates an L2 learner in the target language and the user usually participates as a native speaker"
	default:
		if isChinese {
			return "未知模式，按普通跨文化聊天练习处理"
		}
		return "unknown mode, treat as a regular cross-cultural chat practice session"
	}
}

func getSuggestionLanguageName(msgLang string, targetLang string, isChinese bool) string {
	switch msgLang {
	case "zh":
		if isChinese {
			return "中文"
		}
		return "Chinese"
	case "ja":
		if isChinese {
			return "日语"
		}
		return "Japanese"
	case "ko":
		if isChinese {
			return "韩语"
		}
		return "Korean"
	default:
		return getTargetLanguagePromptName(targetLang, isChinese)
	}
}

func getTargetLanguagePromptName(lang string, isChinese bool) string {
	switch strings.ToUpper(lang) {
	case "ZH":
		if isChinese {
			return "中文"
		}
		return "Chinese"
	case "JP":
		if isChinese {
			return "日语"
		}
		return "Japanese"
	case "KR":
		if isChinese {
			return "韩语"
		}
		return "Korean"
	case "FR":
		if isChinese {
			return "法语"
		}
		return "French"
	case "DE":
		if isChinese {
			return "德语"
		}
		return "German"
	default:
		if isChinese {
			return "英语"
		}
		return "English"
	}
}

func getCountryLanguageName(country string, isChinese bool) string {
	switch country {
	case "CN", "TW", "HK", "SG":
		if isChinese {
			return "中文"
		}
		return "Chinese"
	case "JP":
		if isChinese {
			return "日语"
		}
		return "Japanese"
	case "KR":
		if isChinese {
			return "韩语"
		}
		return "Korean"
	case "FR":
		if isChinese {
			return "法语"
		}
		return "French"
	case "DE":
		if isChinese {
			return "德语"
		}
		return "German"
	default:
		if isChinese {
			return "英语"
		}
		return "English"
	}
}

func getDashScopeAPIKey() string {
	if apiKey := strings.TrimSpace(os.Getenv("DASHSCOPE_API_KEY")); apiKey != "" {
		return apiKey
	}
	return fallbackDashScopeAPIKey
}

func getDashScopeModel() string {
	if model := strings.TrimSpace(os.Getenv("DASHSCOPE_MODEL")); model != "" {
		return model
	}
	return defaultModel
}

func getDashScopeGenerationURL() string {
	if apiURL := strings.TrimSpace(os.Getenv("DASHSCOPE_API_URL")); apiURL != "" {
		return strings.TrimRight(apiURL, "/")
	}

	baseURL := strings.TrimSpace(os.Getenv("DASHSCOPE_BASE_URL"))
	if baseURL == "" {
		return dashScopeDefaultAPIURL
	}
	baseURL = strings.TrimRight(baseURL, "/")
	if strings.HasSuffix(baseURL, "/generation") {
		return baseURL
	}
	if strings.HasSuffix(baseURL, "/api/v1") {
		return baseURL + dashScopeGenerationPath
	}
	return baseURL + "/api/v1" + dashScopeGenerationPath
}

func boolPtr(v bool) *bool {
	return &v
}

func newDashScopeTextMessage(role string, content string) DashScopeMessage {
	return DashScopeMessage{
		Role: role,
		Content: []DashScopeContentPart{
			{Text: content},
		},
	}
}

// makeDashScopeRequest 发起 DashScope 原生 HTTP 请求 (可复用)
func makeDashScopeRequest(apiKey string, reqBody DashScopeRequest) (*DashScopeResponse, error) {
	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("JSON 序列化失败: %v", err)
	}

	req, err := http.NewRequest("POST", getDashScopeGenerationURL(), bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("API 请求失败: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API 返回错误 %d: %s", resp.StatusCode, string(body))
	}

	var dashResp DashScopeResponse
	if err := json.Unmarshal(body, &dashResp); err != nil {
		return nil, fmt.Errorf("解析响应失败: %v", err)
	}
	if dashResp.StatusCode != 0 && dashResp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API 返回错误 %d: %s %s", dashResp.StatusCode, dashResp.Code, dashResp.Message)
	}

	return &dashResp, nil
}

func getDashScopeResponseText(resp *DashScopeResponse) string {
	if resp == nil {
		return ""
	}
	if len(resp.Output.Choices) > 0 {
		text := extractDashScopeContentText(resp.Output.Choices[0].Message.Content)
		if strings.TrimSpace(text) != "" {
			return text
		}
	}
	return resp.Output.Text
}

func extractDashScopeContentText(raw json.RawMessage) string {
	raw = bytes.TrimSpace(raw)
	if len(raw) == 0 || bytes.Equal(raw, []byte("null")) {
		return ""
	}

	var text string
	if err := json.Unmarshal(raw, &text); err == nil {
		return text
	}

	var parts []map[string]any
	if err := json.Unmarshal(raw, &parts); err == nil {
		var sb strings.Builder
		for _, part := range parts {
			if value, ok := part["text"].(string); ok {
				sb.WriteString(value)
			}
		}
		return sb.String()
	}

	var part map[string]any
	if err := json.Unmarshal(raw, &part); err == nil {
		if value, ok := part["text"].(string); ok {
			return value
		}
	}

	return string(raw)
}

// callDashScopeAPI 调用 DashScope API 进行语用检查
func callDashScopeAPI(prompt string) (*GrammarCheckResponse, error) {
	reqBody := DashScopeRequest{
		Model: getDashScopeModel(),
		Input: DashScopeInput{
			Messages: []DashScopeMessage{
				newDashScopeTextMessage("user", prompt),
			},
		},
		Parameters: DashScopeParameters{
			ResultFormat:        "message",
			Temperature:         0.3,
			MaxCompletionTokens: 512,
			ResponseFormat:      &DashScopeResponseFormat{Type: "json_object"},
			EnableThinking:      boolPtr(false),
		},
	}

	dashResp, err := makeDashScopeRequest(getDashScopeAPIKey(), reqBody)
	if err != nil {
		return nil, err
	}

	// 提取响应文本
	responseText := getDashScopeResponseText(dashResp)
	if strings.TrimSpace(responseText) == "" {
		return &GrammarCheckResponse{HasError: false}, nil
	}

	// 提取 JSON
	jsonStr := extractJSON(responseText)
	log.Printf("LLM Response JSON: %s", jsonStr)

	var result GrammarCheckResponse
	if err := json.Unmarshal([]byte(jsonStr), &result); err != nil {
		log.Printf("JSON parse error: %v, raw: %s", err, jsonStr)
		return &GrammarCheckResponse{HasError: false}, nil
	}

	// 如果 LLM 返回了新格式但 has_error 未正确设置, 根据 overall_evaluation 修正
	if !result.HasError && result.OverallEvaluation != "" && result.OverallEvaluation != "good" {
		result.HasError = true
	}

	// 如果 has_error 为 true 但没有新格式字段, 使用旧版兼容逻辑
	if result.HasError && result.OverallEvaluation == "" {
		// 兼容旧格式: 根据 error_type 字段推断
		if result.ErrorType == "社会语用失误" {
			result.SocialPragmaticFailure = true
			result.OverallEvaluation = "improvable"
		} else if result.ErrorType == "语言语用失误" {
			result.LinguisticPragmaticFailure = true
			result.OverallEvaluation = "improvable"
		} else {
			result.LinguisticPragmaticFailure = true
			result.OverallEvaluation = "improvable"
		}
	}

	// 映射到项目错误类型
	if result.HasError {
		result.ErrorType = mapErrorType(&result)
	}

	return &result, nil
}
func extractJSON(text string) string {
	text = strings.ReplaceAll(text, "```json", "")
	text = strings.ReplaceAll(text, "```", "")
	text = strings.TrimSpace(text)

	start := strings.Index(text, "{")
	end := strings.LastIndex(text, "}")

	if start == -1 || end == -1 || start > end {
		return text
	}

	return text[start : end+1]
}

func getCountryName(countryCode string) string {
	countryMap := map[string]string{
		"CN":    "中国(China)",
		"US":    "美国(USA)",
		"GB":    "英国(UK)",
		"JP":    "日本(Japan)",
		"KR":    "韩国(Korea)",
		"FR":    "法国(France)",
		"DE":    "德国(Germany)",
		"CA":    "加拿大(Canada)",
		"AU":    "澳大利亚(Australia)",
		"OTHER": "其他(Other)",
	}
	if name, ok := countryMap[countryCode]; ok {
		return name
	}
	return countryCode
}

// ============================================================================
// 【说明】buildCombinedPrompt 基于消息实际语言和会话上下文构造提示词。
//   - suggestion 字段：与消息内容同语言。
//   - explanation 字段：使用发送者的主要语言，方便学习者理解。
//   - 判定标准：贴合 PFChat 的会话模式、目标语言、关系、主题和五类语用失误。
// ============================================================================

// detectMessageLanguage 检测消息的主要语言。
// 返回 "zh"（中文）、"ja"（日文）、"ko"（韩文）或 "en"（英文及其他）。
func detectMessageLanguage(content string) string {
	chineseCount := 0
	japaneseCount := 0
	koreanCount := 0
	totalLetters := 0

	for _, r := range content {
		if (r >= 0x4E00 && r <= 0x9FFF) || (r >= 0x3400 && r <= 0x4DBF) {
			// CJK 汉字（中文为主，日文汉字也在此范围，但平假名/片假名更有区分力）
			chineseCount++
			totalLetters++
		} else if (r >= 0x3040 && r <= 0x309F) || (r >= 0x30A0 && r <= 0x30FF) {
			// 平假名或片假名 → 日文特征
			japaneseCount++
			totalLetters++
		} else if r >= 0xAC00 && r <= 0xD7AF {
			// 韩文音节
			koreanCount++
			totalLetters++
		} else if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') {
			totalLetters++
		}
	}

	if totalLetters == 0 {
		return "en"
	}

	// 日文优先判断（因为日文也含汉字，但有假名）
	if japaneseCount > 0 {
		return "ja"
	}
	if koreanCount > 0 {
		return "ko"
	}
	threshold := totalLetters / 4 // 超过1/4汉字视为中文
	if chineseCount > threshold {
		return "zh"
	}
	return "en"
}
