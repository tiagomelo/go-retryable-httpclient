SHELL = /bin/bash

.PHONY: help
## help: shows this help message
help:
	@ echo "Usage: make [target]"
	@ sed -n 's/^##//p' ${MAKEFILE_LIST} | column -t -s ':' |  sed -e 's/^/ /'


.PHONY: lint
## lint: runs golangci-lint
lint: 
	@ docker run  --rm -v "`pwd`:/workspace:cached" -w "/workspace/." golangci/golangci-lint:latest golangci-lint run

.PHONY: vul-setup
## vul-setup: installs Golang's vulnerability check tool
vul-setup:
	@ if [ -z "$$(which govulncheck)" ]; then echo "Installing Golang's vulnerability detection tool..."; go install golang.org/x/vuln/cmd/govulncheck@latest; fi

.PHONY: vul-check
## vul-check: checks for any known vulnerabilities
vul-check: vul-setup
	@ govulncheck ./...

.PHONY: vet
## vet: runs go vet
vet:
	@ go vet ./...

.PHONY: test
## test: run unit tests
test:
	@ go test -cover -v ./... -count=1

.PHONY: coverage
## coverage: run unit tests and generate coverage report in html format
coverage:
	@ go test -coverprofile=coverage.out ./...  && go tool cover -html=coverage.out

.PHONY: httpbin
## httpbin: starts httpbin locally via Docker
httpbin:
	@ docker run --rm -d -p 80:80 --name httpbin kennethreitz/httpbin

.PHONY: httpbin-stop
## httpbin-stop: stops httpbin
httpbin-stop:
	@ docker stop httpbin

.PHONY: int-tests
## int-tests: runs integration tests against local httpbin instance
int-tests: httpbin
	@ sleep 1
	@ cd test ; \
	go test -v ./... -tags=integration -count=1
	@ docker stop httpbin