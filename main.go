package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"

	gorag_engine "github.com/lapuglisi/gorag/v2/engine"
)

const (
	HttpDefaultPort  string = "9091"
	HttpDefaultHost  string = "localhost"
	QdrantDefaultUri string = "http://localhost:6333"

	GoragEnvHttpPort    string = "GORAG_ARG_HTTP_PORT"
	GoragEnvHttpHost    string = "GORAG_ARG_HTTP_HOST"
	GoRagEnvEmbedServer string = "GORAG_ARG_EMBED_SERVER"
	GoRagEnvLlamaServer string = "GORAG_ARG_LLAMA_SERVER"
	GoRagEnvQdrantUri   string = "GORAG_ENV_QDRANT_URI"
)

type AppOptions struct {
	HttpHost    string
	HttpPort    string
	QdrantUri   string
	EmbedServer string
	LlamaServer string
}

func getEnvOrDefault(key string, value string) string {
	var s string = os.Getenv(key)
	if len(s) == 0 {
		s = value
	}

	return s
}

func setupEnvironment(opts *AppOptions) (err error) {
	var cwd string

	cwd, err = os.Getwd()
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

	// Setup config options
	envHttpHost := getEnvOrDefault(GoragEnvHttpHost, HttpDefaultHost)
	envHttpPort := getEnvOrDefault(GoragEnvHttpPort, HttpDefaultPort)
	envEmbedServer := getEnvOrDefault(GoRagEnvEmbedServer, "")
	envLlamaServer := getEnvOrDefault(GoRagEnvLlamaServer, "")
	envQdrantUri := getEnvOrDefault(GoRagEnvQdrantUri, QdrantDefaultUri)

	flags := flag.NewFlagSet("gorag_args", flag.ExitOnError)

	flags.StringVar(&(opts.HttpPort), "port", HttpDefaultPort,
		"HTTP port to listen on (env "+GoragEnvHttpPort+")")
	flags.StringVar(&(opts.HttpHost), "host", HttpDefaultHost,
		"HTTP host to listen on (env "+GoragEnvHttpHost+")")
	flags.StringVar(&(opts.QdrantUri), "qdrant", QdrantDefaultUri,
		"Qdrant uri (env "+GoRagEnvQdrantUri+")")
	flags.StringVar(&(opts.EmbedServer), "embed-server", "",
		"Llama embedding server (env "+GoRagEnvEmbedServer+")")
	flags.StringVar(&(opts.LlamaServer), "llama", "",
		"Llama API server (env "+GoRagEnvLlamaServer+")")

	flags.Parse(os.Args[1:])
	if !flags.Parsed() {
		flags.Usage()
		return fmt.Errorf("could not parse arguments")
	}

	// Poor man's approach. Kind of ridiculous
	if len(opts.HttpHost) == 0 {
		opts.HttpHost = envHttpHost
	}

	if len(opts.HttpPort) == 0 {
		opts.HttpPort = envHttpPort
	}

	if len(opts.QdrantUri) == 0 {
		opts.QdrantUri = envQdrantUri
	}

	if len(opts.EmbedServer) == 0 {
		opts.EmbedServer = envEmbedServer
	}

	if len(opts.LlamaServer) == 0 {
		opts.LlamaServer = envLlamaServer
	}

	return nil
}

func main() {
	var options AppOptions
	var eo gorag_engine.EngineOptions
	var err error

	if err = setupEnvironment(&options); err != nil {
		log.Fatal(err)
	}

	eo = gorag_engine.EngineOptions{
		QdrantUri:   options.QdrantUri,
		EmbedServer: options.EmbedServer,
		LlamaServer: options.LlamaServer,
	}

	ge := gorag_engine.NewEngine()
	err = ge.Setup(eo)

	if err != nil {
		log.Printf("[Engine setup] error: %s\n", err.Error())
		ge.Finalize()
		os.Exit(1)
	}

	defer ge.Finalize()

	var listenAddr string = fmt.Sprintf("%s:%s", options.HttpHost, options.HttpPort)

	fmt.Printf("Listening on '%s'...\n", listenAddr)

	err = http.ListenAndServe(listenAddr, nil)
	if err != nil {
		fmt.Printf("\x1b[41;37m error \x1b[0m: %s\n", err.Error())
	}
}
