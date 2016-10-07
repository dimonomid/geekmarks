ROOT=../..
FLAGS_COMMON=
# TODO: pass those as env vars, and use go test $(ROOT)/...
FLAGS_DB=--geekmarks.postgres.url=postgres://geekmarks-test:geekmarks-test@localhost:6001/geekmarks-test?sslmode=disable

DCFLAGS=

V=@
ifeq ("$(VERBOSE)","1")
  V=
endif

.PHONY: all up down test

all: up test

up:
	docker-compose $(DCFLAGS) build && docker-compose $(DCFLAGS) up -d

down:
	docker-compose $(DCFLAGS) down

test:
	$(V) go test $(ROOT)/server $(FLAGS_COMMON) $(FLAGS_DB)
	$(V) go test $(ROOT)/interror $(FLAGS_COMMON)
