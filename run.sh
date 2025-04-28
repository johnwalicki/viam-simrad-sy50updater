#!/bin/sh
cd `dirname $0`

MODULE=$(basename "$PWD")
export PATH=$PATH:$(go env GOPATH)/bin


echo "Downloading necessary go packages..."
if ! (
    go get go.viam.com/rdk@latest
    go get golang.org/x/tools/cmd/goimports@latest
    gofmt -w -s .
    go mod tidy
); then
    echo "Go packages could not be installed. Quitting..." >&2
    exit 1
fi
# entrypoint is bin/$MODULE as specified in meta.json
# go build -o bin/$MODULE main.go
echo "Building bin/$MODULE.exe"
GOOS=windows GOARCH=amd64 go build -o bin/$MODULE.exe main.go


# tar czf module.tar.gz bin/$MODULE
#echo "Starting module..."
#exec go run main.go $@
