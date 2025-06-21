# stremio-subdivx

## Description

Addon for getting subtitles from Subdivx.

## Configuration

The following environment variables can be used to configure the addon:

*   `ADDON_HOST`: Public URL where the addon is accessible (default: `http://127.0.0.1:3593`)
*   `SERVER_LISTEN_ADDR`: Network address the HTTP server listens on (default: `:3593`)

## Build

```bash
make build
```

or

```bash
CGO_ENABLED=0 go build -o .bin/stremio-subdivx cmd/addon/*
```

## Run

```bash
make run
```

or

```bash
go run cmd/addon/*
```

## Docker

### Build

```bash
make docker-build
```

or

```bash
docker build . --tag stremio-subdivx
```

### Run

```bash
make docker-run
```