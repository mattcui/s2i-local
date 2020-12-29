package constants

const (
	// ImageDigestResult is the name of the Tekton result that is expected
	// to surface the digest of a singular image produces by a given task
	// or pipeline.  This digest is NOT fully qualified, so it should have
	// the form: sha256:deadbeef   (shortened for brevity)
	//
	// Generally this input string is paired with a parameter that directs
	// the task to publish the image to a particular tag, e.g.
	//   ghcr.io/mattmoor/mink-images:latest
	//
	// So the fully qualified digest may be assembled by concatenating these
	// with an @:
	//   ghcr.io/mattmoor/mink-images:latest@sha256:deadbeef
	ImageDigestResult = "mink-image-digest"
)
