ROOT=../..
FLAGS_COMMON=

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

test: export GM_POSTGRES_URL=postgres://geekmarks-test:geekmarks-test@localhost:6001/geekmarks-test?sslmode=disable
test:
	$(V) go test $(ROOT)/... $(FLAGS_COMMON)
