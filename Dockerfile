FROM golang:1.20.0-alpine3.17 AS builder
WORKDIR /Api
ENV PATH="/opt/go/bin:${PATH}"
COPY . .
RUN go mod download
RUN go build
# ポート4000を公開する
EXPOSE 4000
CMD ["./server"]