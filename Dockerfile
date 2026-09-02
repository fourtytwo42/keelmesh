FROM node:24-alpine AS web

WORKDIR /web
COPY web/package.json web/package-lock.json ./
RUN npm ci
COPY web ./
COPY contracts/fixtures /contracts/fixtures
RUN npm run build

FROM golang:1.27-alpine AS build

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY internal ./internal
COPY cmd/keelmesh-core ./cmd/keelmesh-core
COPY cmd/keelmesh-ingress ./cmd/keelmesh-ingress
COPY --from=web /web/dist ./cmd/keelmesh-core/web
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/keelmesh-core ./cmd/keelmesh-core
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/keelmesh-ingress ./cmd/keelmesh-ingress

FROM alpine:3.22

RUN addgroup -S keelmesh && adduser -S -G keelmesh keelmesh
COPY --from=build /out/keelmesh-core /usr/local/bin/keelmesh-core
COPY --from=build /out/keelmesh-ingress /usr/local/bin/keelmesh-ingress

USER keelmesh
EXPOSE 8080
ENTRYPOINT ["/usr/local/bin/keelmesh-core"]
