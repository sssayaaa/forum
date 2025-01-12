
FROM golang:1.20.1-alpine3.16 as base


RUN apk add --no-cache build-base


WORKDIR /web
COPY . .


RUN go build -o forum ./cmd/main.go

FROM alpine:3.16


WORKDIR /web

и
COPY --from=base /web/ /web/


COPY ./data /web/data
COPY ./cmd/config /web/cmd/config
COPY ./tls /web/tls


RUN mkdir -p /web/data && \
    chown -R nobody:nogroup /web/data && \
    chmod -R 777 /web/data


USER nobody


CMD ["./forum"]
