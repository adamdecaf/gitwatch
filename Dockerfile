FROM alpine:3.7

RUN apk update && apk add git

COPY bin/gitwatch-linux-amd64 /bin/gitwatch

VOLUME "/gitwatch"

ENTRYPOINT ["/bin/gitwatch"]
