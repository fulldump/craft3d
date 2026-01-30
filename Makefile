

build:
	GOTOOLCHAIN=local go build -o bin/craft3d main.go

run:
	go run .
	
test:
	go test ./...

