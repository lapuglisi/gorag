package gorag_engine

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"path"
	"regexp"
	"strconv"
	"strings"

	gorag_model "github.com/lapuglisi/gorag/v2/model"
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
	EmbedServer  string
	LlamaServer  string
}

func init() {
}

func getCollectionFromModel(model string) (collection string) {
	var basePath string = path.Base(model)
	var final string = strings.TrimSuffix(basePath, path.Ext(model))

	re := regexp.MustCompile(`[_\%\$\.]`)

	collection = re.ReplaceAllString(final, "-")

	return collection
}

func NewEngine() (e *GoRagEngine) {
	return &GoRagEngine{
		QdrantClient: nil,
	}
}

func (e *GoRagEngine) Setup(options EngineOptions) (err error) {
	log.Println("[GoRagEngine] Setting up.")

	e.EmbedServer = options.EmbedServer

	a := strings.Split(options.QdrantUri, ":")
	if len(a) != 2 {
		return fmt.Errorf("invalid QdrantUri format: got '%s', want 'HOST:PORT'", options.QdrantUri)
	}

	qdrantHost := a[0]
	qdrantPort, _ := strconv.ParseInt(a[1], 10, 1)

	log.Printf("[GoRagEngine] Qdrant client: %s:%s\n", qdrantHost, qdrantPort)
	e.QdrantClient, err = qdrant.NewClient(&qdrant.Config{
		Host: qdrantHost,
		Port: int(qdrantPort),
	})

	if err != nil {
		log.Printf("[GoRagEngine::Setup] error: %s\n", err.Error())
		return err
	}

	err = e.getEmbeddings("serominers seroclevers serowonders seropizza")
	if err != nil {
		log.Printf("error: %s\n", err.Error())
	}

	return nil
}

func (e *GoRagEngine) Finalize() {
	if e.QdrantClient != nil {
		e.QdrantClient.Close()
	}
}

// Private methods for GoRagEngine
func (e *GoRagEngine) getEmbeddings(input string) (err error) {
	var llama_resp gorag_model.LlamaEmbedResponse
	var client = &http.Client{}

	var json_request gorag_model.LlamaEmbedRequest = gorag_model.LlamaEmbedRequest{
		Input: input,
	}

	json_bytes, err := json.Marshal(json_request)
	if err != nil {
		return err
	}

	payload := bytes.NewBuffer(json_bytes)

	// Prepare the http.Request struct
	url := fmt.Sprintf("%s/v1/embeddings", e.EmbedServer)
	req, err := http.NewRequest(http.MethodPost, url, payload)

	if err != nil {
		return err
	}

	req.Header.Add("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return err
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if err = json.Unmarshal(body, &llama_resp); err != nil {
		return err
	}

	log.Printf("[getEmbeddings] got response: %v\n", llama_resp)

	return nil
}
