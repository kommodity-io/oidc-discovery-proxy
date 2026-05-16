FROM --platform=$BUILDPLATFORM golang:1.26-trixie AS build

# This is set automatically by buildx
ARG TARGETARCH
ARG TARGETOS
ARG VERSION

WORKDIR /app

RUN go env -w GOCACHE=/go-cache
RUN go env -w GOMODCACHE=/gomod-cache

RUN apt-get update && apt-get install -y --no-install-recommends git make upx && rm -rf /var/lib/apt/lists/*

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN --mount=type=cache,target=/gomod-cache --mount=type=cache,target=/go-cache GOOS=${TARGETOS} GOARCH=${TARGETARCH} VERSION=${VERSION} make build

FROM gcr.io/distroless/static-debian13:nonroot AS runtime

WORKDIR /app

COPY --from=build /app/bin/oidc-discovery-proxy /app/oidc-discovery-proxy

ENV PORT=8080
EXPOSE ${PORT}
USER 65532:65532
ENTRYPOINT ["/app/oidc-discovery-proxy"]
