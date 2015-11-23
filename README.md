# Stout

It's an adapter to spawn workers in [Cocaine cloud](https://github.com/cocaine/) using [Porto](https://github.com/yandex/porto/) via [docker-plugin](https://github.com/cocaine/cocaine-plugins/). Stouts implements `Docker v1.14 API` and maps it to `Porto API`.
Stout supports Docker images and can download them from Registry (v1 only at this moment). Porto layer API is used to create containers
with an overlay volume from docker image layers.

## Configuration

It has just a few options:
 + `http` - endpoint to listen to incoming connections from Cocaine
 + `loglevel` - logging level
 + `root` - root namespace for applications containers. We use it to run Porto inside Porto
 + `layers` - path, where layers from are download for importing to Porto
 + `volumes` - path, where new volumes will be created. Each app has its own volume

## How it works

Currently Stout creates a namespace for each app inside the root namespace. Workers will be spawned inside its named namespace.
It allows us to limit all the workers of one application.

Example:
```
root
|____ example_app
    |________ worker1
    |________ worker2
```
