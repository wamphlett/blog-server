FROM golang as builder
COPY . /build/
WORKDIR /build/cmd/server
RUN go get && go build -o ../../bin/server

FROM ubuntu
RUN apt-get -y update
RUN apt-get -y install git
COPY --from=builder /build/bin/server /mnt/server
WORKDIR /mnt
CMD ["./server"]