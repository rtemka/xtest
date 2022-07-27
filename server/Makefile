PUBLIC_REGISTRY_HOST=docker.io
PUBLIC_REGISTRY_OWNER=rtemka
PUBLIC_REGISTRY_APP_NAME=xtestserver

CI_COMMIT_REF_NAME=latest

all: deps build test lint

deps:
	go mod tidy && go mod download

build:
	go build -o ./bin/ ./cmd/

run:
	go run ./cmd/

test:
	go test -v -cover -count=1 ./...

lint:
	golangci-lint run ./...

clean:
	rm -rf ./bin

image: image_build image_push

image_build:
	@docker build -t ${PUBLIC_REGISTRY_HOST}/${PUBLIC_REGISTRY_OWNER}/${PUBLIC_REGISTRY_APP_NAME}:${CI_COMMIT_REF_NAME} ./

image_push:
	@docker push ${PUBLIC_REGISTRY_HOST}/${PUBLIC_REGISTRY_OWNER}/${PUBLIC_REGISTRY_APP_NAME}:${CI_COMMIT_REF_NAME}
	@echo "${PUBLIC_REGISTRY_HOST}/${PUBLIC_REGISTRY_OWNER}/${PUBLIC_REGISTRY_APP_NAME} image published. Version ${CI_COMMIT_REF_NAME}"
