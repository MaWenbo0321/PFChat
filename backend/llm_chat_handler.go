package main

import (
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// SendLLMMessageRequest 发送给 LLM 的消息请求
type SendLLMMessageRequest struct {
	SessionID uint   `json:"session_id" binding:"required"`
	Content   string `json:"content" binding:"required"`
}

// SendLLMMessageResponse 发送给 LLM 的消息响应
type SendLLMMessageResponse struct {
	UserMessage       Message               `json:"user_message"`
	LLMResponse       Message               `json:"llm_response"`
	PragmaticCheck    *PragmaticCheckResult `json:"pragmatic_check"`
	LLMPragmaticCheck *PragmaticCheckResult `json:"llm_pragmatic_check,omitempty"`
	SessionEnded      bool                  `json:"session_ended"`
	SessionSummary    string                `json:"session_summary"`
}

// PragmaticCheckResult

type PragmaticCheckResult struct {
	HasError          bool   `json:"has_error"`
	ErrorType         string `json:"error_type"`
	Suggestion        string `json:"suggestion"`
	Explanation       string `json:"explanation"`
	OverallEvaluation string `json:"overall_evaluation"`
	ErrorRecordID     uint   `json:"error_record_id"`
}

// sendLLMMessage 用户发消息给LLM，获取回复
func sendLLMMessage(c *gin.Context) {
	userID := getCurrentUserID(c)

	var req SendLLMMessageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求参数"})
		return
	}

	// 获取会话信息
	var session ConversationSession
	if err := db.Where("id = ? AND user_id = ? AND is_active = ?", req.SessionID, userID, true).
		First(&session).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "活跃会话不存在"})
		return
	}

	// 获取用户和bot信息
	var user User
	if err := db.First(&user, userID).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取用户信息失败"})
		return
	}
	var botUser User
	if err := db.First(&botUser, session.BotUserID).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取 Bot 信息失败"})
		return
	}

	// 保存用户消息
	userMsg := Message{
		SenderID:   userID,
		ReceiverID: session.BotUserID,
		Content:    req.Content,
		SessionID:  session.ID,
		Role:       "user",
	}
	if err := db.Create(&userMsg).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "保存消息失败"})
		return
	}
	db.Preload("Sender").Preload("Receiver").First(&userMsg, userMsg.ID)

	// 获取历史消息（最近10条）
	var historyMessages []Message
	db.Where("session_id = ? AND id < ?", session.ID, userMsg.ID).
		Order("created_at DESC").Limit(10).
		Preload("Sender").Preload("Receiver").
		Find(&historyMessages)

	// 1. 对用户消息做语用检测
	pragmaticResult := checkUserMessagePragmatics(userID, userMsg, session, user, botUser, historyMessages)

	// 2. 生成LLM回复
	llmReplyContent := generateLLMChatReply(session, historyMessages, req.Content, user, botUser)

	// 保存LLM回复
	llmMsg := Message{
		SenderID:   session.BotUserID,
		ReceiverID: userID,
		Content:    llmReplyContent,
		SessionID:  session.ID,
		Role:       "llm",
	}
	if err := db.Create(&llmMsg).Error; err != nil {
		log.Printf("保存LLM消息失败: %v", err)
	}
	db.Preload("Sender").Preload("Receiver").First(&llmMsg, llmMsg.ID)

	var llmPragmaticResult *PragmaticCheckResult
	if session.Mode == ModeLLML2 {
		llmHistory := append([]Message{userMsg}, historyMessages...)
		llmPragmaticResult = checkLLMMessagePragmatics(userID, llmMsg, session, botUser, user, llmHistory)
	}

	// 3. 更新轮次计数
	session.RoundCount++
	db.Model(&session).Update("round_count", session.RoundCount)

	// 4. 判断是否需要自动结束（五轮模式）
	sessionEnded := false
	sessionSummary := ""

	if session.FeedbackMode == FeedbackRounds5 && session.RoundCount >= 5 {
		// 获取会话所有消息生成汇总
		var allMessages []Message
		db.Where("session_id = ?", session.ID).Order("created_at ASC").
			Preload("Sender").Preload("Receiver").Find(&allMessages)

		sessionSummary = generateSessionSummary(session, allMessages, user)

		// 自动结束会话
		db.Model(&session).Updates(map[string]interface{}{
			"is_active":        false,
			"summary_feedback": sessionSummary,
		})
		sessionEnded = true
	}

	c.JSON(http.StatusOK, SendLLMMessageResponse{
		UserMessage:       userMsg,
		LLMResponse:       llmMsg,
		PragmaticCheck:    pragmaticResult,
		LLMPragmaticCheck: llmPragmaticResult,
		SessionEnded:      sessionEnded,
		SessionSummary:    sessionSummary,
	})
}

