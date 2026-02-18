package model

type Scripts struct {
	Test  string `json:"test,omitempty"`
	Build string `json:"build,omitempty"`
}

type Config struct {
	Ignore     []string `json:"ignore"`
	Extensions []string `json:"extensions"`
	Scripts    Scripts  `json:"scripts"`
}

type ProjectOutput struct {
	ShortDescription string            `json:"short_description,omitempty"`
	ProjectTree      string            `json:"project_tree"`
	Files            map[string]string `json:"files"`
	FileCount        int               `json:"file_count"`
	TokenCount       int               `json:"token_count"`
	DirCount         int               `json:"dir_count"`
}
