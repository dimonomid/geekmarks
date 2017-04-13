SHELL := /bin/bash

.PHONY: docker-build docker-push

TAG ?= latest
FROM_TAG ?= latest

all: docker-build

docker-build:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go generate dmitryfrank.com/geekmarks/...
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -i -v dmitryfrank.com/geekmarks/server/cmd/geekmarks-server
	docker build -t docker.io/dimonomid/geekmarks-server .
	rm geekmarks-server

docker-tag:
	docker tag docker.io/dimonomid/geekmarks-server:$(FROM_TAG) docker.io/dimonomid/geekmarks-server:$(TAG)

docker-push:
	docker push docker.io/dimonomid/geekmarks-server:$(TAG)
