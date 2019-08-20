# doko

`consul` service integration for `docker` containers

`doko` uses container id as consul service id, use container labels for other service definitions.

## Container Labels

* `doko.name`, name of the service
* `doko.port`, port of the service, both `host` mode and classic `bridge` mode are supported
* `doko.tags`, tags of the service, comma separated
* `doko.check`, check mode, only `http` and `grpc` are supported
* `doko.check.interval`, check interval, default `10s`
* `doko.check.timeout`, check timeout, default `5s`
* `doko.check.http.path`, `http` mode only, the path of health check endpoint, default to `_health`
* `doko.meta.XXX`, meta of the service

## Usage

1. Install and start `docker`

2. Install and start `consul`

    ```
    consul agent -dev -ui
    ```

3. Install and run `doko`

    ```
    go install -u go.guoyk.net/doko

    doko
    ```

4. Start a container with specified labels

    ```
    docker run -d --network host --name nginx1 \
        --label doko.name=demo --label doko.port=80 nginx
    docker run -d -p 80 --name nginx2 \
        --label doko.name=demo --label doko.port=80 nginx
    ```

    You can also set label by `LABEL` command in `Dockerfile`

5. Query `consul` by browsing Web UI `http://127.0.0.1:8500`

    You will see a service with name `demo` and port `80` is registered automatically

6. Stop the container

    ```
    docker stop nginx1 nginx2
    ```

7. Check `consul` Web UI again

    You will see that service is unregistered automatically.

## Health Check

## HTTP

When label `doko.check` is set to `http`, `doko` will register a `http` health check to `consul`

```sh
http://127.0.0.1:[PORT]/_health
```

You can specify the path by label `doko.check.http.path`

## gRPC

When label `doko.check` is set to `grpc`, `doko` will register a `grpc` health check to `consul`

## Persisted Storage

`doko` will register self as a `consul` service, the instance id will be persisted to current directory

```
doko-id
```

To deregister current doko instance, run `doko --deregister` manually.

## Credits

Guo Y.K., MIT License
