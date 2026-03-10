package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"
)

// DashScope API 配置
const (
	dashScopeAPIURL = "https://dashscope.aliyuncs.com/api/v1/services/aigc/text-generation/generation"
	defaultModel    = "qwen-plus"
)

// DashScope 请求结构
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
	Content string `json:"content"`
}

type DashScopeParameters struct {
	ResultFormat string  `json:"result_format,omitempty"`
	Temperature  float64 `json:"temperature,omitempty"`
	MaxTokens    int     `json:"max_tokens,omitempty"`
}

// DashScope 响应结构
type DashScopeResponse struct {
	Output struct {
		Text    string `json:"text,omitempty"`
		Finish  string `json:"finish_reason,omitempty"`
		Choices []struct {
			Message struct {
				Role    string `json:"role"`
				Content string `json:"content"`
			} `json:"message"`
			FinishReason string `json:"finish_reason"`
		} `json:"choices,omitempty"`
	} `json:"output"`
	Usage struct {
		InputTokens  int `json:"input_tokens"`
		OutputTokens int `json:"output_tokens"`
		TotalTokens  int `json:"total_tokens"`
	} `json:"usage"`
	RequestID string `json:"request_id"`
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

// 执行语用失误检查 (异步, 用于消息发送后)
func checkGrammar(userID uint, message Message) {
	var sender, receiver User
	db.First(&sender, message.SenderID)
	db.First(&receiver, message.ReceiverID)

	var historyMessages []Message
	db.Where(
		"((sender_id = ? AND receiver_id = ?) OR (sender_id = ? AND receiver_id = ?)) AND id < ?",
		message.SenderID, message.ReceiverID, message.ReceiverID, message.SenderID, message.ID,
	).Order("created_at DESC").Limit(10).Preload("Sender").Preload("Receiver").Find(&historyMessages)

	prompt := buildCombinedPrompt(historyMessages, message, sender, receiver)

	result, err := callDashScopeAPI(prompt)
	if err != nil {
		log.Printf("DashScope API error: %v", err)
		return
	}

	if result.HasError {
		errorType := mapErrorType(result)

		grammarError := GrammarError{
			UserID:         userID,
			MessageID:      message.ID,
			OriginalText:   message.Content,
			LLMSuggestion:  result.Suggestion,
			LLMExplanation: result.Explanation,
			ErrorType:      errorType,
		}

		if err := db.Create(&grammarError).Error; err != nil {
			log.Printf("Save pragmatic error failed: %v", err)
			return
		}

		// 通知发送方
		notifyUser(userID, grammarError)

		// 【Fix Issue 4】为接收方生成本地化解释，再通知接收方
		receiverExplanation := generateReceiverExplanation(
			message.Content,
			result.Explanation,
			errorType,
			sender,
			receiver,
		)
		notifyReceiver(message.ReceiverID, message.ID, grammarError, receiverExplanation)
	}
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

// buildCombinedPrompt 构建合并的提示词 (新版本 - 基于用户提供的语用学分析prompt)
func buildCombinedPrompt(history []Message, current Message, sender User, receiver User) string {
	var sb strings.Builder

	// 【Fix】基于消息实际语言判断，而非发送者国籍
	// 外国人可能用中文写消息，此时建议也应该用中文；
	// explanation 则用接收者的母语（或发送者的母语）帮助理解
	msgLang := detectMessageLanguage(current.Content)
	isChineseMsg := msgLang == "zh"

	// prompt 框架语言也跟随消息语言（方便 LLM 理解上下文）
	if isChineseMsg {
		// 中文 prompt
		sb.WriteString("你是一位精通语用学和跨文化交际的语言学专家。\n")
		sb.WriteString("你的任务是分析在线聊天中平等关系交际者（如同学或陌生人）之间的话语。\n\n")

		sb.WriteString("重要约束:\n")
		sb.WriteString("- 只检测真正的语用失误，不纠正语法错误或风格问题\n")
		sb.WriteString("- 只有当话语可能导致交际失败或冒犯时才标记错误\n")
		sb.WriteString("- 平等关系中的直接表达通常是可以接受的\n")
		sb.WriteString("- suggestion 字段只包含修改后的句子，不包含任何解释或前缀\n")
		sb.WriteString("- 不要包含任何替代选项如 \"或更自然的...\"\n")
		sb.WriteString("- 不要在 suggestion 中包含任何解释——把解释放在 explanation 字段\n")
		sb.WriteString("- 只输出最佳的单一修改句子，别的什么都不要\n\n")
	} else {
		// 非中文消息用英文 prompt
		sb.WriteString("You are a linguistics expert specializing in pragmatics and cross-cultural communication.\n")
		sb.WriteString("Your task is to analyze utterances between equal-status interlocutors (e.g., classmates or strangers) in online chat.\n\n")

		sb.WriteString("Important constraints:\n")
		sb.WriteString("- Only detect genuine pragmatic failures, not grammatical errors or style issues\n")
		sb.WriteString("- Only flag errors when an utterance may cause communication breakdown or offense\n")
		sb.WriteString("- Direct expression in equal-status relationships is usually acceptable\n")
		sb.WriteString("- The suggestion field should only contain the corrected sentence, no explanation or prefix\n")
		sb.WriteString("- Do NOT include alternative options like \"or alternatively...\"\n")
		sb.WriteString("- Do NOT include any explanation in the suggestion field\n")
		sb.WriteString("- Just output the single best corrected sentence, nothing else\n\n")
	}

	// 聊天历史
	if len(history) > 0 {
		if isChineseMsg {
			sb.WriteString("最近聊天记录（从旧到新）:\n")
		} else {
			sb.WriteString("Recent chat history (oldest first):\n")
		}
		for i := len(history) - 1; i >= 0; i-- {
			msg := history[i]
			var senderName string
			if msg.Sender.ID == sender.ID {
				senderName = "Sender"
			} else {
				senderName = "Receiver"
			}
			sb.WriteString(fmt.Sprintf("[%s]: %s\n", senderName, msg.Content))
		}
		sb.WriteString("\n")
	}

	// 当前待分析消息
	if isChineseMsg {
		sb.WriteString(fmt.Sprintf("发送者来自: %s\n", getCountryName(sender.Country)))
		sb.WriteString(fmt.Sprintf("接收者来自: %s\n", getCountryName(receiver.Country)))
		sb.WriteString(fmt.Sprintf("待分析消息: \"%s\"\n\n", current.Content))
	} else {
		sb.WriteString(fmt.Sprintf("Sender's country: %s\n", getCountryName(sender.Country)))
		sb.WriteString(fmt.Sprintf("Receiver's country: %s\n", getCountryName(receiver.Country)))
		sb.WriteString(fmt.Sprintf("Message to analyze: \"%s\"\n\n", current.Content))
	}

	// JSON 格式要求
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
	sb.WriteString("  \"suggestion\": \"corrected sentence in the SAME language as the original message\",\n")
	sb.WriteString("  \"explanation\": \"detailed explanation\"\n")
	sb.WriteString("}\n\n")

	if isChineseMsg {
		sb.WriteString("如果没有语用失误，返回:\n")
		sb.WriteString("{\"has_error\": false, \"impoliteness\": false, \"linguistic_pragmatic_failure\": false, \"social_pragmatic_failure\": false, \"overall_evaluation\": \"good\", \"suggestion\": \"\", \"explanation\": \"\"}\n\n")
		// 【Fix】suggestion 与原消息同语言（中文），explanation 用接收者母语
		var suggestionLangNote string
		switch msgLang {
		case "zh":
			suggestionLangNote = "suggestion 字段必须用中文（与原消息语言相同）。"
		case "ja":
			suggestionLangNote = "The suggestion field must be in Japanese / 日本語（same language as the original message）."
		case "ko":
			suggestionLangNote = "The suggestion field must be in Korean / 한국어（same language as the original message）."
		default: // "en" 及其他
			suggestionLangNote = "The suggestion field must be in English (same language as the original message)."
		}

		var explanationLangNote string
		switch sender.Country {
		case "CN", "TW", "HK", "SG":
			explanationLangNote = "explanation 字段必须用中文（发送者的母语）。"
		case "JP":
			explanationLangNote = "The explanation field must be in Japanese / 日本語（sender's native language）."
		case "KR":
			explanationLangNote = "The explanation field must be in Korean / 한국어（sender's native language）."
		default:
			explanationLangNote = fmt.Sprintf("The explanation field must be in %s (sender's native language).", sender.Country)
		}

		if msgLang == "zh" {
			sb.WriteString(fmt.Sprintf("重要提醒: %s %s\n", suggestionLangNote, explanationLangNote))
		} else {
			sb.WriteString(fmt.Sprintf("IMPORTANT: %s %s\n", suggestionLangNote, explanationLangNote))
		}
	} else {
		sb.WriteString("If no pragmatic failure is detected, return:\n")
		sb.WriteString("{\"has_error\": false, \"impoliteness\": false, \"linguistic_pragmatic_failure\": false, \"social_pragmatic_failure\": false, \"overall_evaluation\": \"good\", \"suggestion\": \"\", \"explanation\": \"\"}\n\n")
		// 【Fix】suggestion 与原消息同语言
		var suggestionLangNote string
		switch msgLang {
		case "zh":
			suggestionLangNote = "suggestion 字段必须用中文（与原消息语言相同）。"
		case "ja":
			suggestionLangNote = "The suggestion field must be in Japanese / 日本語（same language as the original message）."
		case "ko":
			suggestionLangNote = "The suggestion field must be in Korean / 한국어（same language as the original message）."
		default: // "en" 及其他
			suggestionLangNote = "The suggestion field must be in English (same language as the original message)."
		}

		var explanationLangNote string
		switch sender.Country {
		case "CN", "TW", "HK", "SG":
			explanationLangNote = "explanation 字段必须用中文（发送者的母语）。"
		case "JP":
			explanationLangNote = "The explanation field must be in Japanese / 日本語（sender's native language）."
		case "KR":
			explanationLangNote = "The explanation field must be in Korean / 한국어（sender's native language）."
		default:
			explanationLangNote = fmt.Sprintf("The explanation field must be in %s (sender's native language).", sender.Country)
		}

		if msgLang == "zh" {
			sb.WriteString(fmt.Sprintf("重要提醒: %s %s\n", suggestionLangNote, explanationLangNote))
		} else {
			sb.WriteString(fmt.Sprintf("IMPORTANT: %s %s\n", suggestionLangNote, explanationLangNote))
		}
	}

	return sb.String()
}

// makeDashScopeRequest 发起 DashScope API 请求 (可复用)
func makeDashScopeRequest(apiKey string, reqBody DashScopeRequest) (*DashScopeResponse, error) {
	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("JSON 序列化失败: %v", err)
	}

	req, err := http.NewRequest("POST", dashScopeAPIURL, bytes.NewBuffer(jsonBody))
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

	return &dashResp, nil
}

// buildReceiverExplanationPrompt 为接收方构建本地化解释提示词
func buildReceiverExplanationPrompt(originalText, senderExplanation, errorType string, sender User, receiver User) string {
	var sb strings.Builder

	isChineseReceiver := receiver.Country == "CN"

	if isChineseReceiver {
		sb.WriteString("你是一位跨文化交际专家。请根据以下信息，用中文生成1-2句简短的提示，")
		sb.WriteString("向接收者说明发送者的消息存在语用问题，帮助接收者理解这可能是语言或文化差异导致的。\n\n")
	} else {
		sb.WriteString("You are an intercultural communication expert. Based on the information below, ")
		sb.WriteString("generate 1-2 brief sentences in English for the message receiver, ")
		sb.WriteString("explaining that the sender's message has a pragmatic issue ")
		sb.WriteString("that may be due to language or cultural differences.\n\n")
	}

	sb.WriteString(fmt.Sprintf("Sender's country: %s\n", getCountryName(sender.Country)))
	sb.WriteString(fmt.Sprintf("Receiver's country: %s\n", getCountryName(receiver.Country)))
	sb.WriteString(fmt.Sprintf("Original message: \"%s\"\n", originalText))
	sb.WriteString(fmt.Sprintf("Error type: %s\n", errorType))
	sb.WriteString(fmt.Sprintf("Technical explanation (for reference only, do NOT copy verbatim): %s\n\n", senderExplanation))

	if isChineseReceiver {
		sb.WriteString("用1-2句友善自然的中文说明语用问题及文化差异背景。不要重复原消息，不要使用技术性术语。只输出那1-2句话。")
		sb.WriteString("告诉对方这可能是因为语言或文化差异导致的，不要让对方觉得发送者故意冒犯或不礼貌。")
	} else {
		sb.WriteString("Write 1-2 friendly natural sentences explaining the pragmatic issue and cultural context. " +
			"Do NOT use technical linguistic terms. Output only those sentences.")
		sb.WriteString("Make sure to convey that this may be due to language or cultural differences, " +
			"and not to make the receiver feel that the sender is intentionally offensive or impolite.")
	}

	return sb.String()
}

// generateReceiverExplanation 为接收方生成本地化解释 (调用LLM翻译/适配)
func generateReceiverExplanation(originalText, senderExplanation, errorType string, sender User, receiver User) string {
	if senderExplanation == "" {
		return ""
	}

	apiKey := "sk-8ab77da79b894ba6beb61c9190c74602"

	prompt := buildReceiverExplanationPrompt(originalText, senderExplanation, errorType, sender, receiver)

	reqBody := DashScopeRequest{
		Model: "qwen-turbo",
		Input: DashScopeInput{
			Messages: []DashScopeMessage{
				{Role: "user", Content: prompt},
			},
		},
		Parameters: DashScopeParameters{
			Temperature: 0.5,
			MaxTokens:   200,
		},
	}

	dashResp, err := makeDashScopeRequest(apiKey, reqBody)
	if err != nil {
		log.Printf("generateReceiverExplanation error: %v", err)
		return senderExplanation
	}

	var responseText string
	if len(dashResp.Output.Choices) > 0 {
		responseText = strings.TrimSpace(dashResp.Output.Choices[0].Message.Content)
	} else if dashResp.Output.Text != "" {
		responseText = strings.TrimSpace(dashResp.Output.Text)
	}

	if responseText == "" {
		return senderExplanation
	}
	return responseText
}

// callDashScopeAPI 调用 DashScope API 进行语用检查
func callDashScopeAPI(prompt string) (*GrammarCheckResponse, error) {
	apiKey := "sk-8ab77da79b894ba6beb61c9190c74602"

	reqBody := DashScopeRequest{
		Model: defaultModel,
		Input: DashScopeInput{
			Messages: []DashScopeMessage{
				{
					Role:    "user",
					Content: prompt,
				},
			},
		},
		Parameters: DashScopeParameters{
			ResultFormat: "message",
			Temperature:  0.3,
			MaxTokens:    512,
		},
	}

	dashResp, err := makeDashScopeRequest(apiKey, reqBody)
	if err != nil {
		return nil, err
	}

	// 提取响应文本
	var responseText string
	if len(dashResp.Output.Choices) > 0 {
		responseText = dashResp.Output.Choices[0].Message.Content
	} else if dashResp.Output.Text != "" {
		responseText = dashResp.Output.Text
	} else {
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
// 【修改说明】修改 buildCombinedPrompt 函数中的语言判断逻辑
//
// 原逻辑：基于 sender.Country == "CN" 判断语言
// 新逻辑：检测消息内容的实际语言（中文/日文/韩文/英文等）
//   - suggestion 字段：与消息内容同语言（外国人说中文 → suggestion 用中文）
//   - explanation 字段：用接收者的母语（或发送者的母语），方便双方理解
// ============================================================================

// detectMessageLanguage 检测消息的主要语言
// 返回 "zh"（中文）、"ja"（日文）、"ko"（韩文）或 "en"（英文及其他）
// detectMessageLanguage 检测消息的主要语言
// 返回 "zh"（中文）、"ja"（日文）、"ko"（韩文）或 "en"（英文及其他）
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

// getExplanationLanguage 根据国家代码返回语言名称（供 prompt 中说明用）
func getExplanationLanguage(countryCode string) string {
	switch countryCode {
	case "CN", "TW", "HK", "SG":
		return "Chinese (中文)"
	case "JP":
		return "Japanese (日本語)"
	case "KR":
		return "Korean (한국어)"
	case "FR":
		return "French (Français)"
	case "DE":
		return "German (Deutsch)"
	default:
		return "English"
	}
}

// notifyUser 通知发送方 (通过 WebSocket)
func notifyUser(userID uint, grammarError GrammarError) {
	wsMsg := WSMessage{
		Type: "grammar_check",
		Data: GrammarCheckResult{
			HasError:    true,
			Suggestion:  grammarError.LLMSuggestion,
			Explanation: grammarError.LLMExplanation,
			MessageID:   grammarError.MessageID,
		},
		Timestamp: time.Now(),
	}

	msgBytes, err := json.Marshal(wsMsg)
	if err != nil {
		log.Printf("Marshal websocket message error: %v", err)
		return
	}

	if globalHub != nil {
		globalHub.sendToUser(userID, msgBytes)
	}
}

// notifyReceiver 通知接收方有语用错误 (不弹窗, 只刷新标识)
func notifyReceiver(receiverID uint, messageID uint, grammarError GrammarError, receiverExplanation string) {
	// 如果没有提供接收方解释，降级使用发送方的解释
	explanation := receiverExplanation
	if explanation == "" {
		explanation = grammarError.LLMExplanation
	}

	wsMsg := WSMessage{
		Type: "receiver_error_notify",
		Data: map[string]interface{}{
			"message_id":  messageID,
			"error_type":  grammarError.ErrorType,
			"suggestion":  grammarError.LLMSuggestion,
			"explanation": explanation, // 接收方本地化的解释
		},
		Timestamp: time.Now(),
	}

	msgBytes, err := json.Marshal(wsMsg)
	if err != nil {
		log.Printf("Marshal receiver notify error: %v", err)
		return
	}

	if globalHub != nil {
		globalHub.sendToUser(receiverID, msgBytes)
	}
}

var globalHub *Hub
