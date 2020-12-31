/*
Copyright © 2020 NAME HERE <EMAIL ADDRESS>

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package cmd

import (
	"context"
	"errors"
	"fmt"
	"github.com/davecgh/go-spew/spew"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/mattmoor/mink/pkg/kontext"
	"github.com/spf13/viper"
	"github.com/tektoncd/cli/pkg/cli"
	"github.ibm.com/cuixuex/s2i-local/pkg/builds"
	"github.ibm.com/cuixuex/s2i-local/pkg/builds/buildpacks"
	"github.ibm.com/cuixuex/s2i-local/pkg/builds/dockerfile"
	"io"
	"knative.dev/pkg/injection"
	"knative.dev/pkg/signals"
	"time"

	"github.com/mattmoor/mink/pkg/command"
	"github.com/spf13/cobra"
	"github.com/tektoncd/cli/pkg/options"
	_ "k8s.io/client-go/plugin/pkg/client/auth/oidc"
)

func NewBuildCommand(ctx context.Context) *cobra.Command {
	opts := &BuildOptions{
		ctx: ctx,
	}

	// buildCmd represents the build command
	buildCmd := &cobra.Command{
		Use:   "build",
		Short: "A brief description of your command",
		Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly creatree a Cobra application.`,
		PreRunE: opts.Validate,
		RunE:    opts.Execute,
	}

	opts.AddFlags(buildCmd)

	return buildCmd
}

func init() {
	// We do not start informers.
	ctx, _ := injection.EnableInjectionOrDie(signals.NewContext(), nil)
	rootCmd.AddCommand(NewBuildCommand(ctx))
}

// BuildOptions implements Interface for the `kn im build` command.
type BuildOptions struct {
	ctx context.Context

	// ImageName is the string name of the bundle image to which we should publish things.
	ImageName string

	// Directory is the string containing the directory to bundle.
	// This option signals "kontext mode".
	Directory string

	Strategy string

	// tag is the processed version of ImageName that is populated while validating it.
	tag name.Tag

	ServiceAccount string
}

const activityTimeout = 30 * time.Second
const SourceImageSuffix = "_source"

// Validate implements Interface
func (opts *BuildOptions) Validate(cmd *cobra.Command, args []string) error {
	_ = viper.BindPFlags(cmd.Flags())
	opts.ImageName = viper.GetString("image")

	// See if we're in "kontext mode"
	opts.Directory = viper.GetString("directory")

	opts.Strategy = viper.GetString("strategy")
	if opts.Strategy != "" && opts.Strategy != "kaniko" && opts.Strategy != "buildpack" {
		return errors.New("not supported strategy specified, only support 'kaniko' and 'buildpack'")
	}

	if opts.ImageName == "" {
		return errors.New("image url is required")
	} else if tag, err := name.NewTag(opts.ImageName, name.WeakValidation); err != nil {
		return err
	} else {
		opts.tag = tag
	}

	return nil
}

// AddFlags implements Interface
func (opts *BuildOptions) AddFlags(cmd *cobra.Command) {
	cmd.Flags().StringP("name", "n", "", "The build name")
	cmd.Flags().StringP("directory", "d", ".", "The directory to bundle up")
	cmd.Flags().StringP("image", "i", "", "The image to generate and publish")
	cmd.Flags().StringP("strategy", "s", "local", "The build strategy to build the image")

	_ = cmd.MarkPersistentFlagRequired("name")
	_ = cmd.MarkPersistentFlagRequired("image")
}

func (opts *BuildOptions) bundle(ctx context.Context) (name.Digest, error) {

	bundleTag, err := name.NewTag(opts.tag.RegistryStr() + "/" + opts.tag.RepositoryStr() + SourceImageSuffix + ":" + opts.tag.TagStr())
	if err != nil {
		return name.Digest{}, err
	}
	return kontext.Bundle(ctx, opts.Directory, bundleTag)
}

func (opts *BuildOptions) Execute(cmd *cobra.Command, args []string) error {
	if len(args) != 0 {
		return errors.New("'im bundle' does not take any arguments")
	}

	// Handle ctrl+C
	ctx := opts.GetContext(cmd)
	spew.Dump("ctx: %v", ctx)

	sourceDigest, err := opts.bundle(ctx)
	if err != nil {
		return err
	}

	// Run the produced Build definition to completion, streaming logs to stdout, and
	// returng the digest of the produced image.
	var digest name.Digest
	if opts.Strategy == "kaniko" {
		digest, err = opts.build(ctx, sourceDigest, cmd.OutOrStderr())
	} else if opts.Strategy == "buildpack" {
		digest, err = opts.buildpack(ctx, sourceDigest, cmd.OutOrStderr())
	}
	if err != nil {
		return err
	}

	fmt.Fprintf(cmd.OutOrStdout(), "%s\n", digest.String())
	return nil
}

// GetContext implements Interface
func (opts *BuildOptions) GetContext(cmd *cobra.Command) context.Context {
	return opts.ctx
}

func (opts *BuildOptions) build(ctx context.Context, sourceDigest name.Digest, w io.Writer) (name.Digest, error) {
	tag, err := name.NewTag(opts.ImageName, name.WeakValidation)
	if err != nil {
		return name.Digest{}, err
	}

	// Create a Build definition for turning the source into an image by Dockerfile build.
	tr := dockerfile.Build(ctx, sourceDigest, tag)
	tr.Namespace = command.Namespace()

	// Run the produced Build definition to completion, streaming logs to stdout, and
	// returning the digest of the produced image.
	return builds.Run(ctx, tag.String(), tr, &options.LogOptions{
		ActivityTimeout: activityTimeout,
		Params:          &cli.TektonParams{},
		Stream: &cli.Stream{
			// Send Out to stderr so we can capture the digest for composition.
			Out: w,
			Err: w,
		},
		Follow: true,
	}, builds.WithTaskServiceAccount(ctx, opts.ServiceAccount, tag, sourceDigest))
}

func (opts *BuildOptions) buildpack(ctx context.Context, sourceDigest name.Digest, w io.Writer) (name.Digest, error) {
	tag, err := name.NewTag(opts.ImageName, name.WeakValidation)
	if err != nil {
		return name.Digest{}, err
	}

	// Create a Build definition for turning the source into an image via CNCF Buildpacks.
	tr := buildpacks.Build(ctx, sourceDigest, tag)
	tr.Namespace = command.Namespace()

	// Run the produced Build definition to completion, streaming logs to stdout, and
	// returning the digest of the produced image.
	return builds.Run(ctx, tag.String(), tr, &options.LogOptions{
		ActivityTimeout: activityTimeout,
		Params:          &cli.TektonParams{},
		Stream: &cli.Stream{
			// Send Out to stderr so we can capture the digest for composition.
			Out: w,
			Err: w,
		},
		Follow: true,
	}, builds.WithTaskServiceAccount(ctx, opts.ServiceAccount, tag, sourceDigest))
}
