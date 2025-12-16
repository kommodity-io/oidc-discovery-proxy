FROM --platform=$BUILDPLATFORM golang:1.25-alpine AS build

# This is set automatically by buildx
ARG TARGETARCH
ARG TARGETOS
ARG VERSION

WORKDIR /app

RUN go env -w GOCACHE=/go-cache
RUN go env -w GOMODCACHE=/gomod-cache

RUN apk update && apk add git make upx

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN --mount=type=cache,target=/gomod-cache --mount=type=cache,target=/go-cache GOOS=${TARGETOS} GOARCH=${TARGETARCH} VERSION=${VERSION} make build

FROM gcr.io/distroless/static-debian12:nonroot AS runtime

WORKDIR /app

COPY --from=build /app/bin/oidc-discovery-proxy /app/oidc-discovery-proxy

ENV PORT=8080
EXPOSE ${PORT}
ENTRYPOINT ["/app/oidc-discovery-proxy"]
