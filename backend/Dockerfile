# ── Build stage ──────────────────────────────────────────────
FROM golang:1.26-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -o /out/api ./cmd/api \
    && CGO_ENABLED=0 go build -trimpath -o /out/migrate ./cmd/migrate \
    && CGO_ENABLED=0 go build -trimpath -o /out/seed ./cmd/seed \
    && CGO_ENABLED=0 go build -trimpath -o /out/seed-workflow ./cmd/seed-workflow \
    && CGO_ENABLED=0 go build -trimpath -o /out/link-clerk-user ./cmd/link-clerk-user

# ── Runtime stage ────────────────────────────────────────────
FROM alpine:3.23
RUN apk add --no-cache ca-certificates
WORKDIR /app
COPY --from=build /out/ .
COPY entrypoint.sh .
RUN chmod +x entrypoint.sh
RUN addgroup -S hygeia && adduser -S -G hygeia hygeia
USER hygeia
ENV PATH="/app:${PATH}" PORT=8080
EXPOSE 8080
HEALTHCHECK --interval=30s --timeout=5s --start-period=20s --retries=3 \
    CMD wget -q --spider "http://127.0.0.1:${PORT}/api/v1/readyz" || exit 1
ENTRYPOINT ["./entrypoint.sh"]
CMD ["api"]
