# syntax=docker/dockerfile:1

ARG GO_VERSION=1.25.6
ARG ALPINE_VERSION=3.23

FROM golang:${GO_VERSION}-alpine AS build

WORKDIR /src

COPY go.mod go.sum ./

RUN go mod download

COPY . .

ARG TARGETOS=linux
ARG TARGETARCH
ARG TARGETVARIANT

RUN --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=cache,target=/go/pkg/mod \
    CGO_ENABLED=0 \
    GOOS=${TARGETOS} \
    GOARCH=${TARGETARCH} \
    GOARM="${TARGETVARIANT#v}" \
    go build \
        -trimpath \
        -ldflags="-s -w" \
        -o /out/dockerfiles-refresh \
        ./cmd/refreshmetadata

RUN --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=cache,target=/go/pkg/mod \
    CGO_ENABLED=0 \
    GOOS=${TARGETOS} \
    GOARCH=${TARGETARCH} \
    GOARM="${TARGETVARIANT#v}" \
    go build \
        -trimpath \
        -ldflags="-s -w" \
        -o /out/dockerfiles-dashboard \
        ./cmd/dashboard


FROM scratch AS dashboard

COPY --from=build /out/dockerfiles-dashboard /dockerfiles-dashboard

EXPOSE 8080

ENTRYPOINT ["/dockerfiles-dashboard"]


FROM alpine:${ALPINE_VERSION} AS certificates

RUN apk --no-cache add ca-certificates


FROM scratch AS refreshmetadata

COPY --from=certificates /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt

COPY --from=build /out/dockerfiles-refresh /dockerfiles-refresh

ENTRYPOINT ["/dockerfiles-refresh"]
