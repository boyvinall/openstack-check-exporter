.PHONY: all
all: build lint

define PROMPT
	@echo
	@echo "**********************************************************"
	@echo "*"
	@echo "*   $(1)"
	@echo "*"
	@echo "**********************************************************"
	@echo
endef

BINARY_NAME=openstack-check-exporter
BINARIES=\
	./out/linux-amd64/$(BINARY_NAME) \
	./out/linux-arm64/$(BINARY_NAME) \
	./out/darwin-amd64/$(BINARY_NAME) \

./out/linux-amd64/% : GOOS=linux
./out/linux-amd64/% : GOARCH=amd64
./out/linux-arm64/% : GOOS=linux
./out/linux-arm64/% : GOARCH=arm64
./out/darwin-amd64/% : GOOS=darwin
./out/darwin-amd64/% : GOARCH=amd64

.PHONY: $(BINARIES)
$(BINARIES):
	$(call PROMPT,$@)
	GOARCH=$(GOARCH) GOOS=$(GOOS) CGO_ENABLED=0 go build -o $@ ./cmd/openstack-check-exporter

.PHONY: build
build: $(BINARIES) docker-build

.PHONY: lint
lint:
	$(call PROMPT,$@)
	golangci-lint run

.PHONY: snyk
snyk:
	$(call PROMPT,$@)
	snyk test

GOHOSTARCH=$(shell go env GOHOSTARCH)

.PHONY: docker-build
docker-build: ./out/linux-$(GOHOSTARCH)/$(BINARY_NAME)
	$(call PROMPT,$@)
	docker build -t boyvinall/openstack-check-exporter .
