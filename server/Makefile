.PHONY: docker-build docker-push

TAG ?= latest
FROM_TAG ?= latest

all: docker-build

docker-build:
	go build -i -o geekmarks-server
	docker build -t docker.io/dimonomid/geekmarks-server .
	rm geekmarks-server

docker-tag:
	docker tag docker.io/dimonomid/geekmarks-server:$(FROM_TAG) docker.io/dimonomid/geekmarks-server:$(TAG)

docker-push:
	docker push docker.io/dimonomid/geekmarks-server:$(TAG)
