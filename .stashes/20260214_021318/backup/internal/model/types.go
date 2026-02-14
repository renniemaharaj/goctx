package model

type Config struct {
	Ignore     []string `json:"ignore"` 
	Extensions []string `json:"extensions"` 
}

type ProjectOutput struct {
	InstructionHeader string            `json:"instruction_header,omitempty"` 
	ShortDescription  string            `json:"short_description,omitempty"` 
	EstimatedTokens   int               `json:"estimated_tokens"` 
	ProjectTree       string            `json:"project_tree"` 
	Files             map[string]string `json:"files"` 
}