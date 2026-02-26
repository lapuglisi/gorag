package gorag_engine

import (
	"context"
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
	Prompt      string  `json:"prompt"`
	Temperature float32 `json:"temperature"`
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

func (e *GoRagEngine) getCollectionFromModel(model string) string {

	extpos := strings.LastIndex(model, ".")
	if extpos > -1 {
		model = model[:extpos]
	}

	replacer := strings.NewReplacer(
		"_", "-",
		".", "-",
		"|", "-",
		"%", "-")

	return replacer.Replace(model)
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

	var embedsLen int = len(embeds.Embeddings)
	log.Printf("/api/embeddings: got embeddings (%d) %v\n", embedsLen, embeds)

	erj := EmbedResponseJson{
		Status: EngineResponseJson{
			Status:  "success",
			Message: "embeddings retrieved",
		},
		Embeddings: make([][]float32, embedsLen),
	}

	for i, embed := range embeds.Embeddings {
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
	var lcr llamaCompletionRequest

	resp.Header().Add("Access-Control-Allow-Origin", "https://busco.luizpuglisi.me")
	resp.Header().Add("Access-Control-Allow-Headers", "authorization, content-type")
	resp.WriteHeader(http.StatusOK)

	data, err := io.ReadAll(req.Body)
	if err != nil {
		e.sendResponseError(err.Error(), resp)
		return
	}

	er.Temperature = 0.5
	if err = json.Unmarshal(data, &er); err != nil {
		e.sendResponseError(err.Error(), resp)
		return
	}

	if len(er.Prompt) == 0 {
		e.sendResponseError("no valid input provided", resp)
		return
	}

	log.Printf("[handleCompletion] prompt is %s\n", er.Prompt)

	// Get points from qdrant
	points, err := e.getQdrantPoints(er.Prompt, er.Temperature)
	if err != nil {
		e.sendResponseError(err.Error(), resp)
		return
	}

	// Consider using 'i' and operate on 'Messages' accordingly
	if len(points) > 0 {
		lcr = llamaCompletionRequest{
			Messages: make([]llamaCompletionMessages, 2),
			Stream:   true,
		}

		sysmsg := fmt.Sprintf(
			"Use the following information to answer the user query.\n"+
				"Use the provided information as possible as you (LLM) can but feel free to "+
				"add any extra information should you (LLM) need to or feel like to.\n"+
				"Make sure to answer the user query in the language the user speaks!!!\n\n"+
				"%s", strings.Join(points, "\n"))

		lcr.Messages[0] = llamaCompletionMessages{
			Role:    "system",
			Content: sysmsg,
		}

		lcr.Messages[1] = llamaCompletionMessages{
			Role:    "user",
			Content: er.Prompt,
		}
	} else {
		lcr = llamaCompletionRequest{
			Messages: make([]llamaCompletionMessages, 1),
			Stream:   true,
		}

		lcr.Messages[0] = llamaCompletionMessages{
			Role:    "user",
			Content: er.Prompt,
		}
	}

	log.Printf("[handleCompletion] getting completion for: %v\n", lcr)

	flusher, ok := resp.(http.Flusher)
	if !ok {
		e.sendResponseError("response cannot send Server Side Events", resp)
		return
	}

	// Make sure our response write adds the correct headers for text/event-stream
	resp.Header().Add("content-type", "text/event-stream")
	resp.Header().Add("cache-control", "no-cache")
	resp.Header().Add("connection", "keep-alive")
	resp.Header().Add("transfer-encoding", "chunked")
	resp.Header().Add("keep-alive", "timeout=5, max=100")

	err = e.LlamaClient.GetCompletions(lcr, func(data string) error {
		_, err = fmt.Fprint(resp, data)
		flusher.Flush()

		if err != nil {
			log.Printf("handleCompletion: error while writing response: %s\n", err.Error())
			return err
		}

		return nil
	})
}

func (e *GoRagEngine) getQdrantPoints(input string, temp float32) (data []string, err error) {
	data = make([]string, 0)

	log.Printf("[getQdrantPoints] getting embeds from llama.\n")

	embeds, err := e.LlamaClient.GetEmbeddings(input)
	if err != nil {
		log.Printf("[getQdrantPoints] embeds error: %s\n", err.Error())
		return nil, err
	}

	collection := e.getCollectionFromModel(embeds.Model)
	limit := uint64(3)

	log.Printf("[getQdrantPoints] using collection '%s'\n", collection)

	for _, embed := range embeds.Embeddings {
		log.Printf("[getQdrantPoints] searching points for input...\n")

		sp, err := e.QdrantClient.Query(context.Background(), &qdrant.QueryPoints{
			CollectionName: collection,
			Query:          qdrant.NewQuery(embed...),
			WithVectors:    qdrant.NewWithVectorsEnable(true),
			WithPayload:    qdrant.NewWithPayloadEnable(true),
			Limit:          &limit,
		})

		if err != nil {
			log.Printf("[getQdrantPoints] qdrant error: %s\n", err.Error())
			continue
		}

		log.Printf("[getQdrantPoints] got sp, len is %d\n", len(sp))

		for _, point := range sp {
			log.Printf("[getQdrantPoints] point %s has %d payloads, score %f\n",
				point.Id, len(point.Payload), point.Score)

			if point.Score < temp {
				log.Printf("[getQdrantPoints] point %s has score %f, lower than temp %f. Skipping.\n",
					point.Id, point.Score, temp)
				continue
			}

			payload := point.Payload["source"]

			if payload == nil {
				log.Printf("[getQdrantPoints] payload 'source' not found for point %s. Skipping.\n", point.Id)
				continue
			}

			data = append(data, payload.GetStringValue())
		}
	}

	log.Printf("[getQdrantPoints] got data array: %v\n", data)

	return data, nil
}
