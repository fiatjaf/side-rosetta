all: bindata.go
	go build

bindata.go: $(shell find static)
	go-bindata -o bindata.go static/
