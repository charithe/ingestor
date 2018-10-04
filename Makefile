DOCKER_IMAGE:=charithe/ingestor
VERSION:=$(shell git rev-parse HEAD)

.PHONY: gen-protos test container docker launch

vendor:
	@dep ensure -v

gen-protos:
	@prototool all

test: vendor
	@go test ./...

container: vendor
	CGO_ENABLED=0 GOOS=linux go build -a -ldflags '-w -s' .

docker:
	@docker build --rm -t $(DOCKER_IMAGE):$(VERSION) .

launch:
	@docker-compose up


