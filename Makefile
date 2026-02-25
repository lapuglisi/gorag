all: build

build:
	go build .

debug:
	CGO_CFLAGS="-g" go build .

run:
	go run .

install: build
	sudo cp -iv gorag /usr/local/bin/gorag

clean:
	go clean

