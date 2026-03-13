APP_NAME := share-info
DIST_DIR := dist
TARGET_OS ?= linux

.PHONY: build build-amd64 clean

build:
	go build -o $(APP_NAME) .

build-amd64:
	mkdir -p $(DIST_DIR)
	CGO_ENABLED=0 GOOS=$(TARGET_OS) GOARCH=amd64 go build -o $(DIST_DIR)/$(APP_NAME)-$(TARGET_OS)-amd64 .

clean:
	rm -rf $(DIST_DIR)
