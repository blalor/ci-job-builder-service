package job_builder

import (
    "fmt"
    "time"
    "io/ioutil"
    "net"
    "net/http"

    log "github.com/Sirupsen/logrus"

    "github.com/gorilla/mux"

    nomadapi "github.com/hashicorp/nomad/api"
    "github.com/nomad-ci/ci-job-builder-service/internal/pkg/interfaces"

    "encoding/json"
    "github.com/ghodss/yaml"
)

type JobSpec struct {
    Driver string `json:"driver"`
    Config map[string]interface{} `json:"config"`
}

type JobBuilderPayload struct {
    JobSpec       string `json:"job_spec"`
    SourceArchive string `json:"source_archive"`
}

type JobBuilder struct {
    nomad interfaces.NomadJobs
}

func NewJobBuilder(nomad interfaces.NomadJobs) *JobBuilder {
    return &JobBuilder{
        nomad: nomad,
    }
}

func (self *JobBuilder) InstallHandlers(router *mux.Router) {
    router.
        Methods("POST").
        Path("/build-job").
        Headers(
            "Content-Type", "application/json",
        ).
        HandlerFunc(self.BuildJob)
}

func (self *JobBuilder) BuildJob(resp http.ResponseWriter, req *http.Request) {
    var err error
    var remoteAddr string

    if xff, ok := req.Header["X-Forwarded-For"]; ok {
        remoteAddr = xff[0]
    } else {
        remoteAddr, _, err = net.SplitHostPort(req.RemoteAddr)
        if err != nil {
            log.Warnf("unable to parse RemoteAddr '%s': %s", remoteAddr, err)
            remoteAddr = req.RemoteAddr
        }
    }

    logEntry := log.WithField("remote_ip", remoteAddr)

    body, err := ioutil.ReadAll(req.Body)
    if err != nil {
        logEntry.Errorf("unable to read body: %s", err)
        resp.WriteHeader(http.StatusBadRequest)
        return
    }

    var payload JobBuilderPayload
    err = json.Unmarshal(body, &payload)
    if err != nil {
        logEntry.Errorf("unable to unmarshal body: %s", err)
        resp.WriteHeader(http.StatusBadRequest)
        return
    }

    var jobSpec JobSpec
    err = yaml.Unmarshal([]byte(payload.JobSpec), &jobSpec)
    if err != nil {
        logEntry.Errorf("unable to unmarshal job spec: %s", err)
        resp.WriteHeader(http.StatusBadRequest)
        return
    }

    jobId := fmt.Sprintf("ci-job/%d", time.Now().Unix())

    // @todo only allow a limited set of config params, based on driver
    jobSpec.Config["work_dir"] = "${NOMAD_TASK_DIR}/work"

    job := &nomadapi.Job{
        ID: StringToPtr(jobId),
        Name: StringToPtr(jobId),

        Datacenters: []string{"dc1"},

        Type: StringToPtr("batch"),

        TaskGroups: []*nomadapi.TaskGroup{
            &nomadapi.TaskGroup{
                Name: StringToPtr("builder"),

                Tasks: []*nomadapi.Task{
                    &nomadapi.Task{
                        Name: "builder",

                        Driver: jobSpec.Driver,
                        Config: jobSpec.Config,

                        Artifacts: []*nomadapi.TaskArtifact{
                            &nomadapi.TaskArtifact{
                                GetterSource: StringToPtr(payload.SourceArchive),
                                RelativeDest: StringToPtr("local/work"),
                            },
                        },
                    },
                },
            },
        },

    }

    jobResp, _, err := self.nomad.Register(job, nil)
    if err != nil {
        logEntry.Errorf("unable to submit job: %s", err)
        resp.WriteHeader(http.StatusInternalServerError)
        return
    }

    logEntry.Infof("submitted job with eval id %s", jobResp.EvalID)

    resp.WriteHeader(http.StatusNoContent)
}
