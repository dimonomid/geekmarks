.PHONY: docker-build docker-push

TAG ?= latest
FROM_TAG ?= latest

all: docker-build

docker-build:
	go build -i -o geekmarks-server
	docker build -t dimonomid/geekmarks-server .
	rm geekmarks-server

docker-tag:
	docker tag dimonomid/geekmarks-server:$(FROM_TAG) dimonomid/geekmarks-server:$(TAG)

docker-push:
	docker push dimonomid/geekmarks-server:$(TAG)
