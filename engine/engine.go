package gorag_engine

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/qdrant/go-client/qdrant"
)

// Llama json specs
type EmbedRequestJson struct {
	Input string `json:"input"`
}

type EmbedResponseJson struct {
	Status     EngineResponseJson `json:"result"`
	Embeddings [][]float32        `json:"embeddings"`
}

type EngineResponseJson struct {
	Status  string `json:"status"`
	Message string `json:"message"`
}

type EngineCompletionRequest struct {
	Prompt string `json:"prompt"`
}

type EngineOptions struct {
	ServerUri   string
	QdrantUri   string
	EmbedServer string
	LlamaServer string
}

type GoRagEngine struct {
	ServerUrl    string
	QdrantClient *qdrant.Client
	LlamaClient  *LlamaEngine
}

func init() {
}

func NewEngine() (e *GoRagEngine) {
	return &GoRagEngine{
		QdrantClient: nil,
	}
}

func (e *GoRagEngine) Setup(options EngineOptions) (err error) {
	log.Println("[GoRagEngine] Setting up.")

	a := strings.Split(options.QdrantUri, ":")
	if len(a) != 2 {
		return fmt.Errorf("invalid QdrantUri format: got '%s', want 'HOST:PORT'", options.QdrantUri)
	}

	qdrantHost := a[0]
	qdrantPort, _ := strconv.ParseInt(a[1], 10, 32)

	log.Printf("[GoRagEngine] Qdrant client: %s:%d\n", qdrantHost, qdrantPort)
	e.QdrantClient, err = qdrant.NewClient(&qdrant.Config{
		Host: qdrantHost,
		Port: int(qdrantPort),
	})

	if err != nil {
		log.Printf("[GoRagEngine::Setup] error: %s\n", err.Error())
		return err
	}

	e.LlamaClient = NewLlamaEngine(options.EmbedServer, options.LlamaServer)
	e.ServerUrl = options.ServerUri

	return nil
}

func (e *GoRagEngine) ListenAndServe() (err error) {
	http.HandleFunc("/api/embedding", e.handleEmbedding)
	http.HandleFunc("/api/completion", e.handleCompletion)

	fmt.Printf("[gorag] Listening on '%s'...\n", e.ServerUrl)

	return http.ListenAndServe(e.ServerUrl, nil)
}

func (e *GoRagEngine) Finalize() {
	if e.QdrantClient != nil {
		e.QdrantClient.Close()
	}
}

// Private methods / http handlers
func (e *GoRagEngine) sendResponseError(err string, resp http.ResponseWriter) {
	var v EngineResponseJson = EngineResponseJson{
		Status:  "error",
		Message: err,
	}

	resp.WriteHeader(http.StatusInternalServerError)

	if b, err := json.Marshal(v); err == nil {
		resp.Write(b)
	}
}

func (e *GoRagEngine) handleEmbedding(resp http.ResponseWriter, req *http.Request) {
	var embedJson EmbedRequestJson
	if req.Method != http.MethodPost {
		resp.WriteHeader(http.StatusNotFound)
		return
	}

	reqBytes, err := io.ReadAll(req.Body)
	if err != nil {
		e.sendResponseError("could not read request data", resp)
		return
	}

	log.Printf("[handleEmbedding] got '%s'\n", string(reqBytes))

	err = json.Unmarshal(reqBytes, &embedJson)
	if err != nil {
		e.sendResponseError(err.Error(), resp)
		return
	}

	log.Printf("[handleEmbedding] got json '%v'\n", embedJson)

	embeds, err := e.LlamaClient.GetEmbeddings(embedJson.Input)
	if err != nil {
		e.sendResponseError(err.Error(), resp)
		return
	}

	var embedsLen int = len(embeds)
	log.Printf("/api/embeddings: got embeddings (%d) %v\n", embedsLen, embeds)

	erj := EmbedResponseJson{
		Status: EngineResponseJson{
			Status:  "success",
			Message: "embeddings retrieved",
		},
		Embeddings: make([][]float32, embedsLen),
	}

	for i, embed := range embeds {
		erj.Embeddings[i] = embed
	}

	erjBytes, err := json.Marshal(erj)
	if err != nil {
		e.sendResponseError(err.Error(), resp)
		return
	}

	resp.Header().Add("Content-Type", "application/json")
	written, err := resp.Write(erjBytes)
	if err != nil {
		e.sendResponseError(err.Error(), resp)
		return
	}

	log.Printf("/api/embeddings: sent %d bytes to client\n", written)
}

func (e *GoRagEngine) handleCompletion(resp http.ResponseWriter, req *http.Request) {
	var er EngineCompletionRequest

	data, err := io.ReadAll(req.Body)
	if err != nil {
		e.sendResponseError(err.Error(), resp)
		return
	}

	if err = json.Unmarshal(data, &er); err != nil {
		e.sendResponseError(err.Error(), resp)
		return
	}

	lcr := llamaCompletionRequest{
		Messages: make([]llamaCompletionMessages, 2),
		Stream:   true,
	}

	lcr.Messages[0] = llamaCompletionMessages{
		Role:    "system",
		Content: "busco sexo",
	}

	lcr.Messages[1] = llamaCompletionMessages{
		Role:    "user",
		Content: "busco amizades",
	}

	err = e.LlamaClient.GetCompletions(lcr)

	log.Printf("END handleCompletion: %s\n", err.Error())
}
