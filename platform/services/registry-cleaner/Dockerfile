ARG GO_VERSION

FROM golang:${GO_VERSION}-alpine AS builder

WORKDIR /workspace

COPY go.mod go.sum ./
RUN go mod download

COPY cmd cmd
COPY internal internal

RUN CGO_ENABLED=0 go build -o /workspace/bin/registry-cleaner ./cmd

FROM gcr.io/distroless/static:nonroot AS prod

COPY --from=builder /workspace/bin/registry-cleaner /registry-cleaner

USER 65532:65532

ENTRYPOINT ["/registry-cleaner"]
