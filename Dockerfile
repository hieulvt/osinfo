# syntax=docker/dockerfile:1

FROM golang:1.26-alpine AS builder

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .

ARG VERSION=0.1.0
ARG COMMIT=none
ARG BUILD_DATE=unknown

RUN CGO_ENABLED=0 go build -trimpath \
    -ldflags="-s -w -X github.com/hieulvt/osinfo/internal/version.Version=${VERSION} -X github.com/hieulvt/osinfo/internal/version.Commit=${COMMIT} -X github.com/hieulvt/osinfo/internal/version.BuildDate=${BUILD_DATE}" \
    -o /out/osinfo ./cmd/osinfo

FROM gcr.io/distroless/static-debian12:nonroot

COPY --from=builder /out/osinfo /usr/local/bin/osinfo

ENTRYPOINT ["/usr/local/bin/osinfo"]
CMD ["all"]