// checkUserMessagePragmatics 对用户消息做语用检测并保存
func checkUserMessagePragmatics(userID uint, userMsg Message, session ConversationSession,
	user User, botUser User, history []Message) *PragmaticCheckResult {

	fakeReceiver := User{
		ID:      botUser.ID,
		Country: targetLanguageToCountry(session.TargetLanguage),
	}

	return checkMessagePragmatics(userID, userMsg, session, user, fakeReceiver, history, ErrorSourceUser)
}

// checkLLMMessagePragmatics 对 llm_l2 模式下的 Bot 回复做语用检测并保存
func checkLLMMessagePragmatics(userID uint, llmMsg Message, session ConversationSession,
	botUser User, user User, history []Message) *PragmaticCheckResult {

	return checkMessagePragmatics(userID, llmMsg, session, botUser, user, history, ErrorSourceLLM)
}

func checkMessagePragmatics(userID uint, msg Message, session ConversationSession,
	sender User, receiver User, history []Message, sourceRole string) *PragmaticCheckResult {

	prompt := buildCombinedPrompt(history, msg, sender, receiver, session)
	result, err := callDashScopeAPI(prompt)
	if err != nil {
		log.Printf("语用检测失败: %v", err)
		return &PragmaticCheckResult{HasError: false}
	}

	if !result.HasError {
		return &PragmaticCheckResult{HasError: false}
	}

	errorType := result.ErrorType
	if !IsValidErrorType(errorType) {
		errorType = ErrorTypePragmalinguistic
	}

	overallEvaluation := result.OverallEvaluation
	if overallEvaluation == "" {
		if IsProblematicErrorType(errorType) {
			overallEvaluation = "problematic"
		} else {
			overallEvaluation = "improvable"
		}
	}

	grammarError := GrammarError{
		UserID:            userID,
		SessionID:         session.ID,
		MessageID:         msg.ID,
		SourceRole:        sourceRole,
		OriginalText:      msg.Content,
		LLMSuggestion:     result.Suggestion,
		LLMExplanation:    result.Explanation,
		ErrorType:         errorType,
		OverallEvaluation: overallEvaluation,
	}
	var errorRecordID uint
	if err := db.Create(&grammarError).Error; err != nil {
		log.Printf("保存语用错误记录失败: %v", err)
	} else {
		errorRecordID = grammarError.ID
	}

	return &PragmaticCheckResult{
		HasError:          true,
		ErrorType:         errorType,
		Suggestion:        result.Suggestion,
		Explanation:       result.Explanation,
		OverallEvaluation: overallEvaluation,
		ErrorRecordID:     errorRecordID,
	}
}

// generateLLMChatReply 根据会话模式生成LLM回复
func generateLLMChatReply(session ConversationSession, history []Message, userInput string, user User, botUser User) string {
	var prompt string
	if session.Mode == ModeUserL2 {
		prompt = buildUserL2Prompt(session, history, userInput, user)
	} else {
		prompt = buildLLML2Prompt(session, history, userInput, user)
	}

	reply, err := callLLMChatAPI(prompt)
	if err != nil {
		log.Printf("生成LLM回复失败: %v", err)
		if session.Mode == ModeUserL2 {
			return "I'm sorry, could you please say that again?"
		}
		return "啊... 我不太明白你说什么。能再说一遍吗？"
	}
	return reply
}

