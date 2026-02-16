package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"

	gorag_qdrant "github.com/lapuglisi/gorag/v2/qdrant"
)

type EmbedJson struct {
	Content string `json:"content"`
}

const (
	HttpDefaultPort  int    = 9091
	QdrantDefaultUri string = "http://localhost:6333"
)

/* Allocate a global variable */
type GoRagConfig struct {
	QdrantRag *gorag_qdrant.QdrantRag
}

func setupEnvironment() {
	var cwd string

	cwd, err := os.Getwd()
	if err != nil {
		cwd = "./"
	}

	logFile := fmt.Sprintf("%s/gorag.log", cwd)
	f, err := os.OpenFile(logFile, os.O_CREATE|os.O_APPEND|os.O_RDWR, 0660)
	if err == nil {
		log.SetOutput(f)
	} else {
		fmt.Printf("warning: using stderr as log output.\n")
		log.SetOutput(os.Stderr)
	}
}

func main() {
	var httpPort int
	var httpHost string
	var qdrantHost string
	var qdrantPort int

	var err error

	flag.IntVar(&httpPort, "port", HttpDefaultPort, "HTTP port to listen on")
	flag.StringVar(&httpHost, "host", "127.0.0.1", "HTTP host to listen on")
	flag.StringVar(&qdrantHost, "qdrant_host", "127.0.0.1", "Qdrant host")
	flag.IntVar(&qdrantPort, "qdrant_port", 6334, "Qdrant host")

	flag.Parse()
	if !flag.Parsed() {
		fmt.Println("flags not parsed.")
		os.Exit(0)
	}

	setupEnvironment()

	gc := &GoRagConfig{}

	/* Setup our GoRagInterface */
	gc.QdrantRag, err = gorag_qdrant.NewQdrantRag(qdrantHost, qdrantPort)
	if err != nil {
		log.Fatal(err)
	}

	http.HandleFunc("/api/embed", HandleEmbed)
	http.HandleFunc("/api/points", HandlePoints)

	var listenAddr string = fmt.Sprintf("%s:%d", httpHost, httpPort)

	fmt.Printf("Listening on '%s'...\n", listenAddr)

	err = http.ListenAndServe(listenAddr, nil)
	if err != nil {
		fmt.Printf("\x1b[41;37m error \x1b[0m: %s\n", err.Error())
	}
}

func HandlePoints(w http.ResponseWriter, r *http.Request) {
	const ReadSize int = 2048
	var data []byte = make([]byte, 1)

	if r.Method == http.MethodPost {
		bytes := make([]byte, ReadSize)

		for {
			rd, err := r.Body.Read(bytes)
			if err != nil && err != io.EOF {
				log.Printf("[HandlePoints] error: %s\n", err.Error())
				break
			} else if rd == 0 {
				break
			}

			data = append(data, bytes...)
		}

		log.Printf("[HandlePoints] data received: %s\n", string(data))
	}
}

func HandleEmbed(w http.ResponseWriter, r *http.Request) {

	if r.Method == http.MethodPost {
		log.Println("HandleEmbed: method POST")
		handleEmbedPost(r)
	} else {
		log.Printf("HandleEmbed: method %v\n", r.Method)
	}
}

func parseEmbedJson(j []byte) (err error) {
	var embed_json EmbedJson

	err = json.Unmarshal(j, &embed_json)
	if err != nil {
		log.Println("parseEmbedJson error:", err)
		return err
	}

	log.Println("Parsed json:", embed_json.Content)

	return nil
}

func handleEmbedPost(r *http.Request) {
	const ReadBytes int = 2048
	var bytes []byte = make([]byte, ReadBytes)

	for {
		read, err := r.Body.Read(bytes)
		if err != nil && err != io.EOF {
			log.Printf("handleEmbedPost error: %s\n", err.Error())
			break
		} else if read == 0 {
			break
		}

		res := bytes[:read]

		parseEmbedJson(res)
	}

	log.Println("end handleEmbedPost")
}
