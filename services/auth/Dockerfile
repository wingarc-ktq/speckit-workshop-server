# ビルドコンテキストはリポジトリ直下 (compose.yaml の context: .).
# 共有ライブラリ packages/authjwt に依存するため、サービスディレクトリ単体では閉じない.
FROM golang:1.26-alpine AS builder

WORKDIR /app
RUN apk add --no-cache git

# 共有モジュールと auth サービスをコピー. go.work は使わず、auth の go.mod の
# replace ディレクティブ (../../packages/authjwt) で解決する (GOWORK=off).
ENV GOWORK=off
COPY packages/authjwt/ ./packages/authjwt/
COPY services/auth/ ./services/auth/

WORKDIR /app/services/auth
RUN go mod download || true
RUN CGO_ENABLED=0 GOOS=linux go build -o /bin/auth ./cmd/server

FROM alpine:3.20
RUN apk add --no-cache ca-certificates
# マイグレーションはバイナリに埋め込み済み (起動時 self-migration) のためコピー不要.
COPY --from=builder /bin/auth /bin/auth

EXPOSE 8081
ENTRYPOINT ["/bin/auth"]