// buildUserL2Prompt 模式1：用户使用第二语言，LLM作为母语者正常交流
func buildUserL2Prompt(session ConversationSession, history []Message, userInput string, user User) string {
	var sb strings.Builder

	targetLang := session.TargetLanguage
	relationship := session.RelationshipType
	topic := session.Topic
	userNativeLang := getLanguageNameByCountry(user.Country)
	targetLangFull := getLanguageFullName(targetLang)

	sb.WriteString(fmt.Sprintf("You are a native %s speaker having a natural conversation with a language learner.\n", targetLangFull))
	sb.WriteString(fmt.Sprintf("Your relationship with the user is: %s\n", relationship))
	sb.WriteString(fmt.Sprintf("Conversation topic: %s\n", topic))
	sb.WriteString(fmt.Sprintf("The user's native language is: %s\n\n", userNativeLang))

	sb.WriteString("Instructions:\n")
	sb.WriteString("- Respond naturally and fluently as a native speaker would\n")
	sb.WriteString(fmt.Sprintf("- Always respond in %s\n", targetLangFull))
	sb.WriteString("- Keep responses conversational and appropriate for the relationship type\n")
	sb.WriteString("- If the user makes language errors, do NOT explicitly correct them, just respond naturally\n")
	sb.WriteString("- Keep responses relatively short (2-4 sentences) to maintain natural conversation flow\n\n")

	if len(history) > 0 {
		sb.WriteString("Conversation history:\n")
		for i := len(history) - 1; i >= 0; i-- {
			msg := history[i]
			if msg.Role == "user" {
				sb.WriteString(fmt.Sprintf("User: %s\n", msg.Content))
			} else {
				sb.WriteString(fmt.Sprintf("You: %s\n", msg.Content))
			}
		}
		sb.WriteString("\n")
	}

	sb.WriteString(fmt.Sprintf("User's message: %s\n", userInput))
	sb.WriteString(fmt.Sprintf("Your response (in %s only):", targetLangFull))

	return sb.String()
}

// buildLLML2Prompt 模式2：LLM使用第二语言，模拟非流利说话者
func buildLLML2Prompt(session ConversationSession, history []Message, userInput string, user User) string {
	var sb strings.Builder

	targetLang := session.TargetLanguage
	relationship := session.RelationshipType
	topic := session.Topic
	userNativeLang := getLanguageNameByCountry(user.Country)
	targetLangFull := getLanguageFullName(targetLang)
	// LLM用目标语言回复，但故意不流利
	// 这里targetLang是LLM要"学习"的语言（也是用户的母语）

	sb.WriteString(fmt.Sprintf("You are a language learner whose native language is NOT %s. ", targetLangFull))
	sb.WriteString(fmt.Sprintf("You are learning %s as a second language and your level is intermediate.\n", targetLangFull))
	sb.WriteString(fmt.Sprintf("Your relationship with the user is: %s\n", relationship))
	sb.WriteString(fmt.Sprintf("Conversation topic: %s\n", topic))
	sb.WriteString(fmt.Sprintf("The user is a native %s speaker.\n\n", userNativeLang))

	sb.WriteString("Instructions - simulate an intermediate L2 speaker for pragmatic-awareness training:\n")
	sb.WriteString("- Keep the conversation natural and relevant to the user's message.\n")
	sb.WriteString("- When the context naturally involves a request, refusal, apology, thanks, disagreement, suggestion, invitation, or sensitive cultural/social expectation, include ONE subtle pragmatic failure that a learner might realistically make.\n")
	sb.WriteString("- Useful failure patterns: overly direct request, missing mitigation, wrong politeness level for the relationship, culturally unusual apology/thanks, awkward refusal, or literal transfer from another language.\n")
	sb.WriteString("- Do not force an error into every reply. If the user's message is simple or low-stakes, reply mostly naturally with only mild non-native phrasing.\n")
	sb.WriteString("- Aim for pragmatic failures in about 60-70% of replies, and make them subtle enough for the user to observe and reflect on.\n")
	sb.WriteString("- Do not explain or label your own mistakes in the chat reply.\n")
	sb.WriteString("- Keep responses conversational length (2-4 sentences).\n")
	sb.WriteString(fmt.Sprintf("- Respond ONLY in %s.\n\n", targetLangFull))

	if len(history) > 0 {
		sb.WriteString("Conversation history:\n")
		for i := len(history) - 1; i >= 0; i-- {
			msg := history[i]
			if msg.Role == "user" {
				sb.WriteString(fmt.Sprintf("Native speaker: %s\n", msg.Content))
			} else {
				sb.WriteString(fmt.Sprintf("You (learner): %s\n", msg.Content))
			}
		}
		sb.WriteString("\n")
	}

	sb.WriteString(fmt.Sprintf("Native speaker's message: %s\n", userInput))
	sb.WriteString(fmt.Sprintf("Your response as a %s learner (include an appropriate subtle pragmatic failure only when the context calls for it):", targetLangFull))

	return sb.String()
}

