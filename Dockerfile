FROM golang:1.26-alpine AS build

WORKDIR /src

ARG VERSION=dev
ARG COMMIT=unknown
ARG BUILD_DATE=unknown

COPY go.mod ./
RUN go mod download

COPY . .

RUN set -eu; \
    test -n "${VERSION}"; \
    test -n "${COMMIT}"; \
    test -n "${BUILD_DATE}"; \
    CGO_ENABLED=0 GOOS=linux go build \
        -trimpath \
        -ldflags="-s -w \
            -X github.com/shuklarituparn/cloud_cli/internal/buildinfo.Version=${VERSION} \
            -X github.com/shuklarituparn/cloud_cli/internal/buildinfo.Commit=${COMMIT} \
            -X github.com/shuklarituparn/cloud_cli/internal/buildinfo.Date=${BUILD_DATE}" \
        -o /out/cloud_cli \
        ./cmd/cloud_cli

RUN mkdir -p /workspace \
    && chown 65532:65532 /workspace

FROM scratch

ARG VERSION=dev
ARG COMMIT=unknown
ARG BUILD_DATE=unknown

LABEL org.opencontainers.image.title="cloud_cli" \
      org.opencontainers.image.description="CLI for managing cloud profiles" \
      org.opencontainers.image.source="https://github.com/shuklarituparn/cloud_cli" \
      org.opencontainers.image.version="${VERSION}" \
      org.opencontainers.image.revision="${COMMIT}" \
      org.opencontainers.image.created="${BUILD_DATE}"

COPY --from=build /out/cloud_cli /cloud_cli
COPY --from=build --chown=65532:65532 /workspace /workspace

WORKDIR /workspace
USER 65532:65532

ENTRYPOINT ["/cloud_cli"]