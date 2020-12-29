module github.ibm.com/cuixuex/s2i-local

go 1.15

require (
	github.com/google/go-cmp v0.5.4 // indirect
	github.com/google/go-containerregistry v0.3.0
	github.com/mattmoor/mink v0.19.0
	github.com/mitchellh/go-homedir v1.1.0
	github.com/spf13/cobra v1.1.1
	github.com/spf13/viper v1.7.1
	github.com/stretchr/testify v1.6.1 // indirect
	golang.org/x/sys v0.0.0-20201201145000-ef89a241ccb3 // indirect
	golang.org/x/text v0.3.4 // indirect
)

replace k8s.io/client-go => k8s.io/client-go v0.18.8
