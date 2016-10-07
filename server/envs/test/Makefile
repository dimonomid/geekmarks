ROOT=../..
# Actually, we should've used -test.parallel=1 here, but for some reason
# it doesn't work. Instead, we use the undocumented flag -p=1, see
# http://stackoverflow.com/questions/15721238/go-serial-execution-of-package-tests
# for details.
#
# TODO: We currently need to prevent serialism since there are some tests which
# hit the database. Instead, we should have a separate binary for integration
# tests; because these tests aren't really _unit_ tests.
FLAGS_COMMON=-p=1

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
