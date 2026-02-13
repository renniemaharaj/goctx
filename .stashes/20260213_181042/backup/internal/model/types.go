package model

type Config struct {
	Ignore     []string `json:"ignore"`
	Extensions []string `json:"extensions"`
}

type ProjectOutput struct {
	EstimatedTokens int               `json:"estimated_tokens"`
	ProjectTree     string            `json:"project_tree"`
	Files           map[string]string `json:"files"`
}
