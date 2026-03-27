package ollama

import (
"bytes"
"encoding/json"
"fmt"
"io"
"net/http"
"os"
"time"
)

var (
Host    = envOr("OLLAMA_HOST", "http://localhost:11434")
Model   = envOr("OLLAMA_MODEL", "qwen3:1.7b")
CtxSize = 4096
)

type ChatMessage struct {
Role    string `json:"role"`
Content string `json:"content"`
}

type ChatRequest struct {
Model    string        `json:"model"`
Messages []ChatMessage `json:"messages"`
Stream   bool          `json:"stream"`
Options  Options       `json:"options"`
}

type Options struct {
NumCtx      int     `json:"num_ctx"`
Temperature float64 `json:"temperature"`
}

type ChatResponse struct {
Message       ChatMessage `json:"message"`
Model         string      `json:"model"`
TotalDuration int64       `json:"total_duration"`
PromptEval    int         `json:"prompt_eval_count"`
EvalCount     int         `json:"eval_count"`
}

type GenerateRequest struct {
Model   string  `json:"model"`
Prompt  string  `json:"prompt"`
System  string  `json:"system,omitempty"`
Stream  bool    `json:"stream"`
Options Options `json:"options"`
}

type GenerateResponse struct {
Response      string `json:"response"`
Model         string `json:"model"`
TotalDuration int64  `json:"total_duration"`
PromptEval    int    `json:"prompt_eval_count"`
EvalCount     int    `json:"eval_count"`
}

func Chat(messages []ChatMessage, model string) (*ChatResponse, error) {
if model == "" {
model = Model
}

req := ChatRequest{
Model:    model,
Messages: messages,
Stream:   false,
Options:  Options{NumCtx: CtxSize, Temperature: 0.3},
}

body, err := json.Marshal(req)
if err != nil {
return nil, fmt.Errorf("marshal chat request: %w", err)
}

client := &http.Client{Timeout: 5 * time.Minute}
resp, err := client.Post(Host+"/api/chat", "application/json", bytes.NewReader(body))
if err != nil {
return nil, fmt.Errorf("ollama chat: %w", err)
}
defer resp.Body.Close()

if resp.StatusCode != http.StatusOK {
b, _ := io.ReadAll(resp.Body)
return nil, fmt.Errorf("ollama %d: %s", resp.StatusCode, string(b))
}

var result ChatResponse
if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
return nil, fmt.Errorf("decode chat response: %w", err)
}
return &result, nil
}

func Generate(prompt, system, model string) (*GenerateResponse, error) {
if model == "" {
model = Model
}

req := GenerateRequest{
Model:   model,
Prompt:  prompt,
System:  system,
Stream:  false,
Options: Options{NumCtx: CtxSize, Temperature: 0.3},
}

body, err := json.Marshal(req)
if err != nil {
return nil, fmt.Errorf("marshal generate request: %w", err)
}

client := &http.Client{Timeout: 5 * time.Minute}
resp, err := client.Post(Host+"/api/generate", "application/json", bytes.NewReader(body))
if err != nil {
return nil, fmt.Errorf("ollama generate: %w", err)
}
defer resp.Body.Close()

if resp.StatusCode != http.StatusOK {
b, _ := io.ReadAll(resp.Body)
return nil, fmt.Errorf("ollama %d: %s", resp.StatusCode, string(b))
}

var result GenerateResponse
if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
return nil, fmt.Errorf("decode generate response: %w", err)
}
return &result, nil
}

func IsRunning() bool {
client := &http.Client{Timeout: 2 * time.Second}
resp, err := client.Get(Host + "/api/tags")
if err != nil {
return false
}
resp.Body.Close()
return resp.StatusCode == http.StatusOK
}

func ListModels() ([]string, error) {
client := &http.Client{Timeout: 5 * time.Second}
resp, err := client.Get(Host + "/api/tags")
if err != nil {
return nil, err
}
defer resp.Body.Close()

var result struct {
Models []struct {
Name string `json:"name"`
} `json:"models"`
}
if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
return nil, err
}

names := make([]string, len(result.Models))
for i, m := range result.Models {
names[i] = m.Name
}
return names, nil
}

func envOr(key, fallback string) string {
if v := os.Getenv(key); v != "" {
return v
}
return fallback
}
