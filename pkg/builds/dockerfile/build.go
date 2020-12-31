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
	"github.com/ghodss/yaml"
	buildv1alpha1 "github.com/shipwright-io/build/pkg/apis/build/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	// KanikoBuildStrategyString holds the raw definition of the Kaniko-local buildStrategy.
	// We export this into ./examples/kaniko.yaml
	KanikoBuildStrategyString = `
--- 
apiVersion: build.dev/v1alpha1
kind: ClusterBuildStrategy
metadata: 
  name: kaniko-local
spec: 
  buildSteps: 
    - 
      image: $(build.output.image)_source
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
        - "--skip-tls-verify=true"
        - "--dockerfile=/workspace/Dockerfile"
        - "--context=/workspace"
        - "--destination=$(build.output.image)"
        - "--oci-layout-path=/workspace/output/image"
        - "--snapshotMode=redo"
      command: 
        - /kaniko/executor
      env: 
        - 
          name: DOCKER_CONFIG
          value: /tekton/home/.docker
        - 
          name: AWS_ACCESS_KEY_ID
          value: NOT_SET
        - 
          name: AWS_SECRET_KEY
          value: NOT_SET
      image: "gcr.io/kaniko-project/executor:v1.3.0"
      name: build-and-push
      resources: 
        limits: 
          cpu: 500m
          memory: 1Gi
        requests: 
          cpu: 250m
          memory: 65Mi
      securityContext: 
        capabilities: 
          add: 
            - CHOWN
            - DAC_OVERRIDE
            - FOWNER
            - SETGID
            - SETUID
            - SETFCAP
        runAsUser: 0
      workingDir: /workspace
`
	// KanikoBuildStrategy is the parsed form of KanikoTaskString.
	KanikoBuildStrategy buildv1alpha1.ClusterBuildStrategy
)

// Options holds configuration options specific to Dockerfile builds
type Options struct {
	// Dockerfile is the path to the Dockerfile within the build context.
	//Dockerfile string
	//
	//// The extra kaniko arguments for handling things like insecure registries
	//KanikoArgs []string

	// Build name
	Name string

	// Target image
	ImageURL string

	// secret which is used to push target image
	SecretName string
}

func init() {
	if err := yaml.Unmarshal([]byte(KanikoBuildStrategyString), &KanikoBuildStrategy); err != nil {
		panic(err)
	}
}

// ClusterBuildStrategy returns a ClusterBuildStrategy object for performing a Kaniko local build.
func ClusterBuildStrategy() *buildv1alpha1.ClusterBuildStrategy {
	return &buildv1alpha1.ClusterBuildStrategy{
		ObjectMeta: metav1.ObjectMeta{
			Name: "kaniko-local",
		},
		Spec: *KanikoBuildStrategy.Spec.DeepCopy(),
	}
}

// Build returns a Build object for performing a Dockerfile build over the
// provided source and publishing to the target tag.
func Build(opt Options) *buildv1alpha1.Build {
	buildStrategy := buildv1alpha1.ClusterBuildStrategyKind

	return &buildv1alpha1.Build{
		ObjectMeta: metav1.ObjectMeta{
			Name: opt.Name,
		},
		Spec: buildv1alpha1.BuildSpec{
			Source: buildv1alpha1.GitSource{
				URL: "https://github.com/zhangtbj/empty-for-local-build",
			},
			StrategyRef: &buildv1alpha1.StrategyRef{
				Name: "kaniko-local",
				Kind: &buildStrategy,
			},
			Output: buildv1alpha1.Image{
				ImageURL: opt.ImageURL,
				SecretRef: &corev1.LocalObjectReference{
					Name: opt.SecretName,
				},
			},
		},
	}
}

// BuildRun returns a BuildRun object
func BuildRun(buildName string) *buildv1alpha1.BuildRun {
	return &buildv1alpha1.BuildRun{
		ObjectMeta: metav1.ObjectMeta{
			Name: buildName + "buildrun",
		},
		Spec: buildv1alpha1.BuildRunSpec{
			BuildRef: &buildv1alpha1.BuildRef{
				Name: buildName,
			},
			ServiceAccount: &buildv1alpha1.ServiceAccount{
				Generate: true,
			},
		},
	}
}
