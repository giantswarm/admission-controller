package cluster

import (
	"context"
	"strconv"
	"testing"
	"time"

	infrastructurev1alpha2 "github.com/giantswarm/apiextensions/v3/pkg/apis/infrastructure/v1alpha2"
	releasev1alpha1 "github.com/giantswarm/apiextensions/v3/pkg/apis/release/v1alpha1"
	"github.com/giantswarm/apiextensions/v3/pkg/label"
	"github.com/giantswarm/micrologger/microloggertest"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/giantswarm/aws-admission-controller/v2/pkg/unittest"
)

func TestValidateReleaseVersion(t *testing.T) {
	testCases := []struct {
		ctx  context.Context
		name string

		oldReleaseVersion string
		newReleaseVersion string
		valid             bool
	}{
		{
			// Version unchanged
			name: "case 0",
			ctx:  context.Background(),

			oldReleaseVersion: "3.0.0",
			newReleaseVersion: "3.0.0",
			valid:             true,
		},
		{
			// version changed to valid release
			name: "case 1",
			ctx:  context.Background(),

			oldReleaseVersion: "3.0.0",
			newReleaseVersion: "4.0.0",
			valid:             true,
		},
		{
			// version changed to deprecated release
			name: "case 2",
			ctx:  context.Background(),

			oldReleaseVersion: "3.0.0",
			newReleaseVersion: "3.2.0",
			valid:             false,
		},
		{
			// version changed to invalid release
			name: "case 3",
			ctx:  context.Background(),

			oldReleaseVersion: "3.0.0",
			newReleaseVersion: "3.3.0",
			valid:             false,
		},
		{
			// version changed with major skip
			name: "case 4",
			ctx:  context.Background(),

			oldReleaseVersion: "3.0.0",
			newReleaseVersion: "5.0.0",
			valid:             false,
		},
		{
			// version changed to older release
			name: "case 5",
			ctx:  context.Background(),

			oldReleaseVersion: "3.0.0",
			newReleaseVersion: "2.0.0",
			valid:             false,
		},
		{
			// version changed with multiple major skips
			name: "case 6",
			ctx:  context.Background(),

			oldReleaseVersion: "3.0.0",
			newReleaseVersion: "7.0.0",
			valid:             false,
		},
		{
			// version changed to older minor release
			name: "case 7",
			ctx:  context.Background(),

			oldReleaseVersion: "3.0.0",
			newReleaseVersion: "2.9.0",
			valid:             false,
		},
		{
			// version changed to older minor release
			name: "case 8",
			ctx:  context.Background(),

			oldReleaseVersion: "3.2.0",
			newReleaseVersion: "3.1.0",
			valid:             true,
		},
		{
			// version changed to older minor release
			name: "case 9",
			ctx:  context.Background(),

			oldReleaseVersion: "3.4.1",
			newReleaseVersion: "3.1.0",
			valid:             true,
		},
		{
			// version changed to older patch release
			name: "case 10",
			ctx:  context.Background(),

			oldReleaseVersion: "3.2.2",
			newReleaseVersion: "3.2.1",
			valid:             true,
		},
	}
	for i, tc := range testCases {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			var err error

			fakeK8sClient := unittest.FakeK8sClient()
			handle := &Validator{
				k8sClient: fakeK8sClient,
				logger:    microloggertest.New(),
			}

			// create releases for testing
			releases := []unittest.ReleaseData{
				{
					Name:  "v5.0.0",
					State: releasev1alpha1.StateActive,
				},
				{
					Name:  "v4.0.0",
					State: releasev1alpha1.StateActive,
				},
				{
					Name:  "v3.4.1",
					State: releasev1alpha1.StateActive,
				},
				{
					Name:  "v3.2.2",
					State: releasev1alpha1.StateActive,
				},
				{
					Name:  "v3.2.1",
					State: releasev1alpha1.StateActive,
				},
				{
					Name:  "v3.2.0",
					State: releasev1alpha1.StateDeprecated,
				},
				{
					Name:  "v3.1.0",
					State: releasev1alpha1.StateActive,
				},
				{
					Name:  "v2.0.0",
					State: releasev1alpha1.StateActive,
				},
			}
			for _, r := range releases {
				release := unittest.DefaultRelease()
				release.SetName(r.Name)
				release.Spec.State = r.State
				err = fakeK8sClient.CtrlClient().Create(tc.ctx, &release)
				if err != nil {
					t.Fatal(err)
				}
			}

			// create old and new object with release version labels
			oldObject := unittest.DefaultCluster()
			oldLabels := unittest.DefaultLabels()
			oldLabels[label.ReleaseVersion] = tc.oldReleaseVersion
			oldObject.SetLabels(oldLabels)

			newObject := unittest.DefaultCluster()
			newLabels := unittest.DefaultLabels()
			newLabels[label.ReleaseVersion] = tc.newReleaseVersion
			newObject.SetLabels(newLabels)

			// check if the result is as expected
			err = handle.ReleaseVersionValid(oldObject, newObject)
			if tc.valid && err != nil {
				t.Fatalf("unexpected error %v", err)
			}
			if !tc.valid && err == nil {
				t.Fatalf("expected error but returned %v", err)
			}
		})
	}
}

