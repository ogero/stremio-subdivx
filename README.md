# stremio-subdivx

## Description

Stremio addon for getting subtitles from Subdivx.

### Website & Install

https://stremio-subdivx.xor.ar/

## Configuration

The following environment variables can be used to configure the addon:

*   `ADDON_HOST`: Public URL where the addon is accessible (default: `http://127.0.0.1:3593`)
*   `SERVER_LISTEN_ADDR`: Network address the HTTP server listens on (default: `:3593`)

## Build

```bash
make build
```

## Run

```bash
make run
```

## Docker

### Build

```bash
make docker-build
```

### Run

```bash
make docker-run
```