FROM golang:latest as builder

ADD . /src

WORKDIR /src
RUN ./build.sh

FROM scratch

ADD static /app/static
ADD templates /app/templates

COPY --from=builder /etc/ssl/certs /etc/ssl/certs
COPY --from=builder /src/cyphernode_welcome /cyphernode_welcome

WORKDIR /app/

CMD ["/cyphernode_welcome"]