// Package llm provides LLM client implementations including Google Gemini support.
package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

// GeminiClient provides Google Gemini API access.
type GeminiClient struct {
	apiKey     string
	model      string
	httpClient *http.Client
	baseURL    string
}

// GeminiConfig holds Gemini client configuration.
type GeminiConfig struct {
	APIKey     string
	Model      string
	Timeout    time.Duration
	HTTPSProxy string
}

// NewGeminiClient creates a new Gemini client.
func NewGeminiClient(cfg GeminiConfig) *GeminiClient {
	if cfg.Timeout == 0 {
		cfg.Timeout = 60 * time.Second
	}
	if cfg.Model == "" {
		cfg.Model = "gemini-2.0-flash"
	}

	httpClient := &http.Client{
		Timeout: cfg.Timeout,
	}

	if cfg.HTTPSProxy != "" {
		if u, err := url.Parse(cfg.HTTPSProxy); err == nil {
			httpClient.Transport = &http.Transport{
				Proxy: http.ProxyURL(u),
			}
		}
	}

	return &GeminiClient{
		apiKey:     cfg.APIKey,
		model:      cfg.Model,
		httpClient: httpClient,
		baseURL:    "https://generativelanguage.googleapis.com/v1beta",
	}
}

// GeminiContent represents content in Gemini format.
type GeminiContent struct {
	Parts []GeminiPart `json:"parts"`
	Role  string       `json:"role,omitempty"`
}

// GeminiPart represents a part of content.
type GeminiPart struct {
	Text         string            `json:"text,omitempty"`
	FunctionCall *GeminiFuncCall   `json:"functionCall,omitempty"`
	FunctionResp *GeminiFuncResult `json:"functionResponse,omitempty"`
}

// GeminiFuncCall represents a function call.
type GeminiFuncCall struct {
	Name string                 `json:"name"`
	Args map[string]interface{} `json:"args"`
}

// GeminiFuncResult represents a function result.
type GeminiFuncResult struct {
	Name     string                 `json:"name"`
	Response map[string]interface{} `json:"response"`
}

// GeminiTool represents a tool definition.
type GeminiTool struct {
	FunctionDeclarations []GeminiFunctionDecl `json:"functionDeclarations,omitempty"`
}

// GeminiFunctionDecl represents a function declaration.
type GeminiFunctionDecl struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Parameters  json.RawMessage `json:"parameters,omitempty"`
}

// GeminiRequest is the request payload for Gemini API.
type GeminiRequest struct {
	Contents         []GeminiContent       `json:"contents"`
	Tools            []GeminiTool          `json:"tools,omitempty"`
	SystemInstruct   *GeminiContent        `json:"systemInstruction,omitempty"`
	GenerationConfig *GeminiGenerationCfg  `json:"generationConfig,omitempty"`
	SafetySettings   []GeminiSafetySetting `json:"safetySettings,omitempty"`
}

// GeminiGenerationCfg holds generation parameters.
type GeminiGenerationCfg struct {
	Temperature     float64 `json:"temperature,omitempty"`
	TopP            float64 `json:"topP,omitempty"`
	TopK            int     `json:"topK,omitempty"`
	MaxOutputTokens int     `json:"maxOutputTokens,omitempty"`
}

// GeminiSafetySetting configures safety filtering.
type GeminiSafetySetting struct {
	Category  string `json:"category"`
	Threshold string `json:"threshold"`
}

// GeminiResponse is the response from Gemini API.
type GeminiResponse struct {
	Candidates []struct {
		Content       GeminiContent `json:"content"`
		FinishReason  string        `json:"finishReason"`
		SafetyRatings []struct {
			Category    string `json:"category"`
			Probability string `json:"probability"`
		} `json:"safetyRatings"`
	} `json:"candidates"`
	UsageMetadata struct {
		PromptTokenCount     int `json:"promptTokenCount"`
		CandidatesTokenCount int `json:"candidatesTokenCount"`
		TotalTokenCount      int `json:"totalTokenCount"`
	} `json:"usageMetadata"`
}

