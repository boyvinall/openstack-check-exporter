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

./out/linux-amd64/% : GOOS=linux
./out/linux-amd64/% : GOARCH=amd64

.PHONY: $(BINARIES)
$(BINARIES):
	$(call PROMPT,$@)
	GOARCH=$(GOARCH) GOOS=$(GOOS) CGO_ENABLED=0 go build -o $@ ./cmd/openstack-check-exporter

.PHONY: build
build: $(BINARIES)

.PHONY: lint
lint:
	$(call PROMPT,$@)
	golangci-lint run