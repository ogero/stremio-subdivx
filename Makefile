APP = stremio-subdivx

test:
	test -z "$(shell gofmt -l .)"
	go vet ./...
	go install honnef.co/go/tools/cmd/staticcheck@latest
	staticcheck ./...
	go test -timeout 10s -race ./...

run:
	@go run cmd/addon/*

build:
	mkdir -p frontend/dist && touch frontend/dist/index.html
	@CGO_ENABLED=0 go build -o .bin/$(APP) cmd/addon/*

docker-build:
	@docker build . --tag $(APP)

docker-run: docker-build
	@docker run --rm \
		-e ADDON_HOST='http://127.0.0.1:3593' \
		-e SERVER_LISTEN_ADDR=':3593' \
		-p 3593:3593 \
		-v "./.cache:/app/.cache" \
		$(APP)