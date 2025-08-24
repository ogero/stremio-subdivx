APP = stremio-subdivx

test:
	test -z "$(shell gofmt -l .)"
	go vet ./...
	go install honnef.co/go/tools/cmd/staticcheck@latest
	staticcheck ./...
	go test -timeout 10s -race ./...

run:
	cd frontend && npm run build:dev
	go run cmd/addon/*

build:
	mkdir -p frontend/dist && touch frontend/dist/index.html
	go build -o .bin/$(APP) cmd/addon/*

docker-build:
	@docker build . --tag $(APP)

docker-run: docker-build
	docker run --rm \
		-e SERVICE_ENVIRONMENT='dk' \
		-p 3593:3593 \
		-v "./.cache:/app/.cache" \
		$(APP)