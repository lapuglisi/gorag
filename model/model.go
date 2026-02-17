package gorag_model

type LlamaEmbedRequest struct {
	Input string `json:"input"`
}

type LlamaEmbedResponse struct {
	Model  string `json:"model"`
	Object string `json:"object"`
	Usage  struct {
		PromptTokens int
		TotalTokens  int
	} `json:"usage"`
	Data []struct {
		Embedding []float32 `json:"embedding"`
		Index     int       `json:"index"`
		Object    string    `json:"object"`
	} `json:"data"`
}

type LlamaCompleteRequest struct {
}

type LlamaCompleteResponse struct {
}
