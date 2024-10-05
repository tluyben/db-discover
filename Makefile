.PHONY: build run dev test clean

BINARY_NAME=db-discover

build:
	go build -o ${BINARY_NAME} main.go

run: build
	./${BINARY_NAME}

dev:
	go get github.com/githubnemo/CompileDaemon
	CompileDaemon -command="./${BINARY_NAME}"

test:
	go test ./...

clean:
	go clean
	rm -f ${BINARY_NAME}