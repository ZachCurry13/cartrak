# The Cloudflare connector, copied from Cloudflare's official image (multi-arch).
# CarTrak runs it for you when remote access is switched on in the app.
# The version is written out so Dependabot can propose newer ones. An admin can also install a newer
# connector from the app (Remote access) without rebuilding this image.
FROM cloudflare/cloudflared:2026.10.0 AS cloudflared

FROM golang:1.27-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
ARG VERSION=dev
RUN CGO_ENABLED=0 go build -trimpath -ldflags "-s -w -X main.version=${VERSION}" -o /out/cartrak ./cmd/cartrak

FROM alpine:3.24
RUN apk add --no-cache ca-certificates tzdata && adduser -D -u 568 cartrak && mkdir /data /config && chown cartrak /data /config
COPY --from=build /out/cartrak /usr/local/bin/cartrak
COPY --from=cloudflared /usr/local/bin/cloudflared /usr/local/bin/cloudflared
# Fail the build now (not at run time) if the connector does not run here.
RUN cloudflared --version
USER cartrak
ENV CARTRAK_ADDR=:8080 CARTRAK_DATA=/data CARTRAK_CONFIG=/config
VOLUME ["/data", "/config"]
EXPOSE 8080
HEALTHCHECK --interval=30s --timeout=3s CMD wget -qO- http://127.0.0.1:8080/healthz || exit 1
ENTRYPOINT ["cartrak"]