func TestValidClusterStatus(t *testing.T) {
	testCases := []struct {
		ctx  context.Context
		name string

		oldReleaseVersion string
		newReleaseVersion string
		conditions        []infrastructurev1alpha2.CommonClusterStatusCondition

		valid bool
	}{
		{
			// no upgrade
			name: "case 0",
			ctx:  context.Background(),

			conditions: []infrastructurev1alpha2.CommonClusterStatusCondition{
				{LastTransitionTime: metav1.NewTime(time.Now()),
					Condition: infrastructurev1alpha2.ClusterStatusConditionCreating},
			},
			oldReleaseVersion: "3.0.0",
			newReleaseVersion: "3.0.0",
			valid:             true,
		},
		{
			// Cluster is creating
			name: "case 1",
			ctx:  context.Background(),

			conditions: []infrastructurev1alpha2.CommonClusterStatusCondition{
				{LastTransitionTime: metav1.NewTime(time.Now()),
					Condition: infrastructurev1alpha2.ClusterStatusConditionCreating},
			},
			oldReleaseVersion: "3.0.0",
			newReleaseVersion: "4.0.0",
			valid:             false,
		},
		{
			// Cluster is created
			name: "case 2",
			ctx:  context.Background(),

			conditions: []infrastructurev1alpha2.CommonClusterStatusCondition{
				{LastTransitionTime: metav1.NewTime(time.Now()),
					Condition: infrastructurev1alpha2.ClusterStatusConditionCreated},
				{LastTransitionTime: metav1.NewTime(time.Now().Add(-15 * time.Minute)),
					Condition: infrastructurev1alpha2.ClusterStatusConditionCreating},
			},
			oldReleaseVersion: "3.0.0",
			newReleaseVersion: "4.0.0",
			valid:             true,
		},
		{
			// Cluster is updating
			name: "case 3",
			ctx:  context.Background(),

			conditions: []infrastructurev1alpha2.CommonClusterStatusCondition{
				{LastTransitionTime: metav1.NewTime(time.Now()),
					Condition: infrastructurev1alpha2.ClusterStatusConditionUpdating},
				{LastTransitionTime: metav1.NewTime(time.Now().Add(-15 * time.Minute)),
					Condition: infrastructurev1alpha2.ClusterStatusConditionCreated},
				{LastTransitionTime: metav1.NewTime(time.Now().Add(-30 * time.Minute)),
					Condition: infrastructurev1alpha2.ClusterStatusConditionCreating},
			},
			oldReleaseVersion: "3.0.0",
			newReleaseVersion: "4.0.0",
			valid:             false,
		},
		{
			// Cluster is updated
			name: "case 4",
			ctx:  context.Background(),

			conditions: []infrastructurev1alpha2.CommonClusterStatusCondition{
				{LastTransitionTime: metav1.NewTime(time.Now()),
					Condition: infrastructurev1alpha2.ClusterStatusConditionUpdated},
				{LastTransitionTime: metav1.NewTime(time.Now().Add(-15 * time.Minute)),
					Condition: infrastructurev1alpha2.ClusterStatusConditionUpdating},
				{LastTransitionTime: metav1.NewTime(time.Now().Add(-30 * time.Minute)),
					Condition: infrastructurev1alpha2.ClusterStatusConditionCreated},
				{LastTransitionTime: metav1.NewTime(time.Now().Add(-60 * time.Minute)),
					Condition: infrastructurev1alpha2.ClusterStatusConditionCreating},
			},
			oldReleaseVersion: "3.0.0",
			newReleaseVersion: "4.0.0",
			valid:             true,
		},
	}
	for i, tc := range testCases {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			var err error

			fakeK8sClient := unittest.FakeK8sClient()
			handle := &Validator{
				k8sClient: fakeK8sClient,
				logger:    microloggertest.New(),
			}

			awsCluster := unittest.DefaultAWSCluster()
			awsCluster.Status.Cluster.Conditions = tc.conditions
			err = fakeK8sClient.CtrlClient().Create(tc.ctx, &awsCluster)
			if err != nil {
				t.Fatal(err)
			}

			// create old and new object with release version labels
			oldObject := unittest.DefaultCluster()
			oldLabels := unittest.DefaultLabels()
			oldLabels[label.ReleaseVersion] = tc.oldReleaseVersion
			oldObject.SetLabels(oldLabels)

			newObject := unittest.DefaultCluster()
			newLabels := unittest.DefaultLabels()
			newLabels[label.ReleaseVersion] = tc.newReleaseVersion
			newObject.SetLabels(newLabels)

			// check if the result is as expected
			err = handle.ClusterStatusValid(oldObject, newObject)
			if tc.valid && err != nil {
				t.Fatalf("unexpected error %v", err)
			}
			if !tc.valid && err == nil {
				t.Fatalf("expected error but returned %v", err)
			}
		})
	}
}
