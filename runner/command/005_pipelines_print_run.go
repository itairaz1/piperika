package command

import (
	"context"
	"fmt"
	"github.com/hanochg/piperika/http"
	"github.com/hanochg/piperika/http/models"
	"github.com/hanochg/piperika/http/requests"
	"github.com/hanochg/piperika/utils"
	"strconv"
)

func New005PipelinesPrintRun() *_005 {
	return &_005{}
}

type _005 struct{}

func (c *_005) ResolveState(ctx context.Context, state *PipedCommandState) (Status, error) {
	httpClient := ctx.Value(utils.HttpClientCtxKey).(http.PipelineHttpClient)

	// Get steps statuses
	steps, err := requests.GetSteps(httpClient, models.GetStepsOptions{
		// TODO: shouldn't this also have run number? otherwise it's the steps of the last run and not the intended run
		RunIds: strconv.Itoa(state.RunId),
		Limit:  0,
	})
	if err != nil {
		return Status{}, err
	}

	failedSteps := make([]string, 0)
	successSteps := make([]string, 0)
	processingSteps := make([]string, 0)
	for _, step := range steps.Steps {
		if step.StatusCode == models.Failure {
			failedSteps = append(failedSteps, step.Name)
		}
		if step.StatusCode == models.Success {
			successSteps = append(successSteps, step.Name)
		}
		if step.StatusCode == models.Waiting ||
			step.StatusCode == models.Processing {
			processingSteps = append(processingSteps, step.Name)
		}
	}

	if len(processingSteps) != 0 {
		return Status{}, fmt.Errorf("run %d has %d steps. currently %d are processing, %d failed, and %d succeeded",
			state.RunNumber, len(steps.Steps), len(processingSteps), len(failedSteps), len(successSteps))
	}

	_, err = requests.GetStepsTestReports(httpClient, models.StepsTestReportsOptions{StepIds: state.RunStepIdsCsv})
	if err != nil {
		return Status{}, err

	}
	// TODO - print the tests results

	return Status{
		Message: fmt.Sprintf("run %d has %d steps. %d failed, and %d succeeded",
			state.RunNumber, len(steps.Steps), len(failedSteps), len(successSteps)),
	}, nil
}

func (c *_005) TriggerStateChange(ctx context.Context, state *PipedCommandState) error {
	// do nothing
	return nil
}