// Chat sends a chat request to Gemini.
func (c *GeminiClient) Chat(ctx context.Context, messages []Message, tools []Tool) (*ChatResponse, error) {
	// Convert messages to Gemini format
	var contents []GeminiContent
	var systemContent *GeminiContent

	for _, msg := range messages {
		if msg.Role == "system" {
			systemContent = &GeminiContent{
				Parts: []GeminiPart{{Text: msg.Content}},
			}
			continue
		}

		role := "user"
		if msg.Role == "assistant" {
			role = "model"
		}

		content := GeminiContent{
			Role:  role,
			Parts: []GeminiPart{{Text: msg.Content}},
		}

		// Handle tool calls in assistant messages
		if len(msg.ToolCalls) > 0 {
			content.Parts = nil
			for _, tc := range msg.ToolCalls {
				var args map[string]interface{}
				json.Unmarshal([]byte(tc.Function.Arguments), &args)
				content.Parts = append(content.Parts, GeminiPart{
					FunctionCall: &GeminiFuncCall{
						Name: tc.Function.Name,
						Args: args,
					},
				})
			}
		}

		// Handle tool response
		if msg.ToolCallID != "" {
			content.Parts = []GeminiPart{{
				FunctionResp: &GeminiFuncResult{
					Name:     msg.ToolCallID,
					Response: map[string]interface{}{"result": msg.Content},
				},
			}}
		}

		contents = append(contents, content)
	}

	// Convert tools to Gemini format
	var geminiTools []GeminiTool
	if len(tools) > 0 {
		var funcDecls []GeminiFunctionDecl
		for _, tool := range tools {
			funcDecls = append(funcDecls, GeminiFunctionDecl{
				Name:        tool.Function.Name,
				Description: tool.Function.Description,
				Parameters:  tool.Function.Parameters,
			})
		}
		geminiTools = []GeminiTool{{FunctionDeclarations: funcDecls}}
	}

	req := GeminiRequest{
		Contents:       contents,
		Tools:          geminiTools,
		SystemInstruct: systemContent,
		GenerationConfig: &GeminiGenerationCfg{
			Temperature:     0.7,
			MaxOutputTokens: 4096,
		},
	}

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	url := fmt.Sprintf("%s/models/%s:generateContent?key=%s", c.baseURL, c.model, c.apiKey)
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("http request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API error %d: %s", resp.StatusCode, string(respBody))
	}

	var geminiResp GeminiResponse
	if err := json.Unmarshal(respBody, &geminiResp); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	// Convert Gemini response to standard format
	return c.convertResponse(geminiResp)
}

// convertResponse converts Gemini response to standard ChatResponse.
func (c *GeminiClient) convertResponse(resp GeminiResponse) (*ChatResponse, error) {
	chatResp := &ChatResponse{
		Usage: struct {
			PromptTokens     int `json:"prompt_tokens"`
			CompletionTokens int `json:"completion_tokens"`
			TotalTokens      int `json:"total_tokens"`
		}{
			PromptTokens:     resp.UsageMetadata.PromptTokenCount,
			CompletionTokens: resp.UsageMetadata.CandidatesTokenCount,
			TotalTokens:      resp.UsageMetadata.TotalTokenCount,
		},
	}

	if len(resp.Candidates) == 0 {
		return nil, fmt.Errorf("no candidates in response")
	}

	candidate := resp.Candidates[0]
	msg := Message{Role: "assistant"}

	for _, part := range candidate.Content.Parts {
		if part.Text != "" {
			msg.Content = part.Text
		}
		if part.FunctionCall != nil {
			argsJSON, _ := json.Marshal(part.FunctionCall.Args)
			msg.ToolCalls = append(msg.ToolCalls, ToolCall{
				ID:   part.FunctionCall.Name,
				Type: "function",
				Function: FunctionCall{
					Name:      part.FunctionCall.Name,
					Arguments: string(argsJSON),
				},
			})
		}
	}

	chatResp.Choices = append(chatResp.Choices, struct {
		Index        int     `json:"index"`
		Message      Message `json:"message"`
		FinishReason string  `json:"finish_reason"`
	}{
		Index:        0,
		Message:      msg,
		FinishReason: candidate.FinishReason,
	})

	return chatResp, nil
}

// SimpleChat is a convenience method for text-only chat with Gemini.
func (c *GeminiClient) SimpleChat(ctx context.Context, systemPrompt, userMessage string) (string, error) {
	messages := []Message{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: userMessage},
	}

	resp, err := c.Chat(ctx, messages, nil)
	if err != nil {
		return "", err
	}

	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("no response choices")
	}

	return resp.Choices[0].Message.Content, nil
}

// Model returns the model name.
func (c *GeminiClient) Model() string {
	return c.model
}
