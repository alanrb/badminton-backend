.PHONY: build clean deploy

start:
	docker compose up -d
	go run main.go    

specs:
	swagger generate spec -o swagger.json	

build:
	env GOARCH=arm64 GOOS=linux go build -trimpath -buildvcs=true -ldflags="-s -w" -o build/main/bootstrap main.go
	zip -j build/main.zip build/main/bootstrap

clean:
	rm -rf ./build

deploy: clean build
	sls deploy --verbose
