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
	SessionErrorCount int                   `json:"session_error_count"`
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

	// 1. 生成LLM回复。语用反馈在会话结束时统一生成和保存。
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

	// 2. 更新轮次计数
	session.RoundCount++
	db.Model(&session).Update("round_count", session.RoundCount)

	// 3. 判断是否需要自动结束（五轮模式）
	sessionEnded := false
	sessionSummary := ""
	sessionErrorCount := 0

	if session.FeedbackMode == FeedbackRounds5 && session.RoundCount >= 5 {
		// 获取会话所有消息生成汇总
		var allMessages []Message
		db.Where("session_id = ?", session.ID).Order("created_at ASC").
			Preload("Sender").Preload("Receiver").Find(&allMessages)

		sessionSummary, sessionErrorCount = generateAndStoreSessionFeedback(session, allMessages, user)

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
		PragmaticCheck:    &PragmaticCheckResult{HasError: false},
		SessionEnded:      sessionEnded,
		SessionSummary:    sessionSummary,
		SessionErrorCount: sessionErrorCount,
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
	llmCountry := getLLMPersonaNativeCountry(session)
	llmCulture := getCountryName(llmCountry)

	sb.WriteString(fmt.Sprintf("You are a native %s speaker having a natural conversation with a language learner.\n", targetLangFull))
	sb.WriteString(fmt.Sprintf("Your cultural background is %s. Keep your replies consistent with that cultural background and the relationship context.\n", llmCulture))
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
	learnerCountry := getLLMPersonaNativeCountry(session)
	learnerCulture := getCountryName(learnerCountry)
	learnerNativeLang := getLanguageNameByCountry(learnerCountry)

	sb.WriteString(fmt.Sprintf("You are a language learner from %s. Your native/main language is %s, not %s. ", learnerCulture, learnerNativeLang, targetLangFull))
	sb.WriteString(fmt.Sprintf("You are learning %s as a second language and your level is intermediate.\n", targetLangFull))
	sb.WriteString(fmt.Sprintf("Your relationship with the user is: %s\n", relationship))
	sb.WriteString(fmt.Sprintf("Conversation topic: %s\n", topic))
	sb.WriteString(fmt.Sprintf("The user is a native %s speaker.\n\n", userNativeLang))

	sb.WriteString("Instructions - simulate an intermediate L2 speaker for pragmatic-awareness training:\n")
	sb.WriteString("- Keep the conversation natural and relevant to the user's message.\n")
	sb.WriteString("- Do not make every problem a social/cultural politeness problem. Use a balanced mix of pragmalinguistic and sociopragmatic issues across the session.\n")
	sb.WriteString("- Pragmalinguistic patterns to use naturally: odd word order, poor word choice, reversed sentence parts, typo-like spelling or wrong character choice, missing small function words, awkward collocations, literal transfer from your native language, or expressions that make the speech act sound too blunt or unclear.\n")
	sb.WriteString("- Sociopragmatic patterns to use occasionally: wrong politeness level, too much/too little mitigation, culturally unusual apology/thanks, awkward refusal, or mismatched distance/power expectations.\n")
	sb.WriteString("- Do not force an error into every reply. If the user's message is simple or low-stakes, reply mostly naturally with only mild non-native phrasing or no obvious issue.\n")
	sb.WriteString("- Aim for noticeable but realistic L2 features in about half of replies; make serious social-pragmatic problems less frequent than small wording/order/choice problems.\n")
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

type SessionPragmaticAnalysis struct {
	Summary string                  `json:"summary"`
	Issues  []SessionPragmaticIssue `json:"issues"`
}

type SessionPragmaticIssue struct {
	SourceRole        string `json:"source_role"`
	OriginalText      string `json:"original_text"`
	ErrorType         string `json:"error_type"`
	LLMSuggestion     string `json:"llm_suggestion"`
	LLMExplanation    string `json:"llm_explanation"`
	OverallEvaluation string `json:"overall_evaluation"`
}

// generateAndStoreSessionFeedback 生成会话级反馈，并只保存本次反馈中的代表性问题。
func generateAndStoreSessionFeedback(session ConversationSession, messages []Message, user User) (string, int) {
	analysis, err := analyzeSessionPragmatics(session, messages, user)
	if err != nil {
		log.Printf("生成会话级语用反馈失败: %v", err)
		summary := buildFallbackSessionSummary(session, user)
		replaceSessionFeedbackRecords(session, user.ID, nil)
		return summary, 0
	}

	if strings.TrimSpace(analysis.Summary) == "" {
		analysis.Summary = buildFallbackSessionSummary(session, user)
	}
	count := replaceSessionFeedbackRecords(session, user.ID, analysis.Issues)
	return analysis.Summary, count
}

// generateSessionSummary 保留旧调用入口，内部改为会话级反馈。
func generateSessionSummary(session ConversationSession, messages []Message, user User) string {
	summary, _ := generateAndStoreSessionFeedback(session, messages, user)
	return summary
}

func analyzeSessionPragmatics(session ConversationSession, messages []Message, user User) (*SessionPragmaticAnalysis, error) {
	prompt := buildSessionFeedbackPrompt(session, messages, user)
	var analysis SessionPragmaticAnalysis
	if err := callDashScopeJSON(prompt, &analysis, 1800); err != nil {
		return nil, err
	}
	return &analysis, nil
}

func replaceSessionFeedbackRecords(session ConversationSession, userID uint, issues []SessionPragmaticIssue) int {
	db.Where("user_id = ? AND session_id = ?", userID, session.ID).Delete(&GrammarError{})

	saved := 0
	for _, issue := range issues {
		errorType := normalizeErrorType(issue.ErrorType)
		sourceRole := normalizeSourceRole(issue.SourceRole)
		overallEvaluation := strings.TrimSpace(issue.OverallEvaluation)
		if overallEvaluation == "" {
			if IsProblematicErrorType(errorType) {
				overallEvaluation = "problematic"
			} else {
				overallEvaluation = "improvable"
			}
		}

		originalText := strings.TrimSpace(issue.OriginalText)
		if originalText == "" {
			originalText = "Session-level pragmatic feedback"
		}

		record := GrammarError{
			UserID:            userID,
			SessionID:         session.ID,
			MessageID:         0,
			SourceRole:        sourceRole,
			OriginalText:      originalText,
			LLMSuggestion:     strings.TrimSpace(issue.LLMSuggestion),
			LLMExplanation:    strings.TrimSpace(issue.LLMExplanation),
			ErrorType:         errorType,
			OverallEvaluation: overallEvaluation,
		}
		if err := db.Create(&record).Error; err != nil {
			log.Printf("保存会话级语用反馈失败: %v", err)
			continue
		}
		saved++
	}
	return saved
}

func normalizeErrorType(errorType string) string {
	errorType = strings.TrimSpace(errorType)
	if IsValidErrorType(errorType) {
		return errorType
	}
	return ErrorTypePragmalinguistic
}

func normalizeSourceRole(sourceRole string) string {
	switch strings.TrimSpace(sourceRole) {
	case ErrorSourceLLM:
		return ErrorSourceLLM
	default:
		return ErrorSourceUser
	}
}

func buildFallbackSessionSummary(session ConversationSession, user User) string {
	targetLang := getLanguageFullName(session.TargetLanguage)
	if isChineseUser(user) {
		return fmt.Sprintf("本次%d轮%s对话已完成。系统未能生成详细结构化反馈，但建议继续关注对话关系、礼貌程度、请求/拒绝/感谢等表达方式与文化预期是否匹配。", session.RoundCount, targetLang)
	}
	return fmt.Sprintf("This %d-round %s conversation is complete. A detailed structured report could not be generated, but keep watching whether politeness, requests, refusals, thanks, and relationship management match the cultural context.", session.RoundCount, targetLang)
}

func buildSessionFeedbackPrompt(session ConversationSession, messages []Message, user User) string {
	var sb strings.Builder
	isZh := isChineseUser(user)
	targetLang := getLanguageFullName(session.TargetLanguage)
	humanCulture := getCountryName(user.Country)
	llmCulture := getCountryName(getLLMPersonaNativeCountry(session))
	llmNativeLang := getLanguageNameByCountry(getLLMPersonaNativeCountry(session))

	if isZh {
		sb.WriteString("你是 PFChat 的跨文化语用学会话反馈评估器。请在完整会话结束后进行一次性分析，而不是逐句批改。\n")
		sb.WriteString("你的输出将被保存为本次会话的反馈记录；请只选择最有代表性的 0-6 个问题，不要为每句话都创建记录。\n\n")
		sb.WriteString(fmt.Sprintf("会话模式: %s\n", getSessionModePrompt(session.Mode, true)))
		sb.WriteString(fmt.Sprintf("练习语言: %s\n", targetLang))
		sb.WriteString(fmt.Sprintf("对话关系: %s\n", session.RelationshipType))
		sb.WriteString(fmt.Sprintf("对话主题: %s\n", session.Topic))
		sb.WriteString(fmt.Sprintf("Human Listener/Speaker 文化背景: %s\n", humanCulture))
		sb.WriteString(fmt.Sprintf("LLM Speaker 文化背景: %s，母语/主要语言: %s\n\n", llmCulture, llmNativeLang))
	} else {
		sb.WriteString("You are PFChat's cross-cultural pragmatics session-feedback evaluator. Analyze the completed conversation once, not sentence by sentence.\n")
		sb.WriteString("Your output will be stored as this session's feedback record. Select only the 0-6 most representative issues; do not create a record for every utterance.\n\n")
		sb.WriteString(fmt.Sprintf("Session mode: %s\n", getSessionModePrompt(session.Mode, false)))
		sb.WriteString(fmt.Sprintf("Practice language: %s\n", targetLang))
		sb.WriteString(fmt.Sprintf("Relationship: %s\n", session.RelationshipType))
		sb.WriteString(fmt.Sprintf("Topic: %s\n", session.Topic))
		sb.WriteString(fmt.Sprintf("Human Listener/Speaker cultural background: %s\n", humanCulture))
		sb.WriteString(fmt.Sprintf("LLM Speaker cultural background: %s; native/main language: %s\n\n", llmCulture, llmNativeLang))
	}

	if session.Mode == ModeLLML2 {
		if isZh {
			sb.WriteString("LLM Speaker 是第二语言学习者。它的失误是训练样例：解释必须写给 Human Listener，说明听者可能如何理解、可以观察什么，不要对 LLM Speaker 说教。\n")
			sb.WriteString("请平衡识别语用语言失误和社会语用失误。除礼貌/关系误判外，也要关注自然的二语问题：语序错误、用词不当、句子成分颠倒、错别字/拼写近似、搭配生硬；只有这些影响意图、礼貌或理解时才列为语用语言失误。\n\n")
		} else {
			sb.WriteString("The LLM Speaker is an L2 learner. Its mistakes are training samples: explanations must be written for the Human Listener, describing how the listener may interpret them and what to observe. Do not lecture the LLM Speaker.\n")
			sb.WriteString("Balance pragmalinguistic and sociopragmatic issues. In addition to politeness or relationship mismatches, notice natural L2 problems such as word order errors, poor word choice, reversed sentence parts, typo-like spelling, and awkward collocations; list them as pragmalinguistic only when they affect intent, politeness, or understanding.\n\n")
		}
	}

	if isZh {
		sb.WriteString("完整对话记录:\n")
	} else {
		sb.WriteString("Complete conversation:\n")
	}
	for i, msg := range messages {
		role := "Human"
		if msg.Role == "llm" {
			role = "LLM Speaker"
		}
		sb.WriteString(fmt.Sprintf("%d. %s: %s\n", i+1, role, msg.Content))
	}

	if isZh {
		sb.WriteString("\n请只输出纯 JSON，不要 markdown。格式如下:\n")
	} else {
		sb.WriteString("\nReturn pure JSON only, no markdown. Use this shape:\n")
	}
	sb.WriteString("{\n")
	sb.WriteString("  \"summary\": \"session-level feedback report in the human user's language, 180-300 words\",\n")
	sb.WriteString("  \"issues\": [\n")
	sb.WriteString("    {\n")
	sb.WriteString("      \"source_role\": \"user/llm\",\n")
	sb.WriteString("      \"original_text\": \"short representative excerpt or session-level pattern, not necessarily a full message\",\n")
	sb.WriteString("      \"error_type\": \"语用语言失误/社会语用失误/严重语用语言失误/严重社会语用失误/语用语言失误和社会语用失误\",\n")
	sb.WriteString("      \"llm_suggestion\": \"one best revised wording or concise listening strategy\",\n")
	sb.WriteString("      \"llm_explanation\": \"brief explanation for the human user/listener\",\n")
	sb.WriteString("      \"overall_evaluation\": \"improvable/problematic\"\n")
	sb.WriteString("    }\n")
	sb.WriteString("  ]\n")
	sb.WriteString("}\n")
	return sb.String()
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

func isChineseUser(user User) bool {
	switch user.Country {
	case "CN", "TW", "HK", "SG":
		return true
	default:
		return false
	}
}

func getLLMPersonaNativeCountry(session ConversationSession) string {
	if session.Mode == ModeUserL2 {
		return targetLanguageToCountry(session.TargetLanguage)
	}
	return getLearnerNativeCountryForTarget(session.TargetLanguage)
}

func getLearnerNativeCountryForTarget(targetLang string) string {
	switch strings.ToUpper(targetLang) {
	case "EN", "FR", "DE":
		return "CN"
	case "ZH":
		return "US"
	case "JP":
		return "US"
	case "KR":
		return "JP"
	default:
		return "CN"
	}
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
