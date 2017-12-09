package job_builder_test

import (
	. "github.com/nomad-ci/ci-job-builder-service/internal/app/job_builder"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

    "github.com/stretchr/testify/mock"

    "bytes"
    "net/http"
    "net/http/httptest"
    "github.com/gorilla/mux"
    "encoding/json"

    nomadapi "github.com/hashicorp/nomad/api"
    "github.com/nomad-ci/ci-job-builder-service/internal/pkg/interfaces"
)

var _ = Describe("JobBuilder", func() {
    endpoint := "http://example.com/build-job"

    var handler *JobBuilder
    var router *mux.Router
    var resp *httptest.ResponseRecorder

    var mockNomadJobs interfaces.MockNomadJobs

    defaultTaskArtifact := &nomadapi.TaskArtifact{
        GetterSource: StringToPtr("${NOMAD_META_nomadci_clone_source}"),
    }

    submitPayload := func(payload map[string]string) *nomadapi.Job {
        requestBody, _ := json.Marshal(payload)

        req, err := http.NewRequest("POST", endpoint, bytes.NewReader(requestBody))
        Expect(err).ShouldNot(HaveOccurred())

        req.Header.Add("Content-Type", "application/json")

        router.ServeHTTP(resp, req)
        Expect(resp.Code).To(Equal(http.StatusNoContent))

        mockNomadJobs.AssertExpectations(GinkgoT())

        return mockNomadJobs.Calls[0].Arguments[0].(*nomadapi.Job)
    }

    BeforeEach(func() {
        router = mux.NewRouter()
        resp = httptest.NewRecorder()

        mockNomadJobs = interfaces.MockNomadJobs{}

        handler = NewJobBuilder(&mockNomadJobs)
        handler.InstallHandlers(router.PathPrefix("/").Subrouter())

        mockNomadJobs.
            On(
                "Register",
                mock.AnythingOfType("*api.Job"),
                mock.AnythingOfType("*api.WriteOptions"),
            ).
            Return(
                &nomadapi.JobRegisterResponse{
                    EvalID: "cafedead-beef-cafe-dead-beefcafedead",
                },
                &nomadapi.WriteMeta{},
                nil,
            )
    })

    It("builds and submits a job", func() {
        job := submitPayload(map[string]string{
            "source_archive": "s3.amazonaws.com/bucket/archive.tar.gz",
            "job_spec": `
driver: docker
config:
    image: golang
    work_dir: ${NOMAD_TASK_DIR}/work
    command: build.sh

env:
    foo: bar

resources:
    memory: 100
`,
        })

        // id must be provided, needs to be unique
        Expect(job.ID).ToNot(BeNil())
        Expect(*job.ID).To(HavePrefix("ci-job/"))

        // name must be provided
        Expect(job.Name).To(Equal(job.ID))

        Expect(job.Datacenters).To(Equal([]string{"dc1"}))

        Expect(job.Type).To(Equal(StringToPtr("batch")))

        Expect(job.TaskGroups).To(HaveLen(1))
        Expect(job.TaskGroups[0]).To(Equal(&nomadapi.TaskGroup{
            Name: StringToPtr("builder"),
            RestartPolicy: &nomadapi.RestartPolicy{
                Attempts: IntToPtr(0),
                Mode: StringToPtr("fail"),
            },

            Tasks: []*nomadapi.Task{
                &nomadapi.Task{
                    Name: "builder",

                    Driver: "docker",
                    Config: map[string]interface{}{
                        "image": "golang",
                        "command": "build.sh",
                        "work_dir": "${NOMAD_TASK_DIR}/work",
                    },

                    Meta: map[string]string{
                        "nomadci.clone_source": "s3.amazonaws.com/bucket/archive.tar.gz",
                    },

                    Env: map[string]string{
                        "foo": "bar",
                    },

                    Resources: &nomadapi.Resources{
                        MemoryMB: IntToPtr(100),
                    },

                    Artifacts: []*nomadapi.TaskArtifact{
                        defaultTaskArtifact,
                    },
                },
            },
        }))
    })

    It("builds and submits a job with an artifact", func() {
        job := submitPayload(map[string]string{
            "source_archive": "s3.amazonaws.com/bucket/archive.tar.gz",
            "job_spec": `
driver: docker
config:
    image: golang
    work_dir: ${NOMAD_TASK_DIR}/work
    command: build.sh

artifacts:
    -   source: https://example.com/foo-bar
        destination: local/bin/
        mode: file
        options:
            checksum: sha256:322152b8b50b26e5e3a7f6ebaeb75d9c11a747e64bbfd0d8bb1f4d89a031c2b5
`,
        })

        // id must be provided, needs to be unique
        Expect(job.ID).ToNot(BeNil())
        Expect(*job.ID).To(HavePrefix("ci-job/"))

        // name must be provided
        Expect(job.Name).To(Equal(job.ID))

        Expect(job.Datacenters).To(Equal([]string{"dc1"}))

        Expect(job.Type).To(Equal(StringToPtr("batch")))

        Expect(job.TaskGroups).To(HaveLen(1))
        Expect(job.TaskGroups[0]).To(Equal(&nomadapi.TaskGroup{
            Name: StringToPtr("builder"),
            RestartPolicy: &nomadapi.RestartPolicy{
                Attempts: IntToPtr(0),
                Mode: StringToPtr("fail"),
            },

            Tasks: []*nomadapi.Task{
                &nomadapi.Task{
                    Name: "builder",

                    Driver: "docker",
                    Config: map[string]interface{}{
                        "image": "golang",
                        "command": "build.sh",
                        "work_dir": "${NOMAD_TASK_DIR}/work",
                    },

                    Meta: map[string]string{
                        "nomadci.clone_source": "s3.amazonaws.com/bucket/archive.tar.gz",
                    },

                    Artifacts: []*nomadapi.TaskArtifact{
                        defaultTaskArtifact,
                        &nomadapi.TaskArtifact{
                            GetterSource: StringToPtr("https://example.com/foo-bar"),
                            RelativeDest: StringToPtr("local/bin/"),
                            GetterMode:   StringToPtr("file"),
                            GetterOptions: map[string]string{
                                "checksum": "sha256:322152b8b50b26e5e3a7f6ebaeb75d9c11a747e64bbfd0d8bb1f4d89a031c2b5",
                            },
                        },
                    },
                },
            },
        }))
    })

    It("builds and submits a job with an artifact", func() {
        job := submitPayload(map[string]string{
            "source_archive": "s3.amazonaws.com/bucket/archive.tar.gz",
            "job_spec": `
driver: docker
config:
    image: golang
    work_dir: ${NOMAD_TASK_DIR}/work/src/github.com/example/my-repo/
    command: build.sh

artifacts:
    -   source: ${NOMAD_META_nomadci_clone_source}
        destination: local/work/src/github.com/example/my-repo/
`,
        })

        // id must be provided, needs to be unique
        Expect(job.ID).ToNot(BeNil())
        Expect(*job.ID).To(HavePrefix("ci-job/"))

        // name must be provided
        Expect(job.Name).To(Equal(job.ID))

        Expect(job.Datacenters).To(Equal([]string{"dc1"}))

        Expect(job.Type).To(Equal(StringToPtr("batch")))

        Expect(job.TaskGroups).To(HaveLen(1))
        Expect(job.TaskGroups[0]).To(Equal(&nomadapi.TaskGroup{
            Name: StringToPtr("builder"),
            RestartPolicy: &nomadapi.RestartPolicy{
                Attempts: IntToPtr(0),
                Mode: StringToPtr("fail"),
            },

            Tasks: []*nomadapi.Task{
                &nomadapi.Task{
                    Name: "builder",

                    Driver: "docker",
                    Config: map[string]interface{}{
                        "image": "golang",
                        "command": "build.sh",
                        "work_dir": "${NOMAD_TASK_DIR}/work/src/github.com/example/my-repo/",
                    },

                    Meta: map[string]string{
                        "nomadci.clone_source": "s3.amazonaws.com/bucket/archive.tar.gz",
                    },

                    Artifacts: []*nomadapi.TaskArtifact{
                        &nomadapi.TaskArtifact{
                            GetterSource: StringToPtr("${NOMAD_META_nomadci_clone_source}"),
                            RelativeDest: StringToPtr("local/work/src/github.com/example/my-repo/"),
                        },
                    },
                },
            },
        }))
    })
})
