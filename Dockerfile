FROM golang:1.11.6-alpine3.8 as builder

RUN apk add git

RUN mkdir -p /go/src/cyphernode_status

ADD cnAuth /go/src/cyphernode_status/cnAuth
COPY main.go /go/src/cyphernode_status

WORKDIR /go/src/cyphernode_status

RUN go get

RUN go build main.go
RUN chmod +x /go/src/cyphernode_status/main

FROM alpine:3.8

RUN apk add ca-certificates

RUN mkdir -p /app
RUN mkdir -p /data

COPY --from=builder /go/src/cyphernode_status/main /app/cyphernode_status
ADD static /app/static
ADD templates /app/templates

ENV PATH=$PATH:/app/

WORKDIR /app/

CMD ["cyphernode_status"]