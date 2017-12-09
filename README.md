dispatches a nomad job that builds a commit from source. 

accepts a payload containing the build script definition and url to the source.

build script definition resembles parts of a nomad jobspec.

```yaml
driver: docker
config:
    image: golang
    work_dir: ${NOMAD_TASK_DIR}/go/src/github.com/example/frobnicator/
    args:
        - make

env:
    GOPATH: ${NOMAD_TASK_DIR}/go
    PATH: ${NOMAD_TASK_DIR}/go/bin:/usr/local/go/bin:/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin

resources:
    memory: 512

artifacts:
    -   source: ${NOMAD_META_nomadci_clone_source}
        destination: local/go/src/github.com/example/frobnicator/
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
