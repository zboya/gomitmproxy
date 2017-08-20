os=$1
arch=$2

export GOPATH=`pwd`
GOOS=$os GOARCH=$arch go build -o bin/gomitmproxy$os$arch src/main/*.go
