VERSION := $$(git describe --tags --abbrev=0 2> /dev/null || git rev-parse --short HEAD)
GOFLAGS := -tags netgo -installsuffix netgo -ldflags "-X 'main.version=$(VERSION)' -w -s --extldflags '-static'"

GOOS = $$(go env GOOS)
GOARCH = $$(go env GOARCH)
BUILD_DIR = build/$(GOOS)-$(GOARCH)

clean:
	rm -rf build package

deps:
	go mod download

build: deps
	mkdir -p $(BUILD_DIR)
	GOOS=$(GOOS) GOARCH=$(GOARCH) go build $(GOFLAGS) -o ./$(BUILD_DIR)/answer

package: build
	mkdir -p package
	cd $(BUILD_DIR) && tar zcvf ../../package/answer_$(GOOS)_$(GOARCH).tar.gz answer

all:
	$(MAKE) package GOOS=linux GOARCH=amd64
	$(MAKE) package GOOS=linux GOARCH=386
	$(MAKE) package GOOS=linux GOARCH=arm64
	$(MAKE) package GOOS=linux GOARCH=arm
	$(MAKE) package GOOS=darwin GOARCH=amd64
	$(MAKE) package GOOS=darwin GOARCH=arm64

test:
	go test -v

.PHONY: clean deps build package all test