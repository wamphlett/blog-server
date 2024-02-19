FROM golang as builder
COPY . /build/
WORKDIR /build/cmd/server
RUN go get 
RUN CGO_ENABLED=0 go build -o ../../bin/server

FROM alpine
RUN apk update
RUN apk add git
WORKDIR /mnt
COPY --from=builder /build/bin/server /mnt/server
CMD ["./server"]