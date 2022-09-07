NAME         := trezord
PLATFORMS    := linux-arm64 linux-x64 win-x64 # mac-x64

GOFLAGS      := -a

LINUX_CFLAGS := -Wno-deprecated-declarations
MAC_CFLAGS   := -Wno-deprecated-declarations -Wno-unknown-warning-option
WIN_CFLAGS   := -Wno-deprecated-declarations -Wno-implicit-function-declaration -Wno-stringop-overflow

BUILD_DIR    := build
TARGETS      := $(foreach platform,$(PLATFORMS),$(BUILD_DIR)/$(NAME)-$(platform))

all: $(TARGETS)

$(BUILD_DIR):
	mkdir -p $(BUILD_DIR)

$(BUILD_DIR)/$(NAME)-linux-arm64: $(BUILD_DIR)
	CC=aarch64-unknown-linux-gnu-gcc CGO_ENABLED=1 CGO_CFLAGS="$(LINUX_CFLAGS)" GOOS=linux GOARCH=arm64 go build $(GOFLAGS) -o $(BUILD_DIR)/$(NAME)-linux-arm64

$(BUILD_DIR)/$(NAME)-linux-x64: $(BUILD_DIR)
	CC=gcc CGO_ENABLED=1 GOOS=linux GOARCH=amd64 CGO_CFLAGS="$(LINUX_CFLAGS)" go build $(GOFLAGS) -o $(BUILD_DIR)/$(NAME)-linux-x64

$(BUILD_DIR)/$(NAME)-mac-x64: $(BUILD_DIR)
	CC=x86_64-apple-darwin14-clang CGO_ENABLED=1 GOOS=darwin GOARCH=amd64 CGO_CFLAGS="$(MAC_CFLAGS)" go build $(GOFLAGS) -o $(BUILD_DIR)/$(NAME)-mac-x64

$(BUILD_DIR)/$(NAME)-win-x64: $(BUILD_DIR)
	CC=x86_64-w64-mingw32-gcc CGO_ENABLED=1 GOOS=windows GOARCH=amd64 CGO_CFLAGS="$(WIN_CFLAGS)" go build $(GOFLAGS) -o $(BUILD_DIR)/$(NAME)-win-x64

native:
	CGO_ENABLED=1 go build $(GOFLAGS)

check:
	file $(TARGETS)

clean:
	rm -f $(TARGETS)
