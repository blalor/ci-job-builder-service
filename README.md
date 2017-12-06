dispatches a nomad job that builds a commit from source. 

accepts a payload containing the build script definition and url to the source.

build script definition resembles parts of a nomad jobspec.

```yaml
driver: docker
config:
    image: golang
    command: build.sh
    args:
        - "foo"
        - "bar"

constraints:
    -   attribute: "${attr.os.name}"
        value:     "ubuntu"
```

## building

    make

## running

    work/ci-job-builder-service --nomad-addr http://127.0.0.1:4646

## example

    curl -i \
        -H 'Content-Type: application/json' \
        -d '{
            "source_archive":"https://github.com/nomad-ci/push-handler-service/archive/master.tar.gz",
            "job_spec": "driver: docker\nconfig:\n    image: golang\n    args: ['date']"
        }' \
        localhost:8080/build-job
