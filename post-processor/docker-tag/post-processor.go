package dockertag

import (
	"fmt"

	"github.com/hashicorp/packer/builder/docker"
	"github.com/hashicorp/packer/common"
	"github.com/hashicorp/packer/helper/config"
	"github.com/hashicorp/packer/packer"
	"github.com/hashicorp/packer/post-processor/docker-import"
	"github.com/hashicorp/packer/template/interpolate"
)

const BuilderId = "packer.post-processor.docker-tag"

type Config struct {
	common.PackerConfig `mapstructure:",squash"`

	Repository string `mapstructure:"repository"`
	Tag        string `mapstructure:"tag"`
	Force      bool

	ctx interpolate.Context
}

type PostProcessor struct {
	Driver docker.Driver

	config Config
}

func (p *PostProcessor) Configure(raws ...interface{}) error {
	err := config.Decode(&p.config, &config.DecodeOpts{
		Interpolate:        true,
		InterpolateContext: &p.config.ctx,
		InterpolateFilter: &interpolate.RenderFilter{
			Exclude: []string{},
		},
	}, raws...)
	if err != nil {
		return err
	}

	return nil

}

func (p *PostProcessor) PostProcess(ui packer.Ui, artifact packer.Artifact) (packer.Artifact, bool, bool, error) {
	if artifact.BuilderId() != BuilderId &&
		artifact.BuilderId() != dockerimport.BuilderId {
		err := fmt.Errorf(
			"Unknown artifact type: %s\nCan only tag from Docker builder artifacts.",
			artifact.BuilderId())
		return nil, false, true, err
	}

	driver := p.Driver
	if driver == nil {
		// If no driver is set, then we use the real driver
		driver = &docker.DockerDriver{Ctx: &p.config.ctx, Ui: ui}
	}

	importRepo := p.config.Repository
	if p.config.Tag != "" {
		importRepo += ":" + p.config.Tag
	}

	ui.Message("Tagging image: " + artifact.Id())
	ui.Message("Repository: " + importRepo)
	err := driver.TagImage(artifact.Id(), importRepo, p.config.Force)
	if err != nil {
		return nil, false, true, err
	}

	// Build the artifact
	artifact = &docker.ImportArtifact{
		BuilderIdValue: BuilderId,
		Driver:         driver,
		IdValue:        importRepo,
	}

	// If we tag an image and then delete it, there was no point in creating the
	// tag. Override users to force us to always keep the input artifact.
	return artifact, true, true, nil
}
