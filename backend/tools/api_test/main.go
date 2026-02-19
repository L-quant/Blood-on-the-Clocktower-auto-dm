// API comparison test tool for Blood on the Clocktower Auto-DM
// Tests both Gemini and DeepSeek APIs for function calling capability
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

// ===== Gemini Test =====

type GeminiRequest struct {
	Contents         []GeminiContent       `json:"contents"`
	Tools            []GeminiTool          `json:"tools,omitempty"`
	SystemInstruct   *GeminiContent        `json:"systemInstruction,omitempty"`
	GenerationConfig *GeminiGenConfig      `json:"generationConfig,omitempty"`
	SafetySettings   []GeminiSafetySetting `json:"safetySettings,omitempty"`
}

type GeminiContent struct {
	Parts []GeminiPart `json:"parts"`
	Role  string       `json:"role,omitempty"`
}

type GeminiPart struct {
	Text         string            `json:"text,omitempty"`
	FunctionCall *GeminiFuncCall   `json:"functionCall,omitempty"`
	FunctionResp *GeminiFuncResult `json:"functionResponse,omitempty"`
}

type GeminiFuncCall struct {
	Name string                 `json:"name"`
	Args map[string]interface{} `json:"args"`
}

type GeminiFuncResult struct {
	Name     string                 `json:"name"`
	Response map[string]interface{} `json:"response"`
}

type GeminiTool struct {
	FunctionDeclarations []GeminiFuncDecl `json:"functionDeclarations,omitempty"`
}

type GeminiFuncDecl struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Parameters  json.RawMessage `json:"parameters,omitempty"`
}

type GeminiGenConfig struct {
	Temperature     float64 `json:"temperature,omitempty"`
	MaxOutputTokens int     `json:"maxOutputTokens,omitempty"`
}

type GeminiSafetySetting struct {
	Category  string `json:"category"`
	Threshold string `json:"threshold"`
}

func testGemini(apiKey, model string) {
	fmt.Printf("\n========== Testing Gemini: %s ==========\n", model)

	tools := []GeminiTool{{
		FunctionDeclarations: []GeminiFuncDecl{
			{
				Name:        "send_public_message",
				Description: "Send a public message to all players in the game room",
				Parameters:  json.RawMessage(`{"type":"object","properties":{"message":{"type":"string","description":"The message to send"}},"required":["message"]}`),
			},
			{
				Name:        "advance_phase",
				Description: "Advance the game to the next phase (e.g., from night to day)",
				Parameters:  json.RawMessage(`{"type":"object","properties":{"phase":{"type":"string","enum":["night","day","nomination","vote"],"description":"The phase to advance to"}},"required":["phase"]}`),
			},
			{
				Name:        "send_whisper",
				Description: "Send a private message to a specific player",
				Parameters:  json.RawMessage(`{"type":"object","properties":{"to_user_id":{"type":"string","description":"Target player ID"},"message":{"type":"string","description":"Private message content"}},"required":["to_user_id","message"]}`),
			},
		},
	}}

	req := GeminiRequest{
		Contents: []GeminiContent{{
			Role:  "user",
			Parts: []GeminiPart{{Text: "The night phase has ended. 3 players are alive: Alice (seat 1), Bob (seat 2), Charlie (seat 3). The Imp killed Bob last night. Announce the dawn, reveal who died, and advance to day phase. Keep narration atmospheric but brief (2-3 sentences in Chinese)."}},
		}},
		Tools: tools,
		SystemInstruct: &GeminiContent{
			Parts: []GeminiPart{{Text: "You are an expert Blood on the Clocktower Storyteller. You manage game flow using tools. Always use send_public_message to communicate with players and advance_phase to transition game phases. Respond in Chinese."}},
		},
		GenerationConfig: &GeminiGenConfig{
			Temperature:     0.7,
			MaxOutputTokens: 4096,
		},
	}

	body, _ := json.Marshal(req)
	url := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent?key=%s", model, apiKey)

	start := time.Now()
	httpReq, _ := http.NewRequest("POST", url, bytes.NewReader(body))
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(httpReq)
	latency := time.Since(start)

	if err != nil {
		fmt.Printf("ERROR: %v\n", err)
		return
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	fmt.Printf("Status: %d\n", resp.StatusCode)
	fmt.Printf("Latency: %v\n", latency)

	if resp.StatusCode != 200 {
		fmt.Printf("Error body: %s\n", string(respBody))
		return
	}

	// Parse and display results
	var result map[string]interface{}
	json.Unmarshal(respBody, &result)

	prettyJSON, _ := json.MarshalIndent(result, "", "  ")
	fmt.Printf("Response:\n%s\n", string(prettyJSON))

	// Check for function calls
	if candidates, ok := result["candidates"].([]interface{}); ok && len(candidates) > 0 {
		candidate := candidates[0].(map[string]interface{})
		if content, ok := candidate["content"].(map[string]interface{}); ok {
			if parts, ok := content["parts"].([]interface{}); ok {
				hasToolCall := false
				for _, part := range parts {
					p := part.(map[string]interface{})
					if _, ok := p["functionCall"]; ok {
						hasToolCall = true
					}
				}
				if hasToolCall {
					fmt.Println("\n✅ Function calling: SUPPORTED - Model returned tool calls!")
				} else {
					fmt.Println("\n⚠️ Function calling: Model responded with text only (no tool calls)")
				}
			}
		}
	}

	// Show usage
	if usage, ok := result["usageMetadata"].(map[string]interface{}); ok {
		fmt.Printf("\nTokens - Prompt: %.0f, Response: %.0f, Total: %.0f\n",
			usage["promptTokenCount"], usage["candidatesTokenCount"], usage["totalTokenCount"])
	}
}

// ===== DeepSeek Test =====

type DSRequest struct {
	Model    string      `json:"model"`
	Messages []DSMessage `json:"messages"`
	Tools    []DSTool    `json:"tools,omitempty"`
}

type DSMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type DSTool struct {
	Type     string     `json:"type"`
	Function DSFunction `json:"function"`
}

type DSFunction struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Parameters  json.RawMessage `json:"parameters"`
}

