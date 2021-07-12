package aws

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/blang/semver/v4"
	infrastructurev1alpha2 "github.com/giantswarm/apiextensions/v3/pkg/apis/infrastructure/v1alpha2"
	releasev1alpha1 "github.com/giantswarm/apiextensions/v3/pkg/apis/release/v1alpha1"
	"github.com/giantswarm/backoff"
	"github.com/giantswarm/microerror"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	capiv1alpha2 "sigs.k8s.io/cluster-api/api/v1alpha2"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/giantswarm/aws-admission-controller/v2/pkg/key"
	"github.com/giantswarm/aws-admission-controller/v2/pkg/label"
)

func FetchAWSCluster(m *Handler, meta metav1.Object) (*infrastructurev1alpha2.AWSCluster, error) {
	var awsCluster infrastructurev1alpha2.AWSCluster
	var err error
	var fetch func() error

	namespace := meta.GetNamespace()
	if namespace == "" {
		namespace = metav1.NamespaceDefault
	}

	// Retrieve the Cluster ID.
	clusterID := key.Cluster(meta)
	if clusterID == "" {
		return nil, microerror.Maskf(invalidConfigError, "Object has no %s label, can't fetch AWSCluster.", label.Cluster)
	}

	// Fetch the AWSCluster CR
	{
		m.Logger.Log("level", "debug", "message", fmt.Sprintf("Fetching AWSCluster %s", clusterID))
		fetch = func() error {
			err := m.K8sClient.CtrlClient().Get(context.Background(), client.ObjectKey{Name: clusterID, Namespace: namespace}, &awsCluster)
			if IsNotFound(err) {
				return microerror.Maskf(notFoundError, "Looking for AWSCluster named %s but it was not found.", clusterID)
			} else if err != nil {
				return microerror.Mask(err)
			}
			return nil
		}
	}

	{
		b := backoff.NewMaxRetries(3, 10*time.Millisecond)
		err = backoff.Retry(fetch, b)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}
	return &awsCluster, nil
}

func FetchAWSControlPlane(m *Handler, meta metav1.Object) (*infrastructurev1alpha2.AWSControlPlane, error) {
	var awsControlPlane infrastructurev1alpha2.AWSControlPlane
	var err error
	var fetch func() error

	// Retrieve the Cluster ID.
	clusterID := key.Cluster(meta)
	if clusterID == "" {
		return nil, microerror.Maskf(invalidConfigError, "Object has no %s label, can't fetch AWSControlPlane.", label.Cluster)
	}

	// Fetch the AWSControlPlane.
	{
		m.Logger.Log("level", "debug", "message", fmt.Sprintf("Fetching AWSControlPlane for Cluster %s", clusterID))
		fetch = func() error {
			awsControlPlanes := infrastructurev1alpha2.AWSControlPlaneList{}
			err = m.K8sClient.CtrlClient().List(
				context.Background(),
				&awsControlPlanes,
				client.MatchingLabels{label.Cluster: clusterID},
			)
			if err != nil {
				return microerror.Maskf(notFoundError, "failed to fetch AWSControlplane for Cluster %s: %v", clusterID, err)
			}
			if len(awsControlPlanes.Items) == 0 {
				return microerror.Maskf(notFoundError, "Could not find AWSControlplane for Cluster %s", clusterID)
			}
			if len(awsControlPlanes.Items) > 1 {
				return microerror.Maskf(invalidConfigError, "Found %v AWSControlplanes instead of one for cluster %s", len(awsControlPlanes.Items), clusterID)
			}
			awsControlPlane = awsControlPlanes.Items[0]
			return nil
		}
	}

	{
		b := backoff.NewMaxRetries(3, 10*time.Millisecond)
		err = backoff.Retry(fetch, b)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}
	return &awsControlPlane, nil
}

func FetchCluster(m *Handler, meta metav1.Object) (*capiv1alpha2.Cluster, error) {
	var cluster capiv1alpha2.Cluster
	var err error
	var fetch func() error

	namespace := meta.GetNamespace()
	if namespace == "" {
		namespace = metav1.NamespaceDefault
	}
	// Retrieve the Cluster ID.
	clusterID := key.Cluster(meta)
	if clusterID == "" {
		return nil, microerror.Maskf(invalidConfigError, "Object has no %s label, can't fetch cluster.", label.Cluster)
	}

	// Fetch the Cluster CR
	{
		m.Logger.Log("level", "debug", "message", fmt.Sprintf("Fetching Cluster %s", clusterID))
		fetch = func() error {
			err := m.K8sClient.CtrlClient().Get(context.Background(), client.ObjectKey{Name: clusterID, Namespace: namespace}, &cluster)
			if IsNotFound(err) {
				return microerror.Maskf(notFoundError, "Looking for Cluster named %s but it was not found.", clusterID)
			} else if err != nil {
				return microerror.Mask(err)
			}
			return nil
		}
	}

	{
		b := backoff.NewMaxRetries(3, 10*time.Millisecond)
		err = backoff.Retry(fetch, b)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}
	return &cluster, nil
}

