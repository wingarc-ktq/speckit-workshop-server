# ビルドコンテキストはリポジトリ直下 (compose.yaml の context: .).
# 共有ライブラリ packages/authjwt に依存するため、サービスディレクトリ単体では閉じない.
FROM golang:1.26-alpine AS builder

WORKDIR /app
RUN apk add --no-cache git

# 共有モジュールと files サービスをコピー. go.work は使わず、files の go.mod の
# replace ディレクティブ (../../packages/authjwt) で解決する (GOWORK=off).
ENV GOWORK=off
COPY packages/authjwt/ ./packages/authjwt/
COPY services/files/ ./services/files/

WORKDIR /app/services/files
RUN go mod download || true
RUN CGO_ENABLED=0 GOOS=linux go build -o /bin/files ./cmd/server

FROM alpine:3.21
RUN apk add --no-cache ca-certificates
COPY --from=builder /bin/files /bin/files

EXPOSE 8082
ENTRYPOINT ["/bin/files"]
