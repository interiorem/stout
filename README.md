# Stout  [![Build Status](https://travis-ci.org/noxiouz/stout.svg?branch=master)](https://travis-ci.org/noxiouz/stout) [![codecov.io](https://codecov.io/github/noxiouz/stout/coverage.svg?branch=master)](https://codecov.io/github/noxiouz/stout?branch=master)

Stout is external isolation plugin for Cocaine Cloud.

### Configuration file

See configuration example:

```json
{
    "version": 2,
    "metrics": {
        "type": "graphite",
        "period": "10s",
        "args": {
            "prefix": "cloud.env.{{hostname}}.cocaine_isolate_daemon",
            "addr": ":12345"
        }
    },
    "logger": {
        "level": "debug",
        "output": "/dev/stdout"
    },
    "endpoints": ["0.0.0.0:29042"],
    "debugserver": "127.0.0.1:9000",
    "mtn": {
        "enable": false,
        "allocbuffer": 4,
        "label": "somelabelforallocation",
        "ident": "someidentforallocation",
        "url": "http://net.allocator.service.local/api",
        "headers": {
          "Authorization": "OAuth youroauthkeyfornetallocator"
        },
        "dbpath": "/path/to/state/db",
        "allowlocalstate": false
    },
    "isolate": {
        "porto": {
            "type": "porto",
            "args": {
                "layers": "/tmp",
                "cleanupenabled": true,
                "setimgurl": false,
                "weakenabled": false,
                "gc": true,
                "waitloopstepsec": 5,
                "journal": "/tmp/portojournal.jrnl",
                "containers": "/tmp",
                "registryauth": {
                    "registry.your.domain": "OAuth youroauthkeyforregistry"
                }
            }
        },
        "docker": {
            "type": "docker",
            "args": {
                "registryauth": {
                    "registry.your.domain": "authdatafordockerdaemon"
                }
            }
        },
        "process": {
            "type": "process"
        }
    }
}
```

### Build

```
go get -u github.com/noxiouz/stout/cmd/stout
cd $GOPATH/src/github.com/noxiouz/stout
go build -o cocaine-isolate-daemon cmd/stout/main.go
```

if `$GOPATH/bin` is added to `$PATH`, you can use:

```
go install github.com/noxiouz/stout/cmd/stout
```

### Run it

```bash
cocaine-isolate-daemon -config=path/to/config.conf
```
