package constants

const (
	// SourceBundleParam is the name of the Tekton parameter that is
	// expected to pass the fully-qualified URI of a container image that
	// when run unpacks its payload into the working directory in which
	// it was invoked.
	SourceBundleParam = "mink-source-bundle"

	// ImageTargetParam is the name of the Tekton parameter that is
	// expected to pass the fully-qualified URI for where to publish
	// a container image.
	ImageTargetParam = "mink-image-target"
)
