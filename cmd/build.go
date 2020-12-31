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
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"os"
	"path/filepath"

	"github.com/davecgh/go-spew/spew"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/mattmoor/mink/pkg/kontext"

	buildClient "github.com/shipwright-io/build/pkg/client/build/clientset/versioned"
	"github.com/spf13/viper"
	"github.ibm.com/cuixuex/s2i-local/pkg/builds/buildpacks"
	"github.ibm.com/cuixuex/s2i-local/pkg/builds/dockerfile"
	"io"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"knative.dev/pkg/injection"
	"knative.dev/pkg/signals"
	"time"

	"github.com/mattmoor/mink/pkg/command"
	"github.com/spf13/cobra"
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

	Name string

	SecretName string
}

const activityTimeout = 30 * time.Second
const SourceImageSuffix = "_source"

// Validate implements Interface
func (opts *BuildOptions) Validate(cmd *cobra.Command, args []string) error {
	_ = viper.BindPFlags(cmd.Flags())
	opts.ImageName = viper.GetString("image")

	opts.Name = viper.GetString("name")
	if opts.Name == "" {
		return errors.New("image url is required")
	} else {
		opts.Name = viper.GetString("name")
	}

	// See if we're in "kontext mode"
	opts.Directory = viper.GetString("directory")

	opts.SecretName = viper.GetString("secret")
	if opts.SecretName == "" {
		opts.SecretName = "icr-knbuild"
	}

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
	cmd.Flags().StringP("secret", "e", "icr-knbuild", "The secret used to push target image and pull bundled source code image")

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

	spew.Dump("name: ", opts.Name)
	spew.Dump("secret: ", opts.SecretName)
	spew.Dump("imageURL: ", opts.ImageName)
	// Run the produced Build definition to completion, streaming logs to stdout, and
	// returng the digest of the produced image.
	var digest name.Digest
	if opts.Strategy == "kaniko" {
		err = opts.build(ctx, sourceDigest, cmd.OutOrStderr())
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

func (opts *BuildOptions) build(ctx context.Context, sourceDigest name.Digest, w io.Writer) error {
	//tag, err := name.NewTag(opts.ImageName, name.WeakValidation)
	//if err != nil {
	//	return err
	//}

	// Create a ClusterBuildStrategy definition
	cbs := dockerfile.ClusterBuildStrategy()
	spew.Dump("cbs: ", cbs)

	// Create a Build definition
	b := dockerfile.Build(dockerfile.Options{
		Name:       opts.Name,
		ImageURL:   opts.ImageName,
		SecretName: opts.SecretName,
	})
	b.Namespace = command.Namespace()

	// Create a BuildRun definition
	br := dockerfile.BuildRun(opts.Name)
	br.Namespace = command.Namespace()

	var BuildClientSet *buildClient.Clientset
	BuildClientSet, err := NewClient()
	if err != nil {
		return err
	}
	// Create a ClusterBuildStrategy object
	cbsInterface := BuildClientSet.BuildV1alpha1().ClusterBuildStrategies()
	spew.Dump("cbsInterface: ", cbsInterface)
	clusterBuildStrategy, err := cbsInterface.Create(context.TODO(), cbs, metav1.CreateOptions{})
	spew.Dump("ClusterBuildStrategy: ", clusterBuildStrategy)
	if err != nil {
		return err
	}

	// Create a Build object
	bInterface := BuildClientSet.BuildV1alpha1().Builds(b.Namespace)
	build, err := bInterface.Create(ctx, b, metav1.CreateOptions{})
	spew.Dump("Build: ", build)
	if err != nil {
		return err
	}

	// Create a BuildRun object
	brInterface := BuildClientSet.BuildV1alpha1().BuildRuns(br.Namespace)
	buildRun, err := brInterface.Create(ctx, br, metav1.CreateOptions{})
	spew.Dump("BuildRun: ", buildRun)
	if err != nil {
		return err
	}

	return nil
}

func (opts *BuildOptions) buildpack(ctx context.Context, sourceDigest name.Digest, w io.Writer) (name.Digest, error) {
	tag, err := name.NewTag(opts.ImageName, name.WeakValidation)
	if err != nil {
		return name.Digest{}, err
	}

	// Create a Build definition for turning the source into an image via CNCF Buildpacks.
	tr := buildpacks.Build(ctx, sourceDigest, tag)
	tr.Namespace = command.Namespace()

	return name.Digest{}, nil
}

// NewClient return a build Client
func NewClient() (*buildClient.Clientset, error) {
	_, restConfig, err := KubeConfig()
	if err != nil {
		return nil, err
	}

	buildClientSet, err := buildClient.NewForConfig(restConfig)
	if err != nil {
		return nil, err
	}

	return buildClientSet, nil
}

// KubeConfig returns all required clients to speak with
// the k8s API
func KubeConfig() (*kubernetes.Clientset, *rest.Config, error) {
	location := os.Getenv("KUBECONFIG")
	if location == "" {
		location = filepath.Join(os.Getenv("HOME"), ".kube", "config")
	}

	config, err := clientcmd.BuildConfigFromFlags("", location)
	if err != nil {
		config, err = rest.InClusterConfig()
		if err != nil {
			return nil, nil, err
		}
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, nil, err
	}

	return clientset, config, nil
}
