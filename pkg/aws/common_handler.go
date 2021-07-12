package aws

import (
	"fmt"
	"math/rand"
	"sort"
	"strings"
	"time"

	"github.com/blang/semver/v4"
	releasev1alpha1 "github.com/giantswarm/apiextensions/v3/pkg/apis/release/v1alpha1"
	"github.com/giantswarm/k8sclient/v5/pkg/k8sclient"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/giantswarm/aws-admission-controller/v2/pkg/label"
	"github.com/giantswarm/aws-admission-controller/v2/pkg/mutator"
)

type Handler struct {
	K8sClient k8sclient.Interface
	Logger    micrologger.Logger
}

func GetReleaseComponentLabels(release releasev1alpha1.Release) map[string]string {
	components := map[string]string{}
	for _, component := range release.Spec.Components {
		components[component.Name] = component.Version
	}
	return components
}

func GetNavailabilityZones(m *Handler, n int, azs []string) []string {
	randomAZs := azs
	// In case there are not enough distinct AZs, we repeat them
	for len(randomAZs) < n {
		randomAZs = append(randomAZs, azs...)
	}
	// We shuffle the AZs, pick the first n and sort them alphabetically
	rand.Seed(time.Now().UnixNano())
	rand.Shuffle(len(randomAZs), func(i, j int) { randomAZs[i], randomAZs[j] = randomAZs[j], randomAZs[i] })
	randomAZs = randomAZs[:n]
	sort.Strings(randomAZs)
	m.Logger.Log("level", "debug", "message", fmt.Sprintf("available AZ's: %v, selected AZ's: %v", azs, randomAZs))

	return randomAZs
}

func IsCAPIRelease(meta metav1.Object) (bool, error) {
	if meta.GetLabels()[label.Release] == "" {
		return false, nil
	}
	releaseVersion, err := ReleaseVersion(meta, []mutator.PatchOperation{})
	if err != nil {
		return false, microerror.Maskf(parsingFailedError, "unable to parse release version from object")
	}
	return IsCAPIVersion(releaseVersion)
}

func ReleaseVersion(meta metav1.Object, patch []mutator.PatchOperation) (*semver.Version, error) {
	var version string
	var ok bool
	// check first if the release version is contained in a patch
	for _, p := range patch {
		if p.Path == fmt.Sprintf("/metadata/labels/%s", EscapeJSONPatchString(label.Release)) {
			version = p.Value.(string)
			return semver.New(version)
		}
	}
	// otherwise check the labels
	version, ok = meta.GetLabels()[label.Release]
	if !ok {
		return nil, microerror.Maskf(parsingFailedError, "unable to get release version from Object %s", meta.GetName())
	}
	return semver.New(version)
}

// Ensure the needed escapes are in place. See https://tools.ietf.org/html/rfc6901#section-3 .
func EscapeJSONPatchString(input string) string {
	input = strings.ReplaceAll(input, "~", "~0")
	input = strings.ReplaceAll(input, "/", "~1")

	return input
}
