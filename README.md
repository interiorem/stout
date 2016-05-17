# Stout  [![Build Status](https://travis-ci.org/noxiouz/stout.svg?branch=master)](https://travis-ci.org/noxiouz/stout) [![codecov.io](https://codecov.io/github/noxiouz/stout/coverage.svg?branch=master)](https://codecov.io/github/noxiouz/stout?branch=master)

Stout is external isolation plugin for Cocaine Cloud.

### Configuration file

See configuration example:

```json
{
    "logger": {
        "level": "debug",
        "output": "/dev/stderr"
    },
    "endpoints": ["0.0.0.0:29042"],
    "debugserver": "127.0.0.1:9000",
    "isolate": {
        "docker": {
            "endpoint": "unix:///var/run/docker.sock",
            "version": "v1.19",
            "concurrency": 10
        },
        "process": {
            "spool": "/var/spool/cocaine",
            "locator": "localhost:10053"
        }
    }
}
```

### Build

```
go build -o cocaine-isolate-daemon cmd/stout/main.go
```

### Run it

```bash
cocaine-isolate-daemon -config=path/to/config.conf
```
