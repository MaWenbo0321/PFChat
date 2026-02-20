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

// GrammarCheckResponse 语用失误检查响应结构
type GrammarCheckResponse struct {
	HasError    bool   `json:"has_error"`
	Suggestion  string `json:"suggestion"`
	Explanation string `json:"explanation"`
	ErrorType   string `json:"error_type"`
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
		errorType := result.ErrorType
		if errorType != "语言语用失误" && errorType != "社会语用失误" {
			errorType = "语言语用失误"
		}

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

		// 🆕 通知发送方
		notifyUser(userID, grammarError)

		// 🆕 通知接收方 (用新的类型, 前端不弹窗)
		notifyReceiver(message.ReceiverID, message.ID, grammarError)
	}
}

// buildCombinedPrompt 构建合并的提示词
func buildCombinedPrompt(history []Message, current Message, sender User, receiver User) string {
	var sb strings.Builder

	sb.WriteString("你是一个语言学家，根据对话双方的聊天历史及其文化背景检查用户当前输入的语言是否有语用失误。\n")
	sb.WriteString("如果有语用失误，请根据Thomas的定义，判断该失误属于语言语用失误还是社会语用失误。\n\n")

	sb.WriteString("语用失误分类标准（Thomas理论）:\n")
	sb.WriteString("1. 语言语用失误(Pragmalinguistic failure):\n")
	sb.WriteString("   - 语言形式使用不当，如不恰当的言语行为、语气、礼貌标记等\n")
	sb.WriteString("   - 例如：请求时过于直接、道歉不够真诚、感谢表达方式不当等\n\n")
	sb.WriteString("2. 社会语用失误(Sociopragmatic failure):\n")
	sb.WriteString("   - 违反社会文化规范，如不了解对方文化的禁忌、礼节、价值观等\n")
	sb.WriteString("   - 例如：话题不当、称呼不当、忽视文化差异、违反社交距离等\n\n")

	// 判断标准
	sb.WriteString("判断标准:\n")
	sb.WriteString("- 只标记明显的、可能导致严重误解或冒犯的语用失误\n")
	sb.WriteString("- 如果不确定是否为语用失误, 倾向于判定为没有失误\n")
	sb.WriteString("- 正常的文化表达变体不应被标记为错误\n\n")

	// 对话历史
	if len(history) > 0 {
		sb.WriteString("对话历史:\n")
		for i := len(history) - 1; i >= 0; i-- {
			msg := history[i]
			speaker := "发送者"
			if msg.SenderID == receiver.ID {
				speaker = "接收者"
			}
			sb.WriteString(fmt.Sprintf("- [%s]: %s\n", speaker, msg.Content))
		}
		sb.WriteString("\n")
	}

	// 对话双方信息
	sb.WriteString("对话双方的文化背景:\n")
	sb.WriteString(fmt.Sprintf("- 发送者: %s, 国家: %s\n", sender.Username, getCountryName(sender.Country)))
	sb.WriteString(fmt.Sprintf("- 接收者: %s, 国家: %s\n\n", receiver.Username, getCountryName(receiver.Country)))

	// 当前消息
	sb.WriteString(fmt.Sprintf("当前消息: %s\n\n", current.Content))

	// JSON 格式要求
	sb.WriteString("请严格按照以下 JSON 格式返回结果，不要添加任何其他内容:\n")
	sb.WriteString("{\n")
	sb.WriteString("  \"has_error\": true/false,\n")
	sb.WriteString("  \"suggestion\": \"建议的表达方式\",\n")
	sb.WriteString("  \"explanation\": \"语用失误的详细说明\",\n")
	sb.WriteString("  \"error_type\": \"语言语用失误 或 社会语用失误\"\n")
	sb.WriteString("}\n\n")
	sb.WriteString("如果没有语用失误，请返回:\n")
	sb.WriteString("{\"has_error\": false, \"suggestion\": \"\", \"explanation\": \"\", \"error_type\": \"\"}\n\n")
	sb.WriteString("重要提醒:请根据发送者的国籍选择回复的语言。如果发送者是中国用户，请用中文回复；如果发送者是其他国家用户，请用英文回复。")

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
	if apiKey == "" {
		return nil, fmt.Errorf("DASHSCOPE_API_KEY 环境变量未设置")
	}

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

	if result.HasError {
		if result.ErrorType != "语言语用失误" && result.ErrorType != "社会语用失误" {
			result.ErrorType = inferErrorType(result.Explanation)
		}
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

func inferErrorType(explanation string) string {
	explanation = strings.ToLower(explanation)

	sociopragmaticKeywords := []string{
		"文化", "禁忌", "价值观", "社会规范", "社交距离",
		"culture", "cultural", "taboo", "values", "social norm",
	}

	for _, keyword := range sociopragmaticKeywords {
		if strings.Contains(explanation, keyword) {
			return "社会语用失误"
		}
	}

	return "语言语用失误"
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

// 🆕 notifyReceiver 通知接收方有语用错误 (不弹窗, 只刷新标识)
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
