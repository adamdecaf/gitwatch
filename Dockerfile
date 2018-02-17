FROM alpine:3.7

RUN apk update && apk add git

COPY bin/gitwatch-linux-amd64 /gitwatch

ENTRYPOINT ["/gitwatch"]
