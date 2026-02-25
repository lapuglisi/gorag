package gorag_engine

import (
	"bufio"
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

type llamaCompletionMessages struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type llamaCompletionRequest struct {
	Messages    []llamaCompletionMessages `json:"messages"`
	Stream      bool                      `json:"stream"`
	Temperature float32                   `json:"temperature"`
}

type llamaEmbeddings struct {
	Model      string      `json:"model"`
	Embeddings [][]float32 `json:"embeddings"`
}

type LlamaCompletionCallback func(data string) error

type LlamaCompletionStream struct {
	Choices []struct {
		FinishReason string `json:"finish_reason"`
		Index        int    `json:"index"`
		Delta        struct {
			Content string `json:"content"`
		} `json:"delta"`
	}
	Created           int64  `json:"created"`
	Id                string `json:"id"`
	Model             string `json:"model"`
	SystemFingerprint string `json:"system_fingerprint"`
	Object            string `json:"object"`
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
func (l *LlamaEngine) GetEmbeddings(input string) (embeds *llamaEmbeddings, err error) {
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

	embeds = &llamaEmbeddings{}
	embeds.Model = llama_resp.Model

	if dataLen := len(llama_resp.Data); dataLen > 0 {
		embeds.Embeddings = make([][]float32, dataLen)

		for i, data := range llama_resp.Data {
			embeds.Embeddings[i] = make([]float32, 0)
			embeds.Embeddings[i] = append(embeds.Embeddings[i], data.Embedding...)
		}
	}

	log.Printf("final embeds: %v\n", embeds)

	return embeds, nil
}

func (l *LlamaEngine) GetCompletions(
	data llamaCompletionRequest,
	callback LlamaCompletionCallback) (err error) {
	var client *http.Client = &http.Client{}

	jsonBytes, err := json.Marshal(data)
	if err != nil {
		return err
	}

	payload := bytes.NewBuffer(jsonBytes)

	var uri string = fmt.Sprintf("%s/v1/chat/completions", l.LlamaServer)
	req, err := http.NewRequest(http.MethodPost, uri, payload)

	if err != nil {
		return err
	}

	req.Header.Add("Accept", "text/event-stream")

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	log.Println("LlamaEngine::GetCompletions:", data)

	err = nil

	reader := bufio.NewReader(resp.Body)
	for {
		stream, err := reader.ReadString('\n')
		if err != nil {
			log.Printf("resp read: %s\n", err.Error())
			break
		}

		if err = callback(stream); err != nil {
			log.Printf("GetCompletions: callback error: %s\n", err.Error())
			break
		}
	}

	return err
}
