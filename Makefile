.PHONY: unpack build test
unpack:
	tar -xzf source/khr-go-source.tar.gz -C .
build: unpack
	go build -trimpath -ldflags='-s -w' -o khoorooth ./cmd/khoorooth
test: unpack
	go test ./...
