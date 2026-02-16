package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
)

type EmbedJson struct {
	Content string `json:"content"`
}

const (
	HttpDefaultPort = 9091
)

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
	flag.IntVar(&httpPort, "port", HttpDefaultPort, "HTTP port to listen on")
	flag.StringVar(&httpHost, "host", "127.0.0.1", "HTTP host to listen on")

	flag.Parse()
	if !flag.Parsed() {
		fmt.Println("flags not parsed.")
		os.Exit(0)
	}

	setupEnvironment()

	http.HandleFunc("/api/embed", HandleEmbed)

	var listenAddr string = fmt.Sprintf("%s:%d", httpHost, httpPort)

	fmt.Printf("Listening on '%s'...\n", listenAddr)

	http.ListenAndServe(listenAddr, nil)
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