func testDeepSeek(apiKey, model string) {
	fmt.Printf("\n========== Testing DeepSeek: %s ==========\n", model)

	tools := []DSTool{
		{
			Type: "function",
			Function: DSFunction{
				Name:        "send_public_message",
				Description: "Send a public message to all players in the game room",
				Parameters:  json.RawMessage(`{"type":"object","properties":{"message":{"type":"string","description":"The message to send"}},"required":["message"]}`),
			},
		},
		{
			Type: "function",
			Function: DSFunction{
				Name:        "advance_phase",
				Description: "Advance the game to the next phase",
				Parameters:  json.RawMessage(`{"type":"object","properties":{"phase":{"type":"string","enum":["night","day","nomination","vote"],"description":"The phase to advance to"}},"required":["phase"]}`),
			},
		},
		{
			Type: "function",
			Function: DSFunction{
				Name:        "send_whisper",
				Description: "Send a private message to a specific player",
				Parameters:  json.RawMessage(`{"type":"object","properties":{"to_user_id":{"type":"string","description":"Target player ID"},"message":{"type":"string","description":"Private message content"}},"required":["to_user_id","message"]}`),
			},
		},
	}

	req := DSRequest{
		Model: model,
		Messages: []DSMessage{
			{Role: "system", Content: "You are an expert Blood on the Clocktower Storyteller. You manage game flow using tools. Always use send_public_message to communicate with players and advance_phase to transition game phases. Respond in Chinese."},
			{Role: "user", Content: "The night phase has ended. 3 players are alive: Alice (seat 1), Bob (seat 2), Charlie (seat 3). The Imp killed Bob last night. Announce the dawn, reveal who died, and advance to day phase. Keep narration atmospheric but brief (2-3 sentences in Chinese)."},
		},
		Tools: tools,
	}

	body, _ := json.Marshal(req)

	start := time.Now()
	httpReq, _ := http.NewRequest("POST", "https://api.deepseek.com/chat/completions", bytes.NewReader(body))
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+apiKey)

	client := &http.Client{Timeout: 120 * time.Second}
	resp, err := client.Do(httpReq)
	latency := time.Since(start)

	if err != nil {
		fmt.Printf("ERROR: %v\n", err)
		return
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	fmt.Printf("Status: %d\n", resp.StatusCode)
	fmt.Printf("Latency: %v\n", latency)

	if resp.StatusCode != 200 {
		fmt.Printf("Error body: %s\n", string(respBody))
		return
	}

	var result map[string]interface{}
	json.Unmarshal(respBody, &result)

	prettyJSON, _ := json.MarshalIndent(result, "", "  ")
	fmt.Printf("Response:\n%s\n", string(prettyJSON))

	// Check for tool calls
	if choices, ok := result["choices"].([]interface{}); ok && len(choices) > 0 {
		choice := choices[0].(map[string]interface{})
		if msg, ok := choice["message"].(map[string]interface{}); ok {
			if toolCalls, ok := msg["tool_calls"].([]interface{}); ok && len(toolCalls) > 0 {
				fmt.Println("\n✅ Function calling: SUPPORTED - Model returned tool calls!")
			} else if content, ok := msg["content"].(string); ok && content != "" {
				fmt.Println("\n⚠️ Function calling: Model responded with text only (no tool calls)")
			}
		}
	}

	// Show usage
	if usage, ok := result["usage"].(map[string]interface{}); ok {
		fmt.Printf("\nTokens - Prompt: %.0f, Response: %.0f, Total: %.0f\n",
			usage["prompt_tokens"], usage["completion_tokens"], usage["total_tokens"])
	}
}

func main() {
	geminiKey := os.Getenv("GEMINI_API_KEY")
	deepseekKey := os.Getenv("DEEPSEEK_API_KEY")

	if geminiKey == "" {
		geminiKey = "AIzaSyDBPLTIbQGSIwjcJyanid5xNl7jLjCFvLs"
	}
	if deepseekKey == "" {
		deepseekKey = "sk-361c5a8aec9143bbb49101be8b78738f"
	}

	fmt.Println("=== Blood on the Clocktower Auto-DM API Comparison Test ===")
	fmt.Println("Testing function calling with a realistic game scenario...")

	// Test Gemini 3 Flash Preview (FREE)
	testGemini(geminiKey, "gemini-3-flash-preview")

	// Test DeepSeek V3
	testDeepSeek(deepseekKey, "deepseek-chat")

	fmt.Println("\n========== COMPARISON SUMMARY ==========")
	fmt.Println("See results above to compare:")
	fmt.Println("1. Function calling support (critical for game control)")
	fmt.Println("2. Response latency")
	fmt.Println("3. Response quality (narration + tool usage)")
	fmt.Println("4. Token usage efficiency")
}
