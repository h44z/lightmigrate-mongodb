# Go parameters
GOCMD=go
GOFILES:=$(shell go list ./... | grep -v /vendor/ | grep -v /examples)
BUILDDIR=dist
BINARIES=$(subst cmd/,,$(wildcard cmd/*))

.PHONY: all test clean phony

all: dep

dep:
	$(GOCMD) mod download

format:
	$(GOCMD) fmt $(GOFILES)

validate: dep
	$(GOCMD) vet $(GOFILES)
	$(GOCMD) test -race $(GOFILES)

lint: dep # requires go install golang.org/x/lint/golint
	golint -set_exit_status=1 $(GOFILES)

coverage: dep
	$(GOCMD) test $(GOFILES) -v -coverprofile .testCoverage.txt
	$(GOCMD) tool cover -func=.testCoverage.txt  # use total:\s+\(statements\)\s+(\d+.\d+\%) as Gitlab CI regextotal:\s+\(statements\)\s+(\d+.\d+\%)

coverage-html: coverage
	$(GOCMD) tool cover -html=.testCoverage.txt

test: dep
	$(GOCMD) test $(GOFILES) -v -count=1

clean:
	$(GOCMD) clean $(GOFILES)
	rm -rf .testCoverage.txt
	rm -rf $(BUILDDIR)