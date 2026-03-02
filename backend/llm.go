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

		// 通知接收方 (用新的类型, 前端不弹窗)
		notifyReceiver(message.ReceiverID, message.ID, grammarError)
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

	// 根据发送者国籍选择语言
	isChineseSender := sender.Country == "CN"

	if isChineseSender {
		// 中文 prompt
		sb.WriteString("你是一位精通语用学和跨文化交际的语言学专家。\n")
		sb.WriteString("你的任务是分析在线聊天中平等关系交际者（如同学或陌生人）之间的话语。\n\n")

		sb.WriteString("重要约束:\n")
		sb.WriteString("- 不要假设一定存在错误。\n")
		sb.WriteString("- 不要对轻微的语法问题过度敏感。\n")
		sb.WriteString("- 不影响人际意义的轻微措辞不完美不应被视为语用失误。\n")
		sb.WriteString("- 只关注有意义的人际影响。\n\n")

		// 对话双方信息
		sb.WriteString("对话双方的文化背景:\n")
		sb.WriteString(fmt.Sprintf("- 发送者: %s, 国家: %s\n", sender.Username, getCountryName(sender.Country)))
		sb.WriteString(fmt.Sprintf("- 接收者: %s, 国家: %s\n\n", receiver.Username, getCountryName(receiver.Country)))

		// 对话历史 (最多5条)
		if len(history) > 0 {
			sb.WriteString("对话历史 (作为背景信息):\n")
			limit := len(history)
			if limit > 5 {
				limit = 5
			}
			for i := limit - 1; i >= 0; i-- {
				msg := history[i]
				speaker := "发送者"
				if msg.SenderID == receiver.ID {
					speaker = "接收者"
				}
				sb.WriteString(fmt.Sprintf("- [%s]: %s\n", speaker, msg.Content))
			}
			sb.WriteString("\n")
		}

		// 当前消息 (只判断这句)
		sb.WriteString(fmt.Sprintf("当前待分析话语 (只判断这一句): \"%s\"\n\n", current.Content))

		// 分析步骤
		sb.WriteString("分析步骤:\n\n")

		sb.WriteString("第一步: 判断话语是否包含不礼貌。\n")
		sb.WriteString("不礼貌定义为:\n")
		sb.WriteString("- 没有缓和手段的明显面子威胁行为\n")
		sb.WriteString("- 明显粗鲁、轻蔑、攻击性或贬低的语气\n")
		sb.WriteString("- 在平等地位在线互动中严重违反规范\n")
		sb.WriteString("仅仅直接并不自动等于不礼貌。\n\n")

		sb.WriteString("第二步: 判断是否存在语用失误。\n")
		sb.WriteString("A. 语用语言失误 (Pragmalinguistic failure):\n")
		sb.WriteString("- 使用不恰当的语言形式表达预期意义\n")
		sb.WriteString("- 误用惯用表达（如请求、道歉、拒绝）\n")
		sb.WriteString("- 无意间扭曲人际意义的词汇或语法选择\n")
		sb.WriteString("- 形式与功能不匹配\n")
		sb.WriteString("不影响人际意义的轻微语法错误不应计入。\n\n")

		sb.WriteString("B. 社会语用失误 (Sociopragmatic failure):\n")
		sb.WriteString("- 违反平等地位在线互动的社会规范\n")
		sb.WriteString("- 不适当的直接程度、正式程度或缓和手段\n")
		sb.WriteString("- 对人际距离或关系期望的误判\n")
		sb.WriteString("- 合理情况下会造成人际不适的表达\n")
		sb.WriteString("仅仅直接并不自动构成违反，除非明显超出合理预期。\n\n")

		sb.WriteString("第三步: 根据组合情况给出评价:\n")
		sb.WriteString("- 如果两种失误都没有 → overall_evaluation: \"good\"\n")
		sb.WriteString("- 如果只有一种失误 → overall_evaluation: \"improvable\"，并提供简要建议\n")
		sb.WriteString("- 如果两种失误都有 → overall_evaluation: \"problematic\"，并提供修改版本\n")
		sb.WriteString("例外规则: 如果任一失误极其严重（明显冒犯、强烈威胁面子或严重意义扭曲），直接判定为 \"problematic\"\n\n")

		sb.WriteString("在最终判断前，请重新考虑: 如果以最善意的合理方式解读该话语，判断是否会改变？如果会，倾向于判定为非错误。\n")
		sb.WriteString("阈值规则: 只有当一个合理的中立读者可能会感知到人际不适时，才标记为失误。\n\n")

	} else {
		// 英文 prompt
		sb.WriteString("You are a linguistics expert in pragmatics and intercultural communication.\n")
		sb.WriteString("Your task is to analyze an utterance in an online chat between equal-status interlocutors (e.g., classmates or strangers).\n\n")

		sb.WriteString("Important constraints:\n")
		sb.WriteString("- Do NOT assume that an error exists.\n")
		sb.WriteString("- Do NOT be overly sensitive to minor grammatical issues.\n")
		sb.WriteString("- Minor wording imperfections that do not affect interpersonal meaning should NOT be treated as pragmatic failure.\n")
		sb.WriteString("- Focus only on meaningful interpersonal impact.\n\n")

		// 对话双方信息
		sb.WriteString("Cultural background of the interlocutors:\n")
		sb.WriteString(fmt.Sprintf("- Sender: %s, Country: %s\n", sender.Username, getCountryName(sender.Country)))
		sb.WriteString(fmt.Sprintf("- Receiver: %s, Country: %s\n\n", receiver.Username, getCountryName(receiver.Country)))

		// 对话历史 (最多5条)
		if len(history) > 0 {
			sb.WriteString("Chat history (as background context):\n")
			limit := len(history)
			if limit > 5 {
				limit = 5
			}
			for i := limit - 1; i >= 0; i-- {
				msg := history[i]
				speaker := "Sender"
				if msg.SenderID == receiver.ID {
					speaker = "Receiver"
				}
				sb.WriteString(fmt.Sprintf("- [%s]: %s\n", speaker, msg.Content))
			}
			sb.WriteString("\n")
		}

		// 当前消息
		sb.WriteString(fmt.Sprintf("Utterance to analyze (judge ONLY this one): \"%s\"\n\n", current.Content))

		// 分析步骤
		sb.WriteString("Analysis steps:\n\n")

		sb.WriteString("Step 1: Determine whether the utterance contains impoliteness.\n")
		sb.WriteString("Impoliteness is defined as:\n")
		sb.WriteString("- Clear face-threatening acts without mitigation\n")
		sb.WriteString("- Overtly rude, dismissive, aggressive, or demeaning tone\n")
		sb.WriteString("- Strong norm violation in equal-status online interaction\n")
		sb.WriteString("Minor directness alone is NOT automatically impoliteness.\n\n")

		sb.WriteString("Step 2: Determine whether the utterance contains pragmatic failure.\n")
		sb.WriteString("A. Pragmalinguistic failure:\n")
		sb.WriteString("- Inappropriate linguistic forms used to express an intended meaning\n")
		sb.WriteString("- Misuse of conventional expressions (e.g., requests, apologies, refusals)\n")
		sb.WriteString("- Lexical or grammatical choices that unintentionally distort interpersonal meaning\n")
		sb.WriteString("- Form-function mismatch\n")
		sb.WriteString("Minor grammatical errors that do NOT affect interpersonal meaning should NOT be counted.\n\n")

		sb.WriteString("B. Sociopragmatic failure:\n")
		sb.WriteString("- Violation of social norms appropriate for equal-status online interaction\n")
		sb.WriteString("- Inappropriate level of directness, formality, or mitigation\n")
		sb.WriteString("- Misjudgment of interpersonal distance or relational expectations\n")
		sb.WriteString("- Expressions that would reasonably cause interpersonal discomfort\n")
		sb.WriteString("Directness alone is NOT automatically a violation unless it clearly exceeds reasonable expectations.\n\n")

		sb.WriteString("Step 3: Based on the combination, provide evaluation:\n")
		sb.WriteString("- If both failures are No → overall_evaluation: \"good\"\n")
		sb.WriteString("- If only one type is Yes → overall_evaluation: \"improvable\", with a brief suggestion\n")
		sb.WriteString("- If both types are Yes → overall_evaluation: \"problematic\", with a revised version\n")
		sb.WriteString("Exception rule: If either failure is extremely severe (clearly offensive, strongly face-threatening, or causing serious meaning distortion), classify as \"problematic\"\n\n")

		sb.WriteString("Before finalizing your judgment, reconsider whether your decision would change if the utterance were interpreted in the most charitable reasonable way. If yes, adjust toward non-error.\n")
		sb.WriteString("Threshold rule: Only mark as failure if a reasonable neutral reader would likely perceive interpersonal discomfort.\n\n")
	}

	// JSON 格式要求 (统一使用英文字段名, 便于解析)
	sb.WriteString("Return ONLY the following JSON, no other content:\n")
	sb.WriteString("{\n")
	sb.WriteString("  \"has_error\": true/false,\n")
	sb.WriteString("  \"impoliteness\": true/false,\n")
	sb.WriteString("  \"linguistic_pragmatic_failure\": true/false,\n")
	sb.WriteString("  \"social_pragmatic_failure\": true/false,\n")
	sb.WriteString("  \"overall_evaluation\": \"good\" / \"improvable\" / \"problematic\",\n")
	sb.WriteString("  \"suggestion\": \"suggested revision or improvement\",\n")
	sb.WriteString("  \"explanation\": \"detailed explanation of the pragmatic issue\"\n")
	sb.WriteString("}\n\n")

	sb.WriteString("CRITICAL RULE for the \"suggestion\" field:\n")
	sb.WriteString("- The \"suggestion\" field must contain ONLY the corrected/improved complete sentence that the user should say.\n")
	sb.WriteString("- Do NOT include any prefix like \"改为\", \"修改为\", \"Change to\", \"Try\", \"Revised\" etc.\n")
	sb.WriteString("- Do NOT include alternative options like \"或更自然的...\", \"or alternatively...\".\n")
	sb.WriteString("- Do NOT include any explanation in the suggestion field - put explanations in the \"explanation\" field.\n")
	sb.WriteString("- Just output the single best corrected sentence, nothing else.\n")

	sb.WriteString("If no pragmatic failure is detected, return:\n")
	sb.WriteString("{\"has_error\": false, \"impoliteness\": false, \"linguistic_pragmatic_failure\": false, \"social_pragmatic_failure\": false, \"overall_evaluation\": \"good\", \"suggestion\": \"\", \"explanation\": \"\"}\n\n")

	// 语言选择提醒
	if isChineseSender {
		sb.WriteString("重要提醒: suggestion 和 explanation 字段请用中文回复。\n")
	} else {
		sb.WriteString("Important: Please write the suggestion and explanation fields in English.\n")
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
func notifyReceiver(receiverID uint, messageID uint, grammarError GrammarError) {
	wsMsg := WSMessage{
		Type: "receiver_error_notify",
		Data: map[string]interface{}{
			"message_id":  messageID,
			"error_type":  grammarError.ErrorType,
			"suggestion":  grammarError.LLMSuggestion,
			"explanation": grammarError.LLMExplanation,
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
