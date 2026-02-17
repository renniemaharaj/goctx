package google

import (
	"context"
	"fmt"
	"sync"

	"google.golang.org/genai"
)

type KeyInfo struct {
	Key   string
	Model string
}

type Manager struct {
	keys chan KeyInfo
	mu   sync.Mutex
}

func NewManager(initialKeys map[string]string) *Manager {
	m := &Manager{
		keys: make(chan KeyInfo, 100),
	}
	for k, v := range initialKeys {
		m.keys <- KeyInfo{Key: k, Model: v}
	}
	return m
}

func (m *Manager) SetKeys(keys map[string]string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Drain current channel
	for len(m.keys) > 0 {
		<-m.keys
	}

	for k, v := range keys {
		m.keys <- KeyInfo{Key: k, Model: v}
	}
}

func (m *Manager) Generate(ctx context.Context, system, prompt string) (string, error) {
	info := <-m.keys
	defer func() { m.keys <- info }()

	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		APIKey:  info.Key,
		Backend: genai.BackendGeminiAPI,
	})
	if err != nil {
		return "", err
	}

	resp, err := client.Models.GenerateContent(ctx, info.Model, genai.Text(prompt), &genai.GenerateContentConfig{
		SystemInstruction: &genai.Content{Parts: []*genai.Part{{Text: system}}},
	})

	if err != nil {
		return "", err
	}

	if len(resp.Candidates) == 0 || len(resp.Candidates[0].Content.Parts) == 0 {
		return "", fmt.Errorf("empty response from AI")
	}

	return resp.Candidates[0].Content.Parts[0].Text, nil
}