func FetchG8sControlPlane(m *Handler, meta metav1.Object) (*infrastructurev1alpha2.G8sControlPlane, error) {
	var g8sControlPlane infrastructurev1alpha2.G8sControlPlane
	var err error
	var fetch func() error

	// Retrieve the Cluster ID.
	clusterID := key.Cluster(meta)
	if clusterID == "" {
		return nil, microerror.Maskf(invalidConfigError, "Object has no %s label, can't fetch G8sControlPlane.", label.Cluster)
	}

	// Fetch the G8sControlPlane.
	{
		m.Logger.Log("level", "debug", "message", fmt.Sprintf("Fetching G8sControlPlane for Cluster %s", clusterID))
		fetch = func() error {
			awsControlPlanes := infrastructurev1alpha2.G8sControlPlaneList{}
			err = m.K8sClient.CtrlClient().List(
				context.Background(),
				&awsControlPlanes,
				client.MatchingLabels{label.Cluster: clusterID},
			)
			if err != nil {
				return microerror.Maskf(notFoundError, "failed to fetch G8sControlplane for Cluster %s: %v", clusterID, err)
			}
			if len(awsControlPlanes.Items) == 0 {
				return microerror.Maskf(notFoundError, "Could not find G8sControlplane for Cluster %s", clusterID)
			}
			if len(awsControlPlanes.Items) > 1 {
				return microerror.Maskf(invalidConfigError, "Found %v G8sControlplanes instead of one for cluster %s", len(awsControlPlanes.Items), clusterID)
			}
			m.Logger.Log("level", "debug", "message", fmt.Sprintf("Found G8sControlPlane for Cluster %s", clusterID))
			g8sControlPlane = awsControlPlanes.Items[0]
			return nil
		}
	}

	{
		b := backoff.NewMaxRetries(3, 10*time.Millisecond)
		err = backoff.Retry(fetch, b)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}
	return &g8sControlPlane, nil
}

func FetchNewestReleaseVersion(m *Handler) (*semver.Version, error) {
	var activeReleases []semver.Version
	var err error

	// Fetch the Release CRs
	releases := releasev1alpha1.ReleaseList{}
	{

		err = m.K8sClient.CtrlClient().List(
			context.Background(),
			&releases,
		)
		if err != nil {
			return nil, microerror.Maskf(notFoundError, "failed to fetch releases: %v", err)
		}
		if len(releases.Items) == 0 {
			return nil, microerror.Maskf(notFoundError, "Could not find any releases.")
		}
	}
	// Find the active, production-ready releases
	{
		for _, r := range releases.Items {
			if r.Spec.State != releasev1alpha1.StateActive {
				continue
			}

			var version *semver.Version
			version, err = semver.New(strings.TrimPrefix(r.GetName(), "v"))
			if err != nil {
				continue
			}
			if !IsVersionProductionReady(version) {
				continue
			}
			capi, err := IsCAPIVersion(version)
			if err != nil {
				return nil, microerror.Mask(err)
			}
			if capi {
				continue
			}

			activeReleases = append(activeReleases, *version)
		}
	}
	if len(activeReleases) == 0 {
		return nil, microerror.Maskf(notFoundError, "Could not find any active releases.")
	}
	// Sort releases by version (descending).
	sort.Slice(activeReleases, func(i, j int) bool {
		return activeReleases[i].GT(activeReleases[j])
	})

	return &activeReleases[0], nil
}

func FetchRelease(m *Handler, version *semver.Version) (*releasev1alpha1.Release, error) {
	var releaseName string
	var release releasev1alpha1.Release
	var err error

	// Get release name
	{
		releaseName = version.String()
		if !strings.HasPrefix(releaseName, "v") {
			releaseName = fmt.Sprintf("v%s", releaseName)
		}
	}
	// Fetch the Release CR
	{
		m.Logger.Log("level", "debug", "message", fmt.Sprintf("Fetching Release %s", releaseName))
		err = m.K8sClient.CtrlClient().Get(context.Background(), client.ObjectKey{Name: releaseName, Namespace: metav1.NamespaceDefault}, &release)
		if IsNotFound(err) {
			return nil, microerror.Maskf(notFoundError, "Looking for Release %s but it was not found.", releaseName)
		} else if err != nil {
			return nil, microerror.Mask(err)
		}
	}
	return &release, nil
}