// generateSessionSummary 生成会话结束汇总反馈
func generateSessionSummary(session ConversationSession, messages []Message, user User) string {
	// 获取本次会话所有语用错误。session_id 是新字段，message_id 子查询兼容旧数据。
	var errors []GrammarError
	db.Where("user_id = ? AND (session_id = ? OR message_id IN (?))",
		user.ID,
		session.ID,
		db.Model(&Message{}).Select("id").Where("session_id = ?", session.ID),
	).Find(&errors)

	isZh := user.Country == "CN" || user.Country == "TW" || user.Country == "HK" || user.Country == "SG"
	targetLang := getLanguageFullName(session.TargetLanguage)

	var sb strings.Builder

	if isZh {
		sb.WriteString("你是一位跨文化语用学专家。请根据以下完整对话会话的语用分析，用中文给学习者提供详尽的会话总结报告。\n\n")
		sb.WriteString(fmt.Sprintf("练习语言: %s | 对话关系: %s | 对话主题: %s\n", targetLang, session.RelationshipType, session.Topic))
		sb.WriteString(fmt.Sprintf("总轮次: %d 轮\n\n", session.RoundCount))
	} else {
		sb.WriteString("You are an expert in cross-cultural pragmatics. Based on the complete conversation session analysis, provide a comprehensive session summary report in English.\n\n")
		sb.WriteString(fmt.Sprintf("Practice language: %s | Relationship: %s | Topic: %s\n", targetLang, session.RelationshipType, session.Topic))
		sb.WriteString(fmt.Sprintf("Total rounds: %d\n\n", session.RoundCount))
	}

	if len(errors) == 0 {
		if isZh {
			return fmt.Sprintf("恭喜！在本次%d轮的%s对话练习中，您没有产生明显的语用失误。您的语用能力表现出色，对话自然流畅，语气和表达方式都很恰当。继续保持，期待您下次的练习！", session.RoundCount, targetLang)
		}
		return fmt.Sprintf("Congratulations! In this %d-round %s conversation practice, no significant pragmatic errors were detected. Your pragmatic competence is excellent. Keep up the great work!", session.RoundCount, targetLang)
	}

	// 按错误类型和来源统计
	typeCount := make(map[string]int)
	sourceCount := make(map[string]int)
	for _, e := range errors {
		typeCount[e.ErrorType]++
		sourceCount[e.SourceRole]++
	}

	sb.WriteString(fmt.Sprintf("共检测到 %d 处语用问题:\n", len(errors)))
	if session.Mode == ModeLLML2 {
		sb.WriteString(fmt.Sprintf("- 用户消息: %d 处\n", sourceCount[ErrorSourceUser]))
		sb.WriteString(fmt.Sprintf("- LLM模拟学习者回复: %d 处\n", sourceCount[ErrorSourceLLM]))
	}
	for t, count := range typeCount {
		sb.WriteString(fmt.Sprintf("- %s: %d 处\n", t, count))
	}

	sb.WriteString("\n典型语用问题:\n")
	limit := 5
	if len(errors) < limit {
		limit = len(errors)
	}
	for _, e := range errors[:limit] {
		sourceLabel := getErrorSourceLabel(e.SourceRole, isZh)
		sb.WriteString(fmt.Sprintf("  来源: %s\n  原文: \"%s\"\n  问题: %s\n  建议: %s\n\n", sourceLabel, e.OriginalText, e.ErrorType, e.LLMSuggestion))
	}

	if isZh {
		sb.WriteString("\n请提供:\n1. 本次会话语用总体评价（2-3句）\n2. 主要语用问题类型分析\n3. 您的语用优势\n4. 如果包含 LLM模拟学习者回复，请指出这些问题是AI为训练故意触发的观察样例\n5. 针对性改进建议（3-5条）\n6. 鼓励性结语\n用友好专业的语气，分段落展示，总字数控制在300字以内。")
	} else {
		sb.WriteString("\nPlease provide:\n1. Overall pragmatic evaluation (2-3 sentences)\n2. Main pragmatic error types analysis\n3. Your pragmatic strengths\n4. If LLM learner replies are included, clarify that those issues are intentional observation samples for training\n5. Targeted improvement suggestions (3-5 tips)\n6. Encouraging closing\nUse a friendly professional tone, organize in paragraphs, under 300 words.")
	}

	prompt := sb.String()
	summary, err := callLLMChatAPI(prompt)
	if err != nil {
		log.Printf("生成会话汇总失败: %v", err)
		if isZh {
			return fmt.Sprintf("本次对话共 %d 轮，检测到 %d 处语用问题。主要问题类型：%s。建议关注语言语用规范和社会文化语用差异，继续努力！",
				session.RoundCount, len(errors), getMainErrorTypes(typeCount))
		}
		return fmt.Sprintf("This session had %d rounds with %d pragmatic issues detected. Main error types: %s. Keep practicing!",
			session.RoundCount, len(errors), getMainErrorTypes(typeCount))
	}
	return summary
}

