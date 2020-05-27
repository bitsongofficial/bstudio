VERSION := $(shell echo $(shell git describe --tags) | sed 's/^v//')
COMMIT  := $(shell git log -1 --format='%H')

all: swagger build install

###############################################################################
#                               Build / Install                               #
###############################################################################

LD_FLAGS = -X github.com/bitsongofficial/bstudio/cmd.Version=$(VERSION) \
	-X github.com/bitsongofficial/bstudio/cmd.Commit=$(COMMIT)

BUILD_FLAGS := -ldflags '$(LD_FLAGS)'

build: go.sum
ifeq ($(OS),Windows_NT)
	@echo "building bstudio binary..."
	@go build -mod=readonly $(BUILD_FLAGS) -o build/bstudio.exe .
else
	@echo "building bstudio binary..."
	@go build -mod=readonly $(BUILD_FLAGS) -o build/bstudio .
endif

install: go.sum
	@echo "installing bstudio binary..."
	@go install -mod=readonly $(BUILD_FLAGS) .

###############################################################################
#                                   Docs                                      #
###############################################################################

swagger:
	@swag init --generatedTime=false -g server/swagger.go --output=server/docs

.PHONY: install build swagger clean