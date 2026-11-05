FROM golang:1.23-alpine AS build
WORKDIR /src
COPY go.mod ./
COPY cmd ./cmd
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/khoorooth ./cmd/khoorooth

FROM scratch
COPY --from=build /out/khoorooth /khoorooth
EXPOSE 8080
ENTRYPOINT ["/khoorooth"]
