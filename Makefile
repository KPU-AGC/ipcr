# ./Makefile
BIN_DIR := bin
GO ?= go

# Embed a version at build-time; falls back to "dev".
VERSION := $(shell git describe --tags --dirty --always 2>/dev/null || echo dev)
LDFLAGS := -s -w -X ipcr/internal/version.Version=$(VERSION)
GOFLAGS ?= -trimpath

# ---- Race detector auto-detection -------------------------------------------
CGO_ENABLED ?= $(shell $(GO) env CGO_ENABLED)
GOOS := $(shell $(GO) env GOOS)
GOARCH := $(shell $(GO) env GOARCH)
CCBIN := $(shell $(GO) env CC)
HASCC := $(shell command -v $(CCBIN) >/dev/null 2>&1 && echo yes || echo no)

# Race detector is "supported" only if cgo is enabled AND a C compiler exists.
RACE_SUPPORTED := $(if $(and $(filter 1,$(CGO_ENABLED)),$(filter yes,$(HASCC))),yes,no)
RACEFLAG := $(if $(filter yes,$(RACE_SUPPORTED)),-race,)

define maybe_echo_skip_race
	@if [ "$(RACE_SUPPORTED)" != "yes" ]; then \
	  echo "NOTE: Race detector disabled (CGO_ENABLED=$(CGO_ENABLED), CC=$(CCBIN), GOOS=$(GOOS), GOARCH=$(GOARCH))."; \
	  echo "      To enable, install a C toolchain and run with CGO_ENABLED=1."; \
	fi
endef
# -----------------------------------------------------------------------------

.PHONY: all build build-race test test-race test-short bench cover fmt vet lint tidy clean

all: build

build:
	mkdir -p $(BIN_DIR)
	$(GO) build $(GOFLAGS) -ldflags "$(LDFLAGS)" -o $(BIN_DIR)/ipcr ./cmd/ipcr
	$(GO) build $(GOFLAGS) -ldflags "$(LDFLAGS)" -o $(BIN_DIR)/ipcr-probe ./cmd/ipcr-probe
	$(GO) build $(GOFLAGS) -ldflags "$(LDFLAGS)" -o $(BIN_DIR)/ipcr-multiplex ./cmd/ipcr-multiplex
	$(GO) build $(GOFLAGS) -ldflags "$(LDFLAGS)" -o $(BIN_DIR)/ipcr-nested ./cmd/ipcr-nested
	$(GO) build $(GOFLAGS) -tags "thermo" -ldflags "$(LDFLAGS)" -o $(BIN_DIR)/ipcr-thermo ./cmd/ipcr-thermo

# Force a race build; fails with a helpful message if unsupported.
build-race:
	@if [ "$(RACE_SUPPORTED)" != "yes" ]; then \
	  echo "ERROR: build-race requested but race detector not available."; \
	  echo "       Install a C toolchain and export CGO_ENABLED=1."; \
	  exit 1; \
	fi
	mkdir -p $(BIN_DIR)
	$(GO) build $(GOFLAGS) -race -ldflags "$(LDFLAGS)" -o $(BIN_DIR)/ipcr ./cmd/ipcr
	$(GO) build $(GOFLAGS) -race -ldflags "$(LDFLAGS)" -o $(BIN_DIR)/ipcr-probe ./cmd/ipcr-probe
	$(GO) build $(GOFLAGS) -race -ldflags "$(LDFLAGS)" -o $(BIN_DIR)/ipcr-multiplex ./cmd/ipcr-multiplex
	$(GO) build $(GOFLAGS) -race -ldflags "$(LDFLAGS)" -o $(BIN_DIR)/ipcr-nested ./cmd/ipcr-nested
	$(GO) build $(GOFLAGS) -race -tags "thermo" -ldflags "$(LDFLAGS)" -o $(BIN_DIR)/ipcr-thermo ./cmd/ipcr-thermo

# Auto: uses -race when supported; otherwise skips it with a note.
test:
	$(maybe_echo_skip_race)
	$(GO) test $(GOFLAGS) $(RACEFLAG) ./... -count=1

test-race:
	@if [ "$(RACE_SUPPORTED)" != "yes" ]; then \
	  echo "ERROR: test-race requested but race detector not available."; \
	  echo "       Install a C toolchain and export CGO_ENABLED=1."; \
	  exit 1; \
	fi
	$(GO) test $(GOFLAGS) -race ./... -count=1

test-short:
	$(maybe_echo_skip_race)
	$(GO) test $(GOFLAGS) $(RACEFLAG) ./... -short -count=1

bench:
	$(maybe_echo_skip_race)
	$(GO) test $(GOFLAGS) $(RACEFLAG) ./... -bench=. -run=^$$

cover:
	$(maybe_echo_skip_race)
	$(GO) test $(GOFLAGS) $(RACEFLAG) ./... -coverprofile=coverage.out
	$(GO) tool cover -func=coverage.out

fmt:
	@echo "Formatting..."
	@$(GO) fmt ./...

vet:
	$(GO) vet ./...

lint:
	@command -v golangci-lint >/dev/null 2>&1 || { \
	  echo "golangci-lint not found; install from https://golangci-lint.run/"; exit 0; }
	golangci-lint run

tidy:
	$(GO) mod tidy

clean:
	rm -rf $(BIN_DIR) coverage.out
