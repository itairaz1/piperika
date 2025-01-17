package command

import (
	"context"
	"fmt"
	"github.com/hanochg/piperika/http"
	"github.com/hanochg/piperika/http/requests"
	"github.com/hanochg/piperika/utils"
	"strconv"
	"strings"
)

func New003PipelinesFindRun() *_003 {
	return &_003{}
}

type _003 struct {
	runTriggered bool
}

func (c *_003) ResolveState(ctx context.Context, state *PipedCommandState) Status {
	httpClient := ctx.Value(utils.HttpClientCtxKey).(http.PipelineHttpClient)
	dirConfig := ctx.Value(utils.DirConfigCtxKey).(*utils.DirConfig)
	branchName := ctx.Value(utils.BranchName).(string)

	pipeResp, err := requests.GetPipelines(httpClient, requests.GetPipelinesOptions{
		SortBy:     "latestRunId",
		FilterBy:   branchName,
		Light:      true,
		PipesNames: dirConfig.PipelineName,
	})
	if err != nil {
		return Status{
			Type:    Unrecoverable,
			Message: fmt.Sprintf("Failed fetching pipelines data: %v", err),
		}
	}
	if len(pipeResp.Pipelines) == 0 {
		return Status{
			Type:            InProgress,
			PipelinesStatus: "missing pipeline",
			Message:         fmt.Sprintf("waiting for pipeline '%s' creation", dirConfig.PipelineName),
		}
	}
	state.PipelineId = pipeResp.Pipelines[0].PipelineId

	runResp, err := requests.GetRuns(httpClient, requests.GetRunsOptions{
		PipelineIds: strconv.Itoa(state.PipelineId),
		Limit:       10,
		Light:       true,
		StatusCodes: fmt.Sprintf("%s,%s,%s,%s", http.Ready.String(), http.Creating.String(), http.Waiting.String(), http.Processing.String()),
		SortBy:      "runNumber",
		SortOrder:   -1,
	})
	if err != nil {
		return Status{
			Type:    Unrecoverable,
			Message: fmt.Sprintf("failed fetching pipeline runs data: %v", err),
		}
	}
	if len(runResp.Runs) == 0 {
		return Status{
			PipelinesStatus: "there are no active relevant runs",
			Message:         "waiting for run creation",
			Type:            InProgress,
		}
	}

	runIds := make([]string, 0)
	runNumbers := make([]string, 0)
	for _, run := range runResp.Runs {
		runIds = append(runIds, strconv.Itoa(run.RunId))
		runNumbers = append(runNumbers, strconv.Itoa(run.RunNumber))
	}
	runResourceResp, err := requests.GetRunResourceVersions(httpClient, requests.GetRunResourcesOptions{
		PipelineSourceIds: strconv.Itoa(state.PipelinesSourceId),
		RunIds:            strings.Trim(strings.Join(runIds, ","), "[]"),
		SortBy:            "resourceTypeCode",
		SortOrder:         1,
	})
	if err != nil {
		return Status{
			Type:    InProgress,
			Message: fmt.Sprintf("failed fetching run resources data: %v", err),
		}
	}

	if len(runResourceResp.Resources) == 0 {
		return Status{
			Type:            InProgress,
			PipelinesStatus: "triggering new run",
			Message:         "no resources exist for the resolved pipeline run",
		}
	}

	activeRunIds := -1
	for _, runResource := range runResourceResp.Resources {
		if runResource.ResourceTypeCode != http.GitRepo {
			continue
		}
		if runResource.ResourceVersionContentPropertyBag.CommitSha == state.HeadCommitSha {
			activeRunIds = runResource.RunId
			break
		}
	}

	// Get the most recent run from the list
	for i, runIdStr := range runIds {
		runId, err := strconv.Atoi(runIdStr)
		if err != nil {
			return Status{
				Type:            Failed,
				PipelinesStatus: "triggering new run",
				Message:         fmt.Sprintf("corrupted data for the resolved pipeline run, err %v", err),
			}
		}
		runNumber, err := strconv.Atoi(runNumbers[i])
		if err != nil {
			return Status{
				Type:            Failed,
				PipelinesStatus: "triggering new run",
				Message:         fmt.Sprintf("corrupted data for the resolved pipeline run, err %v", err),
			}
		}
		if activeRunIds == runId {
			state.RunId = runId
			state.RunNumber = runNumber
			break
		}
	}

	msg := fmt.Sprintf("Run #%d was found", state.RunNumber)
	if c.runTriggered {
		msg = fmt.Sprintf("Run #%d was triggered", state.RunNumber)
	}
	if activeRunIds != -1 && state.RunId != -1 {
		return Status{
			Message: msg,
			Type:    Done,
		}
	}

	return Status{
		Type:            InProgress,
		PipelinesStatus: "Not exists",
		Message:         "did not find any active runs",
	}
}

func (c *_003) TriggerOnFail(ctx context.Context, state *PipedCommandState) error {
	httpClient := ctx.Value(utils.HttpClientCtxKey).(http.PipelineHttpClient)
	dirConfig := ctx.Value(utils.DirConfigCtxKey).(*utils.DirConfig)

	pipeSteps, err := requests.GetPipelinesSteps(httpClient, requests.GetPipelinesStepsOptions{
		PipelineIds:       strconv.Itoa(state.PipelineId),
		PipelineSourceIds: strconv.Itoa(state.PipelinesSourceId),
		Names:             dirConfig.DefaultStep,
	})
	if err != nil {
		return fmt.Errorf("failed fetching pipeline steps: %w", err)
	}
	if len(pipeSteps.Steps) == 0 {
		return fmt.Errorf("no pipeline step called '%s'", dirConfig.DefaultStep)
	}
	c.runTriggered = true
	return requests.TriggerPipelinesStep(httpClient, pipeSteps.Steps[0].Id)
}
