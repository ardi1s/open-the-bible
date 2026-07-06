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
	apiKey  string
	baseURL string
	model   string
	http    *http.Client
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

// ChatCompletion 发送一次对话请求，返回助手回复文本。
// 自动注入 "你是一个小红书的运营分析师..." 的 system prompt。
func (c *Client) ChatCompletion(prompt string) (string, error) {
	if c.apiKey == "" {
		return "", fmt.Errorf("LLM_API_KEY 未配置")
	}

	messages := []message{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: prompt},
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

// ──────────── System Prompt ────────────

const systemPrompt = `你是小红书的运营分析师 Agent，负责根据用户数据生成个性化的运营建议。

输出要求：
- 用中文，每条建议一行，每条以 emoji 开头
- 不要客套话，直接给建议
- 控制在 5 条以内
- 不要输出 markdown 格式

分析维度：
1. 图片数量是否充足（≥3 张为佳）
2. 标签是否热门、是否缺失
3. 发布频率是否健康（建议每周 2-3 篇）
4. 内容风格建议
5. 互动引导技巧`
