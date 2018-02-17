# gitwatch

watch git repositories and be have changes available over http

### Install / Usage

There's a docker image available: `docker pull adamdecaf/gitwatch` [Docker Hub](https://hub.docker.com/r/adamdecaf/gitwatch/)

The following flags are supported:

```
$ ./gitwatch -h
Usage of ./gitwatch:
  -config string
    	config file, newline delimited of git repos
  -interval duration
    	how often to refresh git repos (default 1h0m0s)
  -storage string
    	Local filesystem storage path, used for caching (default ".storage/")

$ docker run -t adamdecaf/gitwatch -interval 1s
2018/02/17 19:42:02 git clone on github.com/adamdecaf/gitwatch
2018/02/17 19:42:03 git fetch on github.com/adamdecaf/gitwatch
```
