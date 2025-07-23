FROM node:18-alpine AS nodebuilder
WORKDIR /app/frontend
COPY frontend/package.json frontend/package-lock.json ./
RUN npm ci
COPY frontend/ ./
RUN npm run build

FROM golang:1.24 AS gobuilder
WORKDIR "/app/"
COPY ["go.mod", "go.sum", "./"]
RUN go mod download
COPY --from=nodebuilder /app/frontend/fs.go frontend/
COPY --from=nodebuilder /app/frontend/dist frontend/dist
COPY cmd cmd
COPY internal internal
COPY pkg pkg
COPY Makefile Makefile
RUN make build

FROM gcr.io/distroless/static
WORKDIR /app
COPY --from=gobuilder /app/.bin/stremio-subdivx .
EXPOSE 3593
CMD ["./stremio-subdivx"]
