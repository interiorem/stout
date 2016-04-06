# Stout

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
        "docker": {},
        "process": {}
    }
}
```

### Build

```
go build -o cocaine-porto cmd/stout/main.go
```

### Run it

```bash
cocaine-porto -config path/to/config.conf
```
