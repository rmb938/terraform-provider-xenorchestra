HOSTNAME=registry.terraform.io
NAMESPACE=rmb938
NAME=xenorchestra
BINARY=terraform-provider-${NAME}
VERSION ?= 0.0.1
OS_ARCH ?= linux_amd64

build:
	CGO_ENABLED=0 GOOS=linux go build -o bin/${BINARY} main.go

install: build
	mkdir -p ~/.terraform.d/plugins/${HOSTNAME}/${NAMESPACE}/${NAME}/${VERSION}/${OS_ARCH}
	cp bin/${BINARY} ~/.terraform.d/plugins/${HOSTNAME}/${NAMESPACE}/${NAME}/${VERSION}/${OS_ARCH}
