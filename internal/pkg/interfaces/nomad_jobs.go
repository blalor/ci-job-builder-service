package interfaces

import (
	"github.com/hashicorp/nomad/api"
)

type NomadJobs interface {
	Register(job *api.Job, q *api.WriteOptions) (*api.JobRegisterResponse, *api.WriteMeta, error)
}
