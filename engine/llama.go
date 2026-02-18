package gorag_engine

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
)

// JSON structures for API requests
type llamaEmbedRequest struct {
	Input string `json:"input"`
}

type llamaEmbedResponse struct {
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

// LlamaEngine: The main engine for Llama operations
type LlamaEngine struct {
	LlamaServer string
	EmbedServer string
}

// NewLlamaEngine
func NewLlamaEngine(es string, ls string) (e *LlamaEngine) {
	return &LlamaEngine{
		EmbedServer: es,
		LlamaServer: ls,
	}
}

// GetEmbeddings
func (l *LlamaEngine) GetEmbeddings(input string) (embeds [][]float32, err error) {
	var llama_resp llamaEmbedResponse
	var client = &http.Client{}

	var json_request llamaEmbedRequest = llamaEmbedRequest{
		Input: input,
	}

	json_bytes, err := json.Marshal(json_request)
	if err != nil {
		return nil, err
	}

	payload := bytes.NewBuffer(json_bytes)

	// Prepare the http.Request struct
	url := fmt.Sprintf("%s/v1/embeddings", l.EmbedServer)
	req, err := http.NewRequest(http.MethodPost, url, payload)

	if err != nil {
		return nil, err
	}

	req.Header.Add("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	resp.Body.Close()

	if err = json.Unmarshal(body, &llama_resp); err != nil {
		return nil, err
	}

	if dataLen := len(llama_resp.Data); dataLen > 0 {
		embeds = make([][]float32, dataLen)

		for i, data := range llama_resp.Data {
			embeds[i] = make([]float32, 1)
			embeds[i] = data.Embedding
		}
	}

	log.Printf("final embeds: %v\n", embeds)

	return embeds, nil
}
