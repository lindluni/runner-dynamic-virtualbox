FROM golang:1.15.8-alpine3.12 as builder
RUN mkdir /build
WORKDIR /build
COPY client .
RUN go build -o virtualbox-client .

FROM alpine:3.12
COPY --from=builder /build/virtualbox-client /usr/bin
ENTRYPOINT virtualbox-client