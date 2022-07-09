FROM golang as builder
COPY . /build/
WORKDIR /build
RUN go get && go build -o ./bin/server ./cmd/server/main.go

FROM ubuntu
COPY --from=builder /build/bin/server /mnt/server
WORKDIR /mnt
CMD ["./server"]