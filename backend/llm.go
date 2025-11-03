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

// Ollama API 配置
const (
	ollamaAPIURL = "http://localhost:11434/api/generate" // Ollama 默认地址
	ollamaModel  = "qwen3:4b"                            // 使用 qwen3:4b 模型
)

// Ollama 请求结构
type OllamaRequest struct {
	Model   string                 `json:"model"`
	Prompt  string                 `json:"prompt"`
	Stream  bool                   `json:"stream"`
	Options map[string]interface{} `json:"options,omitempty"`
}

// Ollama 响应结构
type OllamaResponse struct {
	Model     string `json:"model"`
	CreatedAt string `json:"created_at"`
	Response  string `json:"response"`
	Done      bool   `json:"done"`
}

// GrammarCheckResponse 语用失误检查响应结构
type GrammarCheckResponse struct {
	HasError    bool   `json:"has_error"`
	Suggestion  string `json:"suggestion"`
	Explanation string `json:"explanation"`
	ErrorType   string `json:"error_type"` // "语言语用失误" 或 "社会语用失误"
}

// 执行语用失误检查
func checkGrammar(userID uint, message Message) {
	// 获取发送者和接收者信息
	var sender, receiver User
	db.First(&sender, message.SenderID)
	db.First(&receiver, message.ReceiverID)

	// 获取最近的历史消息作为上下文
	var historyMessages []Message
	db.Where(
		"((sender_id = ? AND receiver_id = ?) OR (sender_id = ? AND receiver_id = ?)) AND id < ?",
		message.SenderID, message.ReceiverID, message.ReceiverID, message.SenderID, message.ID,
	).Order("sent_at DESC").Limit(10).Preload("Sender").Preload("Receiver").Find(&historyMessages)

	// 构建上下文和提示词
	prompt := buildPrompt(historyMessages, message, sender, receiver)

	// 调用 Ollama API
	result, err := callOllamaAPI(prompt)
	if err != nil {
		log.Printf("Ollama API error: %v", err)
		return
	}

	// 如果有语用失误，保存记录并通知用户
	if result.HasError {
		// 确保错误类型有效，如果 LLM 未返回或返回无效值，使用默认值
		errorType := result.ErrorType
		if errorType != "语言语用失误" && errorType != "社会语用失误" {
			errorType = "语言语用失误" // 默认为语言语用失误
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

		// 通过 WebSocket 通知用户
		notifyUser(userID, grammarError)
	}
}

// 构建提示词（支持语用失误检查）
func buildPrompt(history []Message, current Message, sender User, receiver User) string {
	var sb strings.Builder

	sb.WriteString("你是一个语言学家，根据对话双方的聊天历史及其文化背景检查用户当前输入的语言是否有语用失误。")
	sb.WriteString("如果有语用失误，请根据Thomas的定义，判断该失误属于语言语用失误还是社会语用失误。\n\n")

	// 添加对话历史
	if len(history) > 0 {
		sb.WriteString("对话历史:\n")
		// 按时间顺序显示（history 是倒序的）
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
	sb.WriteString(fmt.Sprintf("- 发送者(sender): %s, 国家: %s\n", sender.Username, getCountryName(sender.Country)))
	sb.WriteString(fmt.Sprintf("- 接收者(receiver): %s, 国家: %s\n\n", receiver.Username, getCountryName(receiver.Country)))

	// 当前消息
	sb.WriteString(fmt.Sprintf("当前消息: %s\n\n", current.Content))

	// 语用失误分类标准
	sb.WriteString("语用失误分类标准（Thomas理论）:\n")
	sb.WriteString("1. 语言语用失误(Pragmalinguistic failure):\n")
	sb.WriteString("   - 语言形式使用不当，如不恰当的言语行为、语气、礼貌标记等\n")
	sb.WriteString("   - 例如：请求时过于直接、道歉不够真诚、感谢表达方式不当等\n\n")
	sb.WriteString("2. 社会语用失误(Sociopragmatic failure):\n")
	sb.WriteString("   - 违反社会文化规范，如不了解对方文化的禁忌、礼节、价值观等\n")
	sb.WriteString("   - 例如：话题不当、称呼不当、忽视文化差异、违反社交距离等\n\n")

	// JSON 格式要求
	sb.WriteString("请严格按照以下 JSON 格式返回结果，不要添加任何其他内容:\n")
	sb.WriteString("{\n")
	sb.WriteString("  \"has_error\": true/false,\n")
	sb.WriteString("  \"suggestion\": \"建议的表达方式\",\n")
	sb.WriteString("  \"explanation\": \"语用失误的详细说明，包括为什么这样说不恰当，以及文化背景因素\",\n")
	sb.WriteString("  \"error_type\": \"语言语用失误 或 社会语用失误\"\n")
	sb.WriteString("}\n\n")
	sb.WriteString("如果没有语用失误，请返回:\n")
	sb.WriteString("{\"has_error\": false, \"suggestion\": \"\", \"explanation\": \"\", \"error_type\": \"\"}")

	return sb.String()
}

// 获取国家全称
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

// 调用 Ollama API
func callOllamaAPI(prompt string) (*GrammarCheckResponse, error) {
	// 构建请求
	reqBody := OllamaRequest{
		Model:  ollamaModel,
		Prompt: prompt,
		Stream: false,
		Options: map[string]interface{}{
			"temperature": 0.3, // 降低温度以获得更确定的结果
			"top_p":       0.9,
		},
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshal request error: %w", err)
	}

	// 创建 HTTP 请求
	req, err := http.NewRequest("POST", ollamaAPIURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("create request error: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	// 发送请求（增加超时时间，本地模型可能响应较慢）
	client := &http.Client{
		Timeout: 60 * time.Second,
	}

	resp, err := client.Do(req)
	if err != nil {
		log.Printf("Ollama API request failed: %v", err)
		// 如果 API 调用失败，返回无错误（避免阻塞消息发送）
		return &GrammarCheckResponse{HasError: false}, nil
	}
	defer resp.Body.Close()

	// 读取响应
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response error: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		log.Printf("Ollama API error: status=%d, body=%s", resp.StatusCode, string(body))
		return &GrammarCheckResponse{HasError: false}, nil
	}

	// 解析 Ollama 响应
	var ollamaResp OllamaResponse
	if err := json.Unmarshal(body, &ollamaResp); err != nil {
		return nil, fmt.Errorf("unmarshal Ollama response error: %w", err)
	}

	// 提取 JSON 内容
	content := extractJSON(ollamaResp.Response)

	// 解析语用失误检查结果
	var result GrammarCheckResponse
	if err := json.Unmarshal([]byte(content), &result); err != nil {
		log.Printf("Failed to parse LLM response as JSON: %s", content)
		// 如果无法解析，尝试从文本中提取信息
		return parseTextResponse(ollamaResp.Response), nil
	}

	// 验证和规范化错误类型
	if result.HasError && result.ErrorType != "语言语用失误" && result.ErrorType != "社会语用失误" {
		// 如果 LLM 返回的错误类型无效，尝试从 explanation 中推断
		result.ErrorType = inferErrorType(result.Explanation)
	}

	return &result, nil
}

// 提取 JSON 内容（处理模型可能返回的额外文本）
func extractJSON(text string) string {
	// 查找 JSON 对象的开始和结束位置
	start := strings.Index(text, "{")
	end := strings.LastIndex(text, "}")

	if start == -1 || end == -1 || start > end {
		return text
	}

	return text[start : end+1]
}

// 从文本响应中解析结果（备用方案）
func parseTextResponse(text string) *GrammarCheckResponse {
	text = strings.ToLower(text)

	// 如果包含"没有失误"、"correct"等关键词，认为没有错误
	noErrorKeywords := []string{
		"没有失误", "没有语用失误", "no error", "correct",
		"恰当", "appropriate", "合适", "suitable",
	}
	for _, keyword := range noErrorKeywords {
		if strings.Contains(text, keyword) {
			return &GrammarCheckResponse{
				HasError:    false,
				Suggestion:  "",
				Explanation: "",
				ErrorType:   "",
			}
		}
	}

	// 如果包含"失误"、"error"等关键词，但无法解析详细信息
	errorKeywords := []string{"失误", "error", "mistake", "inappropriate", "不当", "不恰当"}
	for _, keyword := range errorKeywords {
		if strings.Contains(text, keyword) {
			// 尝试推断错误类型
			errorType := inferErrorType(text)
			return &GrammarCheckResponse{
				HasError:    true,
				Suggestion:  "",
				Explanation: "检测到可能的语用失误，但无法提供详细建议",
				ErrorType:   errorType,
			}
		}
	}

	// 默认返回无错误
	return &GrammarCheckResponse{
		HasError:    false,
		Suggestion:  "",
		Explanation: "",
		ErrorType:   "",
	}
}

// 根据错误说明推断错误类型
func inferErrorType(explanation string) string {
	explanation = strings.ToLower(explanation)

	// 社会语用失误的关键词
	sociopragmaticKeywords := []string{
		// 中文关键词
		"文化", "禁忌", "价值观", "社会规范", "社交距离", "身份", "地位",
		"话题不当", "称呼不当", "礼节", "习俗", "传统", "宗教",
		"性别", "年龄", "等级", "权力关系",
		// 英文关键词
		"culture", "cultural", "taboo", "values", "social norm", "social distance",
		"status", "identity", "hierarchy", "religion", "tradition", "custom",
		"inappropriate topic", "inappropriate address", "gender", "age", "power",
	}

	// 语言语用失误的关键词
	pragmalinguisticKeywords := []string{
		// 中文关键词
		"语气", "礼貌", "客气", "直接", "间接", "委婉", "请求方式",
		"道歉", "感谢", "拒绝", "邀请", "赞美", "批评",
		"言语行为", "话语标记", "语言形式", "表达方式",
		// 英文关键词
		"tone", "politeness", "polite", "direct", "indirect", "request",
		"apology", "thanks", "refusal", "invitation", "compliment", "criticism",
		"speech act", "discourse marker", "linguistic form", "expression",
	}

	// 检查是否包含社会语用失误关键词
	for _, keyword := range sociopragmaticKeywords {
		if strings.Contains(explanation, keyword) {
			return "社会语用失误"
		}
	}

	// 检查是否包含语言语用失误关键词
	for _, keyword := range pragmalinguisticKeywords {
		if strings.Contains(explanation, keyword) {
			return "语言语用失误"
		}
	}

	// 默认返回语言语用失误
	return "语言语用失误"
}

// 通知用户语用失误
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

	// 通过 hub 发送给用户
	if globalHub != nil {
		globalHub.sendToUser(userID, msgBytes)
	}
}

// 全局 hub 实例（在 main.go 中初始化后设置）
var globalHub *Hub
