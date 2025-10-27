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
	ollamaModel  = "qwen3:4b"                            // 使用 qwen2.5:3b 模型
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

// GrammarCheckResponse 语法检查响应结构
type GrammarCheckResponse struct {
	HasError    bool   `json:"has_error"`
	Suggestion  string `json:"suggestion"`
	Explanation string `json:"explanation"`
}

// 执行语法检查
func checkGrammar(userID uint, message Message) {
	// 获取最近的历史消息作为上下文
	var historyMessages []Message
	db.Where(
		"(sender_id = ? OR receiver_id = ?) AND id < ?",
		userID, userID, message.ID,
	).Order("sent_at DESC").Limit(5).Find(&historyMessages)

	// 构建上下文和提示词
	prompt := buildPrompt(historyMessages, message)

	// 调用 Ollama API
	result, err := callOllamaAPI(prompt)
	if err != nil {
		log.Printf("Ollama API error: %v", err)
		return
	}

	// 如果有语法错误，保存记录并通知用户
	if result.HasError {
		grammarError := GrammarError{
			UserID:         userID,
			MessageID:      message.ID,
			OriginalText:   message.Content,
			LLMSuggestion:  result.Suggestion,
			LLMExplanation: result.Explanation,
		}

		if err := db.Create(&grammarError).Error; err != nil {
			log.Printf("Save grammar error failed: %v", err)
			return
		}

		// 通过 WebSocket 通知用户
		notifyUser(userID, grammarError)
	}
}

// 构建提示词
func buildPrompt(history []Message, current Message) string {
	var sb strings.Builder

	sb.WriteString("你是一个专业的英语语法检查助手。请检查用户的英语消息是否有语法错误。\n\n")

	// 添加对话历史（倒序，因为查询时用的是 DESC）
	if len(history) > 0 {
		sb.WriteString("对话历史:\n")
		for i := len(history) - 1; i >= 0; i-- {
			sb.WriteString(fmt.Sprintf("- %s\n", history[i].Content))
		}
		sb.WriteString("\n")
	}

	sb.WriteString(fmt.Sprintf("当前消息: %s\n\n", current.Content))

	sb.WriteString("请严格按照以下 JSON 格式返回结果，不要添加任何其他内容:\n")
	sb.WriteString("{\n")
	sb.WriteString("  \"has_error\": true/false,\n")
	sb.WriteString("  \"suggestion\": \"修改后的正确句子\",\n")
	sb.WriteString("  \"explanation\": \"错误说明\"\n")
	sb.WriteString("}\n\n")
	sb.WriteString("如果没有语法错误，请返回:\n")
	sb.WriteString("{\"has_error\": false, \"suggestion\": \"\", \"explanation\": \"\"}")

	return sb.String()
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

	// 解析语法检查结果
	var result GrammarCheckResponse
	if err := json.Unmarshal([]byte(content), &result); err != nil {
		log.Printf("Failed to parse LLM response as JSON: %s", content)
		// 如果无法解析，尝试从文本中提取信息
		return parseTextResponse(ollamaResp.Response), nil
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

	// 如果包含"没有错误"、"correct"等关键词，认为没有错误
	noErrorKeywords := []string{"没有错误", "no error", "correct", "语法正确", "没有语法错误"}
	for _, keyword := range noErrorKeywords {
		if strings.Contains(text, keyword) {
			return &GrammarCheckResponse{
				HasError:    false,
				Suggestion:  "",
				Explanation: "",
			}
		}
	}

	// 如果包含"错误"、"error"等关键词，但无法解析详细信息
	errorKeywords := []string{"错误", "error", "mistake", "incorrect", "wrong"}
	for _, keyword := range errorKeywords {
		if strings.Contains(text, keyword) {
			return &GrammarCheckResponse{
				HasError:    true,
				Suggestion:  "",
				Explanation: "检测到可能的语法问题，但无法提供详细建议",
			}
		}
	}

	// 默认返回无错误
	return &GrammarCheckResponse{
		HasError:    false,
		Suggestion:  "",
		Explanation: "",
	}
}

// 通知用户语法错误
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
