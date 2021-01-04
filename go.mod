module github.ibm.com/cuixuex/s2i-local

go 1.15

require (
	github.com/davecgh/go-spew v1.1.1
	github.com/ghodss/yaml v1.0.1-0.20190212211648-25d852aebe32
	github.com/google/go-cmp v0.5.4 // indirect
	github.com/google/go-containerregistry v0.3.0
	github.com/mattmoor/mink v0.19.0
	github.com/mitchellh/go-homedir v1.1.0
	github.com/shipwright-io/build v0.2.0
	github.com/spf13/cobra v1.1.1
	github.com/spf13/viper v1.7.1
	github.com/stretchr/testify v1.6.1 // indirect
	github.com/tektoncd/cli v0.15.0 // indirect
	golang.org/x/sys v0.0.0-20201201145000-ef89a241ccb3 // indirect
	golang.org/x/text v0.3.4 // indirect
	k8s.io/api v0.19.2
	k8s.io/apimachinery v0.19.2
	k8s.io/client-go v12.0.0+incompatible
	knative.dev/pkg v0.0.0-20201103163404-5514ab0c1fdf
)

replace github.com/spf13/cobra => github.com/chmouel/cobra v0.0.0-20200107083527-379e7a80af0c

replace (
	k8s.io/api => k8s.io/api v0.18.8
	k8s.io/client-go => k8s.io/client-go v0.18.8
	k8s.io/kube-openapi => k8s.io/kube-openapi v0.0.0-20200410145947-61e04a5be9a6
)
