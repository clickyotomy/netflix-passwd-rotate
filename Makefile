.DEFAULT_GOAL := all

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

all: build fmt vet

install: all
	@cp ${BIN_DIR}/${CMD} ${INSTALL_DIR}

clean:
	@rm -rf ${BIN_DIR}/*

.PHONY: build fmt vet all install clean
