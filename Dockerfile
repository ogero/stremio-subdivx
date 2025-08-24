FROM node:18-alpine AS nodebuilder
WORKDIR /app/frontend
COPY frontend/package.json frontend/package-lock.json ./
RUN npm ci
COPY frontend/ ./
RUN npm run build

FROM golang:1.24-alpine AS gobuilder
RUN apk add --no-cache ca-certificates make gcc musl-dev
WORKDIR "/app/"
COPY ["go.mod", "go.sum", "./"]
RUN go mod download
COPY --from=nodebuilder /app/frontend/fs.go frontend/
COPY --from=nodebuilder /app/frontend/dist frontend/dist
COPY cmd cmd
COPY internal internal
COPY pkg pkg
RUN CC=x86_64-alpine-linux-musl-gcc go build -trimpath -ldflags="-s -w -extldflags '-static'" -o .bin/stremio-subdivx cmd/addon/*

FROM scratch
WORKDIR /app
COPY --from=gobuilder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=gobuilder /app/.bin/stremio-subdivx .
EXPOSE 3593
CMD ["./stremio-subdivx"]
