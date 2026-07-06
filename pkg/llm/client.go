// Package llm 提供 OpenAI 兼容的 LLM 调用客户端。
// 默认对接 DeepSeek，也可替换为其他兼容服务（OpenAI / 通义千问 / Moonshot 等）。
// 通过环境变量 LLM_API_KEY 和 LLM_BASE_URL 配置。
package llm

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"
)

// ──────────── OpenAI 兼容请求/响应结构体 ────────────

type message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type requestBody struct {
	Model    string    `json:"model"`
	Messages []message `json:"messages"`
}

type choice struct {
	Message message `json:"message"`
}

type responseBody struct {
	Choices []choice `json:"choices"`
	Error   *struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

// ──────────── Client ────────────

// Client 封装一次 LLM 调用的配置。
type Client struct {
	apiKey       string
	baseURL      string
	model        string
	systemPrompt string
	http         *http.Client
}

// New 创建 LLM 客户端。
//
// 环境变量（优先级从高到低）：
//
//	LLM_API_KEY   — API 密钥，默认 ""
//	LLM_BASE_URL  — 服务地址，默认 https://api.deepseek.com/v1
//	LLM_MODEL     — 模型名，默认 deepseek-chat
func New() *Client {
	key := os.Getenv("LLM_API_KEY")
	base := os.Getenv("LLM_BASE_URL")
	if base == "" {
		base = "https://api.deepseek.com/v1"
	}
	model := os.Getenv("LLM_MODEL")
	if model == "" {
		model = "deepseek-chat"
	}

	return &Client{
		apiKey:  key,
		baseURL: base,
		model:   model,
		http:    &http.Client{Timeout: 15 * time.Second},
	}
}

// Enabled 返回是否配置了 API Key（即可用）。
func (c *Client) Enabled() bool { return c.apiKey != "" }

// SetSystemPrompt 设置系统提示词，优先级高于默认。
func (c *Client) SetSystemPrompt(p string) { c.systemPrompt = p }

// ChatCompletion 发送一次对话请求，返回助手回复文本。
// 若已调用 SetSystemPrompt 则使用自定义提示词，否则使用默认。
func (c *Client) ChatCompletion(userPrompt string) (string, error) {
	if c.apiKey == "" {
		return "", fmt.Errorf("LLM_API_KEY 未配置")
	}
	sp := c.systemPrompt
	if sp == "" {
		sp = SystemDefault
	}

	messages := []message{
		{Role: "system", Content: sp},
		{Role: "user", Content: userPrompt},
	}

	body := requestBody{Model: c.model, Messages: messages}
	payload, _ := json.Marshal(body)

	req, err := http.NewRequest("POST", c.baseURL+"/chat/completions", bytes.NewReader(payload))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.http.Do(req)
	if err != nil {
		return "", fmt.Errorf("LLM 请求失败: %w", err)
	}
	defer resp.Body.Close()

	data, _ := io.ReadAll(resp.Body)

	var result responseBody
	if err := json.Unmarshal(data, &result); err != nil {
		return "", fmt.Errorf("解析 LLM 响应失败: %w", err)
	}

	if result.Error != nil {
		return "", fmt.Errorf("LLM 返回错误: %s", result.Error.Message)
	}

	if len(result.Choices) == 0 {
		return "", fmt.Errorf("LLM 返回空 choices")
	}

	content := result.Choices[0].Message.Content
	log.Printf("[llm] 调用成功 model=%s len=%d", c.model, len(content))
	return content, nil
}

// ──────────── 系统提示词 ────────────

// SystemDefault 默认 system prompt（未指定时使用）。
const SystemDefault = `你是小红书的智能助手，请用中文简洁友好地回答用户问题。`

