package engine

import (
	"errors"

	"code.cloudfoundry.org/lager"
	"github.com/concourse/concourse/atc"
	"github.com/concourse/concourse/atc/db"
)

type execV1DummyEngine struct{}

const execV1DummyEngineSchema = "exec.v1"

func NewExecV1DummyEngine() Engine {
	return execV1DummyEngine{}
}

func (execV1DummyEngine) Schema() string {
	return execV1DummyEngineSchema
}

func (execV1DummyEngine) CreateBuild(logger lager.Logger, build db.Build, plan atc.Plan) (Build, error) {
	return nil, errors.New("dummy engine does not support new builds")
}

func (execV1DummyEngine) LookupBuild(logger lager.Logger, build db.Build) (Build, error) {
	return execV1DummyBuild{}, nil
}

func (execV1DummyEngine) ReleaseAll(lager.Logger) {
}

type execV1DummyBuild struct {
}

func (execV1DummyBuild) Metadata() string {
	return ""
}

func (execV1DummyBuild) PublicPlan(lager.Logger) (atc.PublicBuildPlan, error) {
	return atc.PublicBuildPlan{
		Schema: execV1DummyEngineSchema,
		Plan:   nil,
	}, nil
}

func (execV1DummyBuild) Abort(lager.Logger) error {
	return nil
}

func (execV1DummyBuild) Resume(logger lager.Logger) {
}
