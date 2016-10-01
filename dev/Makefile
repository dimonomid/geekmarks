DCFLAGS=

.PHONY: all up down

all: up

up:
	docker-compose $(DCFLAGS) build && docker-compose $(DCFLAGS) up

down:
	docker-compose $(DCFLAGS) down
