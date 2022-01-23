HOSTNAME=github.com
NAMESPACE=netactuate
NAME=netactuate
BINARY=terraform-provider-${NAME}
VERSION=0.0.1
OS_ARCH=darwin_amd64
DISTR_DIR=~/.terraform.d/plugins/${HOSTNAME}/${NAMESPACE}/${NAME}/${VERSION}/${OS_ARCH}

default: install

build:
	go build -o ${BINARY}

release:
	GOOS=darwin GOARCH=amd64 go build -o ./bin/${BINARY}_${VERSION}_darwin_amd64
	GOOS=linux GOARCH=amd64 go build -o ./bin/${BINARY}_${VERSION}_linux_amd64
	GOOS=windows GOARCH=amd64 go build -o ./bin/${BINARY}_${VERSION}_windows_amd64

fmt:
	go fmt ./...

debug:
	dlv --listen=:50191 --headless=true --api-version=2 --accept-multiclient exec ${DISTR_DIR}/${BINARY} -- --debug

install: build
	mkdir -p ${DISTR_DIR}
	mv ${BINARY} ${DISTR_DIR}