func getErrorSourceLabel(sourceRole string, isChinese bool) string {
	if sourceRole == ErrorSourceLLM {
		if isChinese {
			return "LLM模拟学习者"
		}
		return "LLM learner"
	}
	if isChinese {
		return "用户"
	}
	return "User"
}
func getMainErrorTypes(typeCount map[string]int) string {
	types := make([]string, 0)
	for t := range typeCount {
		types = append(types, t)
	}
	return strings.Join(types, "、")
}

// callLLMChatAPI 调用 DashScope API 生成 LLM 对话回复
func callLLMChatAPI(prompt string) (string, error) {
	reqBody := DashScopeRequest{
		Model: getDashScopeModel(),
		Input: DashScopeInput{
			Messages: []DashScopeMessage{
				newDashScopeTextMessage("user", prompt),
			},
		},
		Parameters: DashScopeParameters{
			ResultFormat:        "message",
			Temperature:         0.7,
			MaxCompletionTokens: 512,
			EnableThinking:      boolPtr(false),
		},
	}

	dashResp, err := makeDashScopeRequest(getDashScopeAPIKey(), reqBody)
	if err != nil {
		return "", err
	}

	if responseText := strings.TrimSpace(getDashScopeResponseText(dashResp)); responseText != "" {
		return responseText, nil
	}
	return "", fmt.Errorf("empty response from LLM")
}

// targetLanguageToCountry 将目标语言代码转换为国家代码（用于语用检测）
func targetLanguageToCountry(lang string) string {
	switch strings.ToUpper(lang) {
	case "EN":
		return "US"
	case "ZH":
		return "CN"
	case "JP":
		return "JP"
	case "KR":
		return "KR"
	case "FR":
		return "FR"
	case "DE":
		return "DE"
	default:
		return "US"
	}
}

// getLanguageFullName 根据语言代码获取完整语言名称（英文）
func getLanguageFullName(lang string) string {
	switch strings.ToUpper(lang) {
	case "EN":
		return "English"
	case "ZH":
		return "Chinese"
	case "JP":
		return "Japanese"
	case "KR":
		return "Korean"
	case "FR":
		return "French"
	case "DE":
		return "German"
	default:
		return "English"
	}
}

// getLanguageNameByCountry 根据国家代码获取该国主要语言名称
func getLanguageNameByCountry(country string) string {
	switch country {
	case "CN", "TW", "HK", "SG":
		return "Chinese"
	case "JP":
		return "Japanese"
	case "KR":
		return "Korean"
	case "FR":
		return "French"
	case "DE":
		return "German"
	case "US", "GB", "CA", "AU":
		return "English"
	default:
		return "English"
	}
}
