VERSION := $(shell echo $(shell git describe --tags) | sed 's/^v//')
COMMIT  := $(shell git log -1 --format='%H')

all: swagger build

###############################################################################
#                               Build / Install                               #
###############################################################################

LD_FLAGS = -X github.com/bitsongofficial/bitsong-media-server/cmd.Version=$(VERSION) \
	-X github.com/bitsongofficial/bitsong-media-server/cmd.Commit=$(COMMIT)

BUILD_FLAGS := -ldflags '$(LD_FLAGS)'

build: go.sum
ifeq ($(OS),Windows_NT)
	@echo "building bitsongms binary..."
	@go build -mod=readonly $(BUILD_FLAGS) -o build/bitsongms.exe .
else
	@echo "building bitsongms binary..."
	@go build -mod=readonly $(BUILD_FLAGS) -o build/bitsongms .
endif

install: go.sum
	@echo "installing bitsongms binary..."
	@go install -mod=readonly $(BUILD_FLAGS) .

###############################################################################
#                                   Docs                                      #
###############################################################################

swagger:
	@swag init --generatedTime=false -g server/swagger.go --output=server/docs

.PHONY: install build swagger clean