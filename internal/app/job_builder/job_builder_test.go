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

    type submitPayload struct {
        JobSpec string `json:"job_spec"`
        SourceArchive string `json:"source_archive"`
    }

    BeforeEach(func() {
        router = mux.NewRouter()
        resp = httptest.NewRecorder()

        mockNomadJobs = interfaces.MockNomadJobs{}

        handler = NewJobBuilder(&mockNomadJobs)
        handler.InstallHandlers(router.PathPrefix("/").Subrouter())
    })

    It("builds and submits a job", func() {
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

        requestBody, _ := json.Marshal(map[string]string{
            "job_spec": `
driver: docker
config:
    image: golang
    command: build.sh
`,
            "source_archive": "s3.amazonaws.com/bucket/archive.tar.gz",
        })

        req, err := http.NewRequest("POST", endpoint, bytes.NewReader(requestBody))
        Expect(err).ShouldNot(HaveOccurred())

        req.Header.Add("Content-Type", "application/json")

        router.ServeHTTP(resp, req)
        Expect(resp.Code).To(Equal(http.StatusNoContent))

        mockNomadJobs.AssertExpectations(GinkgoT())

        job := mockNomadJobs.Calls[0].Arguments[0].(*nomadapi.Job)

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

            Tasks: []*nomadapi.Task{
                &nomadapi.Task{
                    Name: "builder",

                    Driver: "docker",
                    Config: map[string]interface{}{
                        "image": "golang",
                        "command": "build.sh",

                        "work_dir": "${NOMAD_TASK_DIR}/work",
                    },

                    Artifacts: []*nomadapi.TaskArtifact{
                        &nomadapi.TaskArtifact{
                            GetterSource: StringToPtr("s3.amazonaws.com/bucket/archive.tar.gz"),
                            RelativeDest: StringToPtr("local/work"),
                        },
                    },
                },
            },
        }))

    })
})
