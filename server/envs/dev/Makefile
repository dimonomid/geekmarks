ROOT=../..
DCFLAGS=

.PHONY: all up down

all: up

up:
	make -C $(ROOT)
	docker-compose $(DCFLAGS) build && docker-compose $(DCFLAGS) up

down:
	docker-compose $(DCFLAGS) down
