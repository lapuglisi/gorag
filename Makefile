GOLLAMA_CPP_PATH = ~/sources/github.com/go-skynet/go-llama.cpp

C_INCLUDE_PATH = $(GOLLAMA_CPP_PATH)

all:

run:
	LIBRARY_PATH="$(GOLLAMA_CPP_PATH)" C_INCLUDE_PATH="$(GOLLAMA_CPP_PATH)" go run -tags openblas .

