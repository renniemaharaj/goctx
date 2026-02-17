package google

import (
	"context"
	"log"

	"google.golang.org/genai"
)

// var model = flag.String("model", "gemini-3-flash-preview", "the model name, e.g. gemini-3-flash-preview")

func newClient(ctx context.Context, key string) (*genai.Client, error) {
	return genai.NewClient(ctx, &genai.ClientConfig{
		Backend: genai.BackendGeminiAPI,
		APIKey:  key,
	})
}

func tools() []*genai.Tool {
	return []*genai.Tool{
		{
			GoogleSearch: &genai.GoogleSearch{},
		},
	}
}

func contentConfig(sysInstructions string) *genai.GenerateContentConfig {
	return &genai.GenerateContentConfig{
		Tools: tools(),
		SystemInstruction: &genai.Content{
			Parts: []*genai.Part{
				&genai.Part{Text: sysInstructions},
			},
		},
		ThinkingConfig: &genai.ThinkingConfig{
			ThinkingLevel: genai.ThinkingLevelHigh,
		},
	}
}

func run(ctx context.Context, sysInstructions, prompt string) string {
	key := BorrowOneGoogleApiKey()
	defer FreeOneGoogleApiKey(key)

	client, err := newClient(ctx, key.key)
	if err != nil {
		log.Fatal(err)
	}

	var config *genai.GenerateContentConfig = contentConfig(sysInstructions)
	var contents = []*genai.Content{
		&genai.Content{
			Role: "user",
			Parts: []*genai.Part{
				&genai.Part{
					Text: "INSERT_INPUT_HERE",
				},
			},
		},
	}

	// Call the GenerateContent method.
	result, err := client.Models.GenerateContent(ctx, key.model, contents, config)
	if err != nil {
		log.Fatal(err)
	}

	return result.Text()
}
