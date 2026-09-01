FROM golang:1.27-alpine AS build

WORKDIR /src
COPY go.mod ./
COPY cmd/keelmesh-core ./cmd/keelmesh-core
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/keelmesh-core ./cmd/keelmesh-core

FROM alpine:3.22

RUN addgroup -S keelmesh && adduser -S -G keelmesh keelmesh
COPY --from=build /out/keelmesh-core /usr/local/bin/keelmesh-core

USER keelmesh
EXPOSE 8080
ENTRYPOINT ["/usr/local/bin/keelmesh-core"]

