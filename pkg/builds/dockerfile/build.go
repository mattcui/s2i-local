/*
Copyright 2020 The Knative Authors

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

package dockerfile

import (
	"context"

	"github.com/ghodss/yaml"
	"github.com/google/go-containerregistry/pkg/name"
	tknv1beta1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	"github.ibm.com/cuixuex/s2i-local/pkg/constants"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"knative.dev/pkg/ptr"
)

var (
	// KanikoTaskString holds the raw definition of the Kaniko task.
	// We export this into ./examples/kaniko.yaml
	KanikoTaskString = `
--- 
apiVersion: tekton.dev/v1beta1
kind: Task
metadata:
  name: kaniko
spec:
  description: "An example kaniko task illustrating some of the parameter processing."
  params: 
    - 
      description: "A self-extracting container image of source"
      name: source-bundle
    - 
      description: "Where to publish an image."
      name: image-target

  results: 
    - 
      description: "The digest of the resulting image."
      name: image-digest
  steps: 
    - 
      image: $(params.source-bundle)
      name: extract-bundle
      workingDir: /workspace
    - 
      command: 
        - sh
        - "-c"
        - "echo \"Hello, Kubernetes!\" && sleep 15"
      image: busybox
      name: debug
    - 
      args: 
        - "--dockerfile=/workspace/Dockerfile"
        - "--context=/workspace"
        - "--destination=$(params.image-target)"
        - "--digest-file=/tekton/results/image-digest"
        - "--cache=true"
        - "--cache-ttl=24h"
      env: 
        - 
          name: DOCKER_CONFIG
          value: /tekton/home/.docker
      image: "gcr.io/kaniko-project/executor:multi-arch"
      name: build-and-push
`
	// KanikoTask is the parsed form of KanikoTaskString.
	KanikoTask tknv1beta1.Task
)

func init() {
	if err := yaml.Unmarshal([]byte(KanikoTaskString), &KanikoTask); err != nil {
		panic(err)
	}
}

// Build returns a TaskRun suitable for performing a Dockerfile build over the
// provided source and publishing to the target tag.
func Build(ctx context.Context, source name.Reference, target name.Tag) *tknv1beta1.TaskRun {
	return &tknv1beta1.TaskRun{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "dockerfile-",
		},
		Spec: tknv1beta1.TaskRunSpec{
			PodTemplate: &tknv1beta1.PodTemplate{
				EnableServiceLinks: ptr.Bool(false),
			},
			TaskSpec: KanikoTask.Spec.DeepCopy(),
			Params: []tknv1beta1.Param{{
				Name:  constants.SourceBundleParam,
				Value: *tknv1beta1.NewArrayOrString(source.String()),
			}, {
				Name:  constants.ImageTargetParam,
				Value: *tknv1beta1.NewArrayOrString(target.String()),
			}},
		},
	}
}

