FROM golang:1.24 AS build

WORKDIR "/app/"

COPY ["go.mod", "go.sum", "/app/"]
RUN go mod download

COPY cmd cmd
COPY internal internal
COPY pkg pkg
COPY Makefile Makefile
RUN make build

FROM gcr.io/distroless/static

WORKDIR /app

# Copy only the built binary
COPY --from=build /app/.bin/stremio-subdivx .

EXPOSE 3593

CMD ["./stremio-subdivx"]
