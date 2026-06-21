FROM golang:1.25 AS builder
COPY . /build/
WORKDIR /build/cmd/server
RUN go get 
RUN CGO_ENABLED=0 go build -o ../../bin/server

FROM alpine
RUN apk update
RUN apk add git
WORKDIR /mnt

ARG VERSION=unknown
ENV OTEL_RESOURCE_ATTRIBUTES=service.version=${VERSION}

COPY --from=builder /build/bin/server /mnt/server
CMD ["./server"]