FROM golang:1.27-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
ARG VERSION=dev
RUN CGO_ENABLED=0 go build -trimpath -ldflags "-s -w -X main.version=${VERSION}" -o /out/cartrak ./cmd/cartrak

FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata && adduser -D -u 568 cartrak && mkdir /data /config && chown cartrak /data /config
COPY --from=build /out/cartrak /usr/local/bin/cartrak
USER cartrak
ENV CARTRAK_ADDR=:8080 CARTRAK_DATA=/data CARTRAK_CONFIG=/config
VOLUME ["/data", "/config"]
EXPOSE 8080
HEALTHCHECK --interval=30s --timeout=3s CMD wget -qO- http://127.0.0.1:8080/healthz || exit 1
ENTRYPOINT ["cartrak"]
