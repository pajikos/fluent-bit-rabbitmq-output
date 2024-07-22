GO_FILES := out_rabbitmq.go routing_key_validator.go routing_key_creator.go record_parser.go helper.go

PLATFORMS := linux-amd64 linux-arm64 darwin-amd64 darwin-arm64
LINUX_PLATFORMS := linux-amd64 linux-arm64
MAC_PLATFORMS := darwin-amd64 darwin-arm64

# Cross-compiler variables
CC_linux-arm64 := aarch64-linux-gnu-gcc
CXX_linux-arm64 := aarch64-linux-gnu-g++
CC_darwin-arm64 := clang
CXX_darwin-arm64 := clang++

# Cross-compiler for building on macOS ARM for linux-arm64
CC_linux-arm64_mac := aarch64-linux-musl-gcc
CXX_linux-arm64_mac := aarch64-linux-musl-g++

# Cross-compiler for building on macOS for linux-amd64
CC_linux-amd64_mac := x86_64-linux-musl-gcc
CXX_linux-amd64_mac := x86_64-linux-musl-g++

all: $(PLATFORMS)

all-linux: $(LINUX_PLATFORMS)

all-mac: $(MAC_PLATFORMS)

$(PLATFORMS):
	$(MAKE) build PLATFORM=$@

build:
	$(eval GOOS := $(word 1,$(subst -, ,$(PLATFORM))))
	$(eval GOARCH := $(word 2,$(subst -, ,$(PLATFORM))))
	$(eval HOST_OS := $(shell uname -s | tr '[:upper:]' '[:lower:]'))
	$(eval HOST_ARCH := $(shell uname -m))
	$(eval CC := $(CC_$(PLATFORM)))
	$(eval CXX := $(CXX_$(PLATFORM)))
	
	@if [ "$(GOOS)" = "linux" ] && [ "$(HOST_OS)" = "darwin" ]; then \
		if [ "$(GOARCH)" = "arm64" ]; then \
			echo "Building for linux-arm64 on macOS"; \
			CC=$(CC_linux-arm64_mac) CXX=$(CXX_linux-arm64_mac) GOOS=$(GOOS) GOARCH=$(GOARCH) CGO_ENABLED=1 go build -buildmode=c-shared -o out_rabbitmq_$(PLATFORM).so $(GO_FILES); \
		elif [ "$(GOARCH)" = "amd64" ]; then \
			echo "Building for linux-amd64 on macOS"; \
			CC=$(CC_linux-amd64_mac) CXX=$(CXX_linux-amd64_mac) GOOS=$(GOOS) GOARCH=$(GOARCH) CGO_ENABLED=1 go build -buildmode=c-shared -o out_rabbitmq_$(PLATFORM).so $(GO_FILES); \
		fi; \
	else \
		echo "Building for $(PLATFORM)"; \
		CC=$(CC) CXX=$(CXX) GOOS=$(GOOS) GOARCH=$(GOARCH) CGO_ENABLED=1 go build -buildmode=c-shared -o out_rabbitmq_$(PLATFORM).so $(GO_FILES); \
	fi

clean:
	rm -rf *.so *.h *~
