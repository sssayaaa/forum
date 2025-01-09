FROM golang:alpine AS build
ENV CGO_ENABLED=1
RUN apk add --no-cache gcc musl-dev
WORKDIR /app
COPY . .
RUN go build -o forum ./cmd/main.go

FROM alpine:latest
RUN apk add --no-cache sqlite-libs && \
    adduser -D appuser
WORKDIR /app
RUN mkdir -p ./cmd/config && mkdir -p ./data && mkdir -p ./web/templates && \
    chown -R appuser:appuser /app
USER appuser
COPY --from=build /app/forum ./forum
COPY --from=build /app/cmd/config/Config.json ./cmd/config/
COPY --from=build /app/internal/database/migration ./internal/database/migration
COPY --from=build /app/internal/web/templates/*.html ./web/templates/
COPY --from=build /app/tls ./tls
EXPOSE 8080
CMD ["./forum"]