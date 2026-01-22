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
		Text   string `json:"text,omitempty"`
		Finish string `json:"finish_reason,omitempty"`
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

// 执行语用失误检查
func checkGrammar(userID uint, message Message) {
	var sender, receiver User
	db.First(&sender, message.SenderID)
	db.First(&receiver, message.ReceiverID)

	var historyMessages []Message
	db.Where(
		"((sender_id = ? AND receiver_id = ?) OR (sender_id = ? AND receiver_id = ?)) AND id < ?",
		message.SenderID, message.ReceiverID, message.ReceiverID, message.SenderID, message.ID,
	).Order("created_at DESC").Limit(10).Preload("Sender").Preload("Receiver").Find(&historyMessages)

	// 🔧 修改：合并 system 和 user prompt
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

		notifyUser(userID, grammarError)
	}
}

// 🔧 修改：构建合并的提示词（不使用 system 角色）
func buildCombinedPrompt(history []Message, current Message, sender User, receiver User) string {
	var sb strings.Builder

	// 将系统提示合并到用户消息中
	sb.WriteString("你是一个语言学家，根据对话双方的聊天历史及其文化背景检查用户当前输入的语言是否有语用失误。\n")
	sb.WriteString("如果有语用失误，请根据Thomas的定义，判断该失误属于语言语用失误还是社会语用失误。\n\n")

	sb.WriteString("语用失误分类标准（Thomas理论）:\n")
	sb.WriteString("1. 语言语用失误(Pragmalinguistic failure):\n")
	sb.WriteString("   - 语言形式使用不当，如不恰当的言语行为、语气、礼貌标记等\n")
	sb.WriteString("   - 例如：请求时过于直接、道歉不够真诚、感谢表达方式不当等\n\n")
	sb.WriteString("2. 社会语用失误(Sociopragmatic failure):\n")
	sb.WriteString("   - 违反社会文化规范，如不了解对方文化的禁忌、礼节、价值观等\n")
	sb.WriteString("   - 例如：话题不当、称呼不当、忽视文化差异、违反社交距离等\n\n")

	// 添加对话历史
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

	// 添加对话双方的文化背景
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

// 🔧 修改：调用 DashScope API（只使用 user 角色）
func callDashScopeAPI(prompt string) (*GrammarCheckResponse, error) {
	apiKey := "sk-8ab77da79b894ba6beb61c9190c74602"
	if apiKey == "" {
		return nil, fmt.Errorf("DASHSCOPE_API_KEY 环境变量未设置")
	}

	// 🔧 只使用 user 角色
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
			Temperature:  0.7,
			MaxTokens:    1500,
		},
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshal request failed: %v", err)
	}

	log.Printf("模型: %s", defaultModel)
	log.Printf("Prompt 长度: %d 字符", len(prompt))

	req, err := http.NewRequest("POST", dashScopeAPIURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("create request failed: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	log.Printf("发送请求到 DashScope...")
	client := &http.Client{Timeout: 30 * time.Second}
	startTime := time.Now()
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %v", err)
	}
	defer resp.Body.Close()

	log.Printf("响应时间: %v", time.Since(startTime))
	log.Printf("状态码: %d", resp.StatusCode)

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response body failed: %v", err)
	}

	log.Printf("原始响应: %s", string(body))

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API返回状态码 %d: %s", resp.StatusCode, string(body))
	}

	var dashScopeResp DashScopeResponse
	if err := json.Unmarshal(body, &dashScopeResp); err != nil {
		return nil, fmt.Errorf("unmarshal response failed: %v", err)
	}

	var responseText string
	if len(dashScopeResp.Output.Choices) > 0 {
		responseText = dashScopeResp.Output.Choices[0].Message.Content
	} else if dashScopeResp.Output.Text != "" {
		responseText = dashScopeResp.Output.Text
	} else {
		return nil, fmt.Errorf("empty response from API")
	}

	log.Printf("LLM 返回内容: %s", responseText)

	jsonText := extractJSON(responseText)
	var result GrammarCheckResponse

	if err := json.Unmarshal([]byte(jsonText), &result); err != nil {
		log.Printf("JSON 解析失败: %v, 尝试文本解析...", err)
		// 如果 JSON 解析失败，返回无错误
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

var globalHub *Hub