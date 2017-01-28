ROOT=../..
FLAGS_COMMON= #-test.v

# Actually, we should've used -test.parallel=1 here, but for some reason
# it doesn't work. Instead, we use the undocumented flag -p=1, see
# http://stackoverflow.com/questions/15721238/go-serial-execution-of-package-tests
# for details.
FLAGS_INTEGRATION=-p=1

DCFLAGS=

V=@
ifeq ("$(VERBOSE)","1")
  V=
endif

.PHONY: all up down unit-tests integration-tests

all: unit-tests integration-tests

up:
	docker-compose $(DCFLAGS) build && docker-compose $(DCFLAGS) up -d

down:
	docker-compose $(DCFLAGS) down

integration-tests: export GM_POSTGRES_URL=postgres://geekmarks-test:geekmarks-test@localhost:6001/geekmarks-test?sslmode=disable
integration-tests: up
	$(V) echo "Integration tests:"
	$(V) go test -race -tags integration_tests $(ROOT)/... $(FLAGS_COMMON) $(FLAGS_INTEGRATION)

unit-tests:
	$(V) echo "Unit tests:"
	$(V) go test -race -tags="unit_tests" $(ROOT)/... $(FLAGS_COMMON)
