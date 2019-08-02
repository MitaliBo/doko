# docons

consul service integration for docker container

`docons` uses container id as consul service id, use container labels for other service definitions.

## Container Labels

* `docons/name`, name of the service
* `docons/port`, port of the service
* `docons/tags`, comma separated tags of the service
* `docons/meta-XXX`, meta of the service, `docons` is the reserved key

## Usage

1. Install and start `docker`

2. Install and start `consul`

    ```
    consul agent -dev -ui
    ```

3. Install and run `docons`

    ```
    go install -u go.guoyk.net/docons

    docons
    ```

4. Start a container with specified labels

    ```
    docker run -d --network host --name nginx \
        --label docons/name=demo --label docons/port=80 nginx
    ```
    
    You can also set label by `LABEL` command in `Dockerfile`
    
5. Query `consul` by browsing Web UI `http://127.0.0.1:8500`

    You will see a service with name `demo` and port `80` is registered automatically

6. Stop the container

    ```
    docker stop nginx
    ```
    
7. Check `consul` Web UI again

    You will see that service is unregistered automatically.

## Credits

Guo Y.K., MIT License
