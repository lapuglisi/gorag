package gorag_engine

import (
	"log"
	"os"
	"path"
	"regexp"
	"strings"

	"github.com/go-skynet/go-llama.cpp"
	"github.com/qdrant/go-client/qdrant"
)

type EngineOptions struct {
	QdrantHost string
	QdrantPort int
	LlamaModel string
}

type GoRagEngine struct {
	QdrantClient *qdrant.Client
	LLama        *llama.LLama
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
		LLama:        nil,
	}
}

func (e *GoRagEngine) Setup(options *EngineOptions) (err error) {
	log.Println("[GoRagEngine] Setting up.")

	_, err = os.Stat(options.LlamaModel)
	if os.IsNotExist(err) {
		log.Printf("[Setup] Model file '%s' is not a valid file.\n", options.LlamaModel)
		return err
	}

	col := getCollectionFromModel(options.LlamaModel)
	log.Printf("[Setup] collection name is '%s'\n", col)

	log.Printf("[GoRagEngine] Qdrant client: %s:%d\n", options.QdrantHost, options.QdrantPort)
	e.QdrantClient, err = qdrant.NewClient(&qdrant.Config{
		Host: options.QdrantHost,
		Port: options.QdrantPort,
	})

	if err != nil {
		log.Printf("[GoRagEngine::Setup] error: %s\n", err.Error())
		return err
	}

	mo := llama.ModelOptions{}
	mo.Embeddings = true

	e.LLama, err = llama.New(options.LlamaModel,
		llama.EnableEmbeddings, llama.SetContext(512))

	if err != nil {
		log.Printf("[GoRagEngine::Setup] error while loading the llama model: %s\n", err.Error())
		return err
	}

	return nil
}

func (e *GoRagEngine) Finalize() {
	if e.QdrantClient != nil {
		e.QdrantClient.Close()
	}

	if e.LLama != nil {
		e.LLama.Free()
	}
}
