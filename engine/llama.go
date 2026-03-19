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

// Constants
const (
	LlamaRoleUser      string = "user"
	LlamaRoleSystem    string = "system"
	LlamaRoleAssistant string = "assistant"

	// Constants for llama request
	LlamaDefaultTemperature float32 = 0.2
	LlamaDefaultTopK        int     = 40
	LlamaDefaultTopP        float32 = 0.9
	LlamaDefaultNPredict    int     = 512
	LlamaMirostatMode       int     = 1
	GoRagMirostatTau        float32 = 5.0
	GoRagMirostatEta        float32 = 0.5
	GoRagMaxTokens                  = 2048

	LlamaRagSystemPrompt string = "You are a very helpfull asssistant expert in answering " +
		"questions in a RAG pipeline when provided contexts.\n" +
		"Make sure to answer the question in the original language."

	LlamaRagAssistantPrompt string = "Answer the user query using the provided context." +
		"Use as much information from the context as possible." +
		"If you cannot find an answer with the context, simply state that you don't know."
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

type llamaCompletionMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type llamaCompletionRequest struct {
	Messages    []llamaCompletionMessage `json:"messages"`
	TopK        int                      `json:"top_k,omitempty"`
	TopP        float32                  `json:"top_p,omitempty"`
	N_keep      int                      `json:"n_keep,omitempty"`
	N_predict   int                      `json:"n_predict,omitempty"`
	CachePrompt bool                     `json:"cache_prompt,omitempty"`
	Stream      bool                     `json:"stream"`
	Temperature float32                  `json:"temperature"`
	MaxTokens   int                      `json:"max_tokens,omitempty"`
	TypicalP    float32                  `json:"typical_p,omitempty"`
	Mirostat    int                      `json:"mirostat,omitempty"`
	MirostatTau float32                  `json:"mirostat_tau,omitempty"`
	MirostatEta float32                  `json:"mirostat_eta,omitempty"`
}

func NewCompletionRequest() *llamaCompletionRequest {
	return &llamaCompletionRequest{
		Messages:    make([]llamaCompletionMessage, 0),
		Stream:      true,
		N_keep:      -1,
		Temperature: LlamaDefaultTemperature,
		TopK:        LlamaDefaultTopK,
		TopP:        LlamaDefaultTopP,
		N_predict:   LlamaDefaultNPredict,
		CachePrompt: true,
		MaxTokens:   GoRagMaxTokens,
		Mirostat:    LlamaMirostatMode,
		MirostatTau: GoRagMirostatTau,
		MirostatEta: GoRagMirostatEta,
	}
}

func (l *llamaCompletionRequest) WithTemperature(temp float32) *llamaCompletionRequest {
	if temp >= 0 {
		l.Temperature = temp
	}

	return l
}

func (l *llamaCompletionRequest) WithTopK(topk int) *llamaCompletionRequest {
	if topk > 0 {
		l.TopK = topk
	}

	return l
}

func (l *llamaCompletionRequest) WithTopP(topp float32) *llamaCompletionRequest {
	if topp >= 0 && topp <= 1 {
		l.TopP = topp
	}

	return l
}

func (l *llamaCompletionRequest) WithNPredict(n_predict int) *llamaCompletionRequest {
	if n_predict > 0 {
		l.N_predict = n_predict
	}

	return l
}

func (l *llamaCompletionRequest) WithStream(stream bool) *llamaCompletionRequest {
	l.Stream = true

	return l
}

func (l *llamaCompletionRequest) WithMessages(messages []llamaCompletionMessage) *llamaCompletionRequest {
	if messages != nil {
		if l.Messages == nil {
			l.Messages = make([]llamaCompletionMessage, 0)
		}

		for _, message := range messages {
			l.Messages = append(l.Messages, message)
		}
	}

	return l
}

func (l *llamaCompletionRequest) WithCachePrompt(do_cache bool) *llamaCompletionRequest {
	l.CachePrompt = do_cache

	return l
}

func (l *llamaCompletionRequest) WithMaxTokens(max int) *llamaCompletionRequest {
	if max > 0 {
		l.MaxTokens = max
	}

	return l
}

//
//
//

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

// LlamaTokenizeRequest
type llamaTokenizeRequest struct {
	Content string `json:"content"`
}

type llamaTokenizeResponse struct {
	Tokens []uint
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

func LlamaAppendRequestMessage(
	msgs []llamaCompletionMessage,
	role string,
	content string) (result []llamaCompletionMessage) {

	return append(msgs, llamaCompletionMessage{
		Role:    role,
		Content: content,
	})
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
	data *llamaCompletionRequest,
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
		if resp.StatusCode != http.StatusOK {
			log.Printf("got non 200 code from endpoint: %s\n", resp.Status)
		}

		if err != nil {
			if err == io.EOF {
				callback(stream)
				log.Printf("EOF from response.")
			} else {
				log.Printf("resp read error: %s\n", err.Error())
			}
			break
		}

		if err = callback(stream); err != nil {
			log.Printf("GetCompletions: callback error: %s\n", err.Error())
			break
		}
	}

	return err
}

func (l *LlamaEngine) Tokenize(input string) (tokens []uint, err error) {
	var uri string = fmt.Sprintf("%s/tokenize", l.LlamaServer)
	var client *http.Client = &http.Client{}

	data := llamaTokenizeRequest{
		Content: input,
	}

	jsonBytes, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	payload := bytes.NewBuffer(jsonBytes)

	req, err := http.NewRequest(http.MethodPost, uri, payload)
	if err != nil {
		return nil, err
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	log.Println("[LlamaEngine::Tokenize] ", data)

	err = nil

	tokensJson, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var tokenResp llamaTokenizeResponse
	if err = json.Unmarshal(tokensJson, &tokenResp); err != nil {
		return nil, err
	}

	return tokenResp.Tokens, nil
}
