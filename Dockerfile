FROM golang:1.23-alpine AS build
WORKDIR /src
RUN apk add --no-cache ca-certificates tar
COPY go.mod ./
COPY source/khr-go-source.tar.gz ./source/khr-go-source.tar.gz
RUN tar -xzf source/khr-go-source.tar.gz -C /src && go mod download
COPY . .
RUN tar -xzf source/khr-go-source.tar.gz -C /src && CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags='-s -w' -o /out/khoorooth ./cmd/khoorooth

FROM alpine:3.21
RUN apk add --no-cache ca-certificates
WORKDIR /app
COPY --from=build /out/khoorooth /app/khoorooth
ENV HOST=:20128 PORT=20128 DATA_DIR=/data REQUIRE_API_KEY=true
VOLUME ["/data"]
EXPOSE 20128
ENTRYPOINT ["/app/khoorooth"]
