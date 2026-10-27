# ── Build stage ──────────────────────────────────────────────
FROM golang:1.26-alpine AS build
RUN apk add --no-cache git
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /out/api      ./cmd/api
RUN CGO_ENABLED=0 go build -o /out/migrate  ./cmd/migrate
RUN CGO_ENABLED=0 go build -o /out/seed     ./cmd/seed

# ── Runtime stage ────────────────────────────────────────────
FROM alpine:3.20
RUN apk add --no-cache ca-certificates
WORKDIR /app
COPY --from=build /out/ .
COPY entrypoint.sh .
RUN chmod +x entrypoint.sh
RUN addgroup -S hygeia && adduser -S -G hygeia hygeia
USER hygeia
EXPOSE 8080
ENTRYPOINT ["./entrypoint.sh"]
