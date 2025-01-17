package commands

import (
	"fmt"
	"github.com/hanochg/piperika/terminal"
	"github.com/hanochg/piperika/utils"
	"github.com/jfrog/jfrog-cli-core/plugins/components"
)

func GetLinkCommand() components.Command {
	return components.Command{
		Name:        "link",
		Description: "Get Pipelines Link",
		Aliases:     []string{"l"},
		Arguments:   getArguments(),
		Flags:       getFlags(),
		Action:      getPipelinesLink,
	}
}

func getPipelinesLink(c *components.Context) error {
	dirConfig, err := utils.GetDirConfig()
	if err != nil {
		return err
	}

	uiUrl, err := utils.GetUIBaseUrl(c)
	if err != nil {
		return err
	}
	branchName, err := utils.GetCurrentBranchName(c)
	if err != nil {
		return err
	}
	link := fmt.Sprintf("%s ",
		utils.GetPipelinesBranchURL(uiUrl, dirConfig.PipelineName, branchName))
	return terminal.DoneMessage("Link", "", link)
}
