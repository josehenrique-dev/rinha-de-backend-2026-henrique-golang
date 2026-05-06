FROM golang:1.24-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
COPY vendor/ vendor/
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -mod=vendor -ldflags="-s -w" -o /indexer ./cmd/indexer
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -mod=vendor -ldflags="-s -w" -o /server ./cmd/server

FROM golang:1.24-alpine AS indexer
COPY --from=builder /indexer /indexer
COPY resources/references.json.gz /data/references.json.gz
COPY resources/mcc_risk.json /data/mcc_risk.json
COPY resources/normalization.json /data/normalization.json
RUN SRC_GZ_PATH=/data/references.json.gz \
    VECTORS_PATH=/data/vectors.bin \
    LABELS_PATH=/data/labels.bin \
    INDEX_PATH=/data/index.bin \
    /indexer

FROM alpine:3.20
COPY --from=builder /server /server
COPY --from=indexer /data/vectors.bin /data/vectors.bin
COPY --from=indexer /data/labels.bin /data/labels.bin
COPY --from=indexer /data/index.bin /data/index.bin
COPY --from=indexer /data/mcc_risk.json /data/mcc_risk.json
COPY --from=indexer /data/normalization.json /data/normalization.json
ENTRYPOINT ["/server"]
