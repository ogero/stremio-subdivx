FROM node:20-alpine AS nodebuilder
WORKDIR /app/frontend
COPY frontend/package.json frontend/package-lock.json ./
RUN npm ci
COPY frontend/ ./
RUN npm run build

FROM golang:1.25-alpine AS gobuilder
RUN apk add --no-cache ca-certificates gcc musl-dev
WORKDIR "/app/"
COPY ["go.mod", "go.sum", "./"]
RUN go mod download
COPY --from=nodebuilder /app/frontend/fs.go frontend/
COPY --from=nodebuilder /app/frontend/dist frontend/dist
COPY cmd cmd
COPY internal internal
COPY pkg pkg
RUN CGO_ENABLED=1 CC=gcc go build -trimpath -tags netgo,osusergo -ldflags="-s -w -extldflags '-static'" -o /app/stremio-subdivx ./cmd/addon

FROM scratch
WORKDIR /app
COPY --from=gobuilder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=gobuilder /app/stremio-subdivx /app/stremio-subdivx
EXPOSE 3593
CMD ["/app/stremio-subdivx"]
