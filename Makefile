GOPATH    := $(shell go env GOPATH)
export PATH := $(PATH):$(GOPATH)/bin
GOOS?=$(shell go env GOOS)
GOARCH?=$(shell go env GOARCH)
PLATFORM := $(GOOS)/$(GOARCH)

BIN_DIR		 := bin
PROTO_DIR    := proto
GEN_DIR      := proto/gen
DIRS := $(PROTO_DIR) $(GEN_DIR) $(BIN_DIR) $(BIN_DIR)/$(PLATFORM)

COVERAGE_OUT := coverage.out
BIN_NAMES := master cli
BINS := $(BIN_NAMES:%=$(BIN_DIR)/$(PLATFORM)/%)

GO_SRCS := $(shell find . -name '*.go' -not -path './vendor/*')

PROTO_SRCS := $(PROTO_DIR)/pipeline.proto $(PROTO_DIR)/runner.proto $(PROTO_DIR)/api.proto
PROTO_GENS := $(GEN_DIR)/pipeline.pb.go $(GEN_DIR)/runner.pb.go $(GEN_DIR)/api.pb.go \
              $(GEN_DIR)/api_grpc.pb.go



# Install all build/codegen tools
.PHONY: setup
setup: $(DIRS)
	brew install protobuf
	npm install pm2
	go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
	go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
	go get modernc.org/sqlite github.com/lib/pq github.com/spf13/cobra
	cd sdk/ts && npm install
	cd web && npm install

$(DIRS):
	mkdir -p $(@)

.PHONY: proto
proto: $(PROTO_GENS)
$(PROTO_GENS): $(PROTO_SRCS)
	protoc \
		--proto_path=$(PROTO_DIR) \
		--go_out=$(GEN_DIR) \
		--go_opt=paths=source_relative \
		--go-grpc_out=$(GEN_DIR) \
		--go-grpc_opt=paths=source_relative \
		$(PROTO_SRCS)

.PHONY: test
test: $(COVERAGE_OUT)
TS_SRCS := $(shell find sdk/ts/src -name '*.ts')

$(COVERAGE_OUT): $(GO_SRCS)
	go test -coverprofile=$(COVERAGE_OUT) ./...
	cd sdk/go && go test ./...
	cd sdk/ts && npm test

.PHONY: tidy
tidy:
	go mod tidy
	cd sdk/go && go mod tidy

# Run the master service (gRPC on :9000, HTTP webhook on :8080)
.PHONY: master
master:
	go run ./cmd/master

.PHONY: build
build: $(BINS) | $(DIRS)
$(BINS): $(GO_SRCS)
	env GOOS=$(GOOS) GOARCH=$(GOARCH) go build -o $(@) ./cmd/$(@F)

# Run the Vite dev server for the web UI (proxies gRPC-web to :9000)
.PHONY: web-dev
web-dev:
	cd web && npm install && npm run dev

# Run master + web UI via pm2 (npm install -g pm2)
.PHONY: dev
dev:
	pm2 start ecosystem.config.js

.PHONY: dev-stop
dev-stop:
	pm2 stop ecosystem.config.js

.PHONY: dev-logs
dev-logs:
	pm2 logs

print-%:
	@echo $* = $($*)