FROM golang:1.11.6-alpine3.8 as builder

RUN apk add git build-base

RUN mkdir -p /go/src/cyphernode_welcome

ADD cnAuth /go/src/cyphernode_welcome/cnAuth
COPY main.go /go/src/cyphernode_welcome

WORKDIR /go/src/cyphernode_welcome

RUN go get

RUN go build main.go
RUN chmod +x /go/src/cyphernode_welcome/main

FROM cyphernode/alpine-glibc-base:3.8

RUN apk add ca-certificates

RUN mkdir -p /app
RUN mkdir -p /data

COPY --from=builder /go/src/cyphernode_welcome/main /app/cyphernode_welcome
ADD static /app/static
ADD templates /app/templates

ENV PATH=$PATH:/app/

WORKDIR /app/

CMD ["cyphernode_welcome"]