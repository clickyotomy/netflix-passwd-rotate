.DEFAULT_GOAL := dev

GO111MODULE := on

CMD := netflix-passwd-rotate

BIN_DIR := ./bin
INSTALL_DIR ?= /usr/local/bin

build:
	@mkdir -p ${BIN_DIR}
	@go build -o ${BIN_DIR}/${CMD} ./...

fmt:
	@go fmt ./...

vet:
	@go vet ./...

test:
	@go test ./...

dev: build fmt vet test

install: dev
	@go install ./...

install-bin: dev
	@cp ${BIN_DIR}/${CMD} ${INSTALL_DIR}

uninstall-bin:
	@rm -f ${INSTALL_DIR}/${CMD}

clean:
	@rm -rf ${BIN_DIR}/*

.PHONY: build fmt vet test dev install install-bin uninstall-bin clean
