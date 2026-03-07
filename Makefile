.PHONY: tidy
tidy:
	go mod tidy

.PHONY: style
style:
	goimports -l -w ./

.PHONY: test
test: unit-test integration-test

.PHONY: unit-test
unit-test:
	@echo "=== UNIT TESTS ==="                                                                             
	go clean -testcache && go test -v ./...

.PHONY: integration-test
integration-test:
	@echo "=== INTEGRATION TESTS ==="
	go clean -testcache && INTEGRATION=1 go test -v ./...

.PHONY: go-build
go-build:
	CGO_ENABLED=0 go build -o ./bin/gofer ./cmd/gofer

.PHONY: go-install
go-install:
	go install ./cmd/gofer
