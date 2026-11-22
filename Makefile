.PHONY: build test madeira

## build: builds, vets and checks the formatting
build:
	go build ./...
	go vet ./...
	@test -z "$$(gofmt -l .)" || (gofmt -l . && exit 1)

## test: runs every test but the one that makes Madeira whole
test:
	go test ./...

## madeira: makes both of Madeira's maps from the cache and holds them to their sums (minutes)
madeira:
	GRAMAP_MADEIRA=1 go test -run TestMadeira -count=1 .
