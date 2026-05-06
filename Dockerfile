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
    INDEX_PATH=/data/ivf.bin \
    /indexer

FROM alpine:3.20
COPY --from=builder /server /server
COPY --from=indexer /data/ivf.bin /data/ivf.bin
COPY --from=indexer /data/mcc_risk.json /data/mcc_risk.json
COPY --from=indexer /data/normalization.json /data/normalization.json
ENTRYPOINT ["/server"]
