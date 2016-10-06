DCFLAGS=

.PHONY: all up down

all: up

up:
	docker-compose $(DCFLAGS) build && \
	docker-compose $(DCFLAGS) run geekmarks-test

down:
	docker-compose $(DCFLAGS) down
