package gorag_engine

import (
	"fmt"
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
}

type EngineOptions struct {
	QdrantUri   string
	EmbedServer string
	LlamaServer string
}

type GoRagEngine struct {
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

	return nil
}

func (e *GoRagEngine) SetupEndpoints() (err error) {
	// http.HandleFunc("/api/embedding", handleEmbedding)
	// http.HandleFunc("/api/completion", handleCompletion)

	return nil
}

func (e *GoRagEngine) Finalize() {
	if e.QdrantClient != nil {
		e.QdrantClient.Close()
	}
}

// Private methods / http handlers
