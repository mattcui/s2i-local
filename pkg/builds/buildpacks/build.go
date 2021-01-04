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

package buildpacks

import (
	"github.com/ghodss/yaml"
	buildv1alpha1 "github.com/shipwright-io/build/pkg/apis/build/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	// BuildpackBuildStrategyString holds the raw definition of the Buildpack strategy.
	// We export this into ./examples/buildpack.yaml
	BuildpackBuildStrategyString = `
---
apiVersion: build.dev/v1alpha1
kind: ClusterBuildStrategy
metadata:
  name: buildpacks-local
spec:
  buildSteps:
    - name: prepare
      image: docker.io/paketobuildpacks/builder:full
      securityContext:
        runAsUser: 0
        capabilities:
          add: 
            - CHOWN
      command:
        - /bin/bash
      args:
        - -c
        - >
          chown -R "1000:1000" /workspace/source &&
          chown -R "1000:1000" /tekton/home &&
          chown -R "1000:1000" /cache &&
          chown -R "1000:1000" /layers
      resources:
        limits:
          cpu: 500m
          memory: 1Gi
        requests:
          cpu: 250m
          memory: 65Mi
      volumeMounts:
        - name: cache-dir
          mountPath: /cache
        - name: layers-dir
          mountPath: /layers
    - name: extract-bundle
      image: $(build.output.image)_source
      workingDir: /workspace
      securityContext:
        runAsUser: 1000
        runAsGroup: 1000
    - name: build-and-push
      image: docker.io/paketobuildpacks/builder:full
      securityContext:
        runAsUser: 1000
        runAsGroup: 1000
      command:
        - /cnb/lifecycle/creator
      args:
        - -app=/workspace
        - -cache-dir=/cache
        - -layers=/layers
        - $(build.output.image)
      resources:
        limits:
          cpu: 500m
          memory: 1Gi
        requests:
          cpu: 250m
          memory: 65Mi
      volumeMounts:
        - name: cache-dir
          mountPath: /cache
        - name: layers-dir
          mountPath: /layers
`

	// BuildpackTask is the parsed form of BuildpackTaskString.
	BuildpackBuildStrategy buildv1alpha1.ClusterBuildStrategy
)

// Options holds configuration options specific to Dockerfile builds
type Options struct {
	// Build name
	Name string

	// Target image
	ImageURL string

	// secret which is used to push target image
	SecretName string
}

func init() {
	if err := yaml.Unmarshal([]byte(BuildpackBuildStrategyString), &BuildpackBuildStrategy); err != nil {
		panic(err)
	}
}

// BuildpackClusterBuildStrategy returns a ClusterBuildStrategy object for performing a Buildpacks local build.
func BuildpackClusterBuildStrategy() *buildv1alpha1.ClusterBuildStrategy {
	return &buildv1alpha1.ClusterBuildStrategy{
		ObjectMeta: metav1.ObjectMeta{
			Name: "buildpacks-local",
		},
		Spec: *BuildpackBuildStrategy.Spec.DeepCopy(),
	}
}
