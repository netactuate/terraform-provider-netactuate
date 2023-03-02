HOSTNAME=github.com
NAMESPACE=netactuate
NAME=netactuate
BINARY=terraform-provider-${NAME}
VERSION=0.2.2
OS_ARCH=darwin_amd64
BINARY_NAME=${BINARY}_${VERSION}
BINARY_FULL_NAME=${BINARY_NAME}_${OS_ARCH}
DISTR_DIR=~/.terraform.d/plugins/${HOSTNAME}/${NAMESPACE}/${NAME}/${VERSION}

default: install

clear:
	rm -rf bin
	rm -rf ${DISTR_DIR}

build: clear
	go build -o ./bin/${BINARY_FULL_NAME}

release: clear
	export GOOS=darwin; export GOARCH=arm64; go build -o ./bin/${BINARY}_${VERSION}_$${GOOS}_$${GOARCH}
	export GOOS=darwin; export GOARCH=amd64; go build -o ./bin/${BINARY}_${VERSION}_$${GOOS}_$${GOARCH}
	export GOOS=linux; export GOARCH=amd64; go build -o ./bin/${BINARY}_${VERSION}_$${GOOS}_$${GOARCH}
	export GOOS=windows; export GOARCH=amd64; go build -o ./bin/${BINARY}_${VERSION}_$${GOOS}_$${GOARCH}
	md5sum ./bin/${BINARY}_${VERSION}_*

fmt:
	go fmt ./...

debug:
	dlv --listen=:50191 --headless=true --api-version=2 --accept-multiclient exec ${DISTR_DIR}/${OS_ARCH}/${BINARY_FULL_NAME} -- --debug

install: build
	mkdir -p ${DISTR_DIR}/${OS_ARCH}
	cp bin/${BINARY_FULL_NAME} ${DISTR_DIR}/${OS_ARCH}

install-all: release
	for os in 'darwin_arm64' 'darwin_amd64' 'linux_amd64' 'windows_amd64'; do \
  		mkdir -p ${DISTR_DIR}/$${os} ; \
  		cp bin/${BINARY_NAME}_$${os} ${DISTR_DIR}/$${os} ; \
	done
