FROM scratch

COPY bin/gitwatch-linux-amd64 /gitwatch

ENTRYPOINT ["/gitwatch"]
