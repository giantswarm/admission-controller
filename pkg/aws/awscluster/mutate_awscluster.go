// Package awsmachinedeployment intercepts write activity to AWSMachineDeployment objects.
package awscluster

import (
	"context"
	"fmt"
	"strings"

	"github.com/blang/semver/v4"
	"github.com/giantswarm/apiextensions/v3/pkg/annotation"
	infrastructurev1alpha2 "github.com/giantswarm/apiextensions/v3/pkg/apis/infrastructure/v1alpha2"
	"github.com/giantswarm/k8sclient/v5/pkg/k8sclient"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	admissionv1 "k8s.io/api/admission/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/giantswarm/aws-admission-controller/v2/config"
	"github.com/giantswarm/aws-admission-controller/v2/pkg/aws"
	"github.com/giantswarm/aws-admission-controller/v2/pkg/key"
	"github.com/giantswarm/aws-admission-controller/v2/pkg/label"
	"github.com/giantswarm/aws-admission-controller/v2/pkg/mutator"
)

const (
	stringTrue  = "true"
	stringFalse = "false"
)

type Config struct {
	K8sClient k8sclient.Interface
	Logger    micrologger.Logger
}

// Mutator for AWSMachineDeployment object.
type Mutator struct {
	k8sClient k8sclient.Interface
	logger    micrologger.Logger

	podCIDRBlock           string
	dnsDomain              string
	region                 string
	validAvailabilityZones []string
}

func NewMutator(config config.Config) (*Mutator, error) {
	if config.K8sClient == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.K8sClient must not be empty", config)
	}
	if config.Logger == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Logger must not be empty", config)
	}

	var availabilityZones []string = strings.Split(config.AvailabilityZones, ",")
	mutator := &Mutator{
		k8sClient: config.K8sClient,
		logger:    config.Logger,

		podCIDRBlock:           fmt.Sprintf("%s/%s", config.PodSubnet, config.PodCIDR),
		dnsDomain:              strings.TrimPrefix(config.Endpoint, "k8s."),
		region:                 config.Region,
		validAvailabilityZones: availabilityZones,
	}

	return mutator, nil
}

// Mutate is the function executed for every matching webhook request.
func (m *Mutator) Mutate(request *admissionv1.AdmissionRequest) ([]mutator.PatchOperation, error) {
	var result []mutator.PatchOperation

	if request.DryRun != nil && *request.DryRun {
		return result, nil
	}
	if request.Operation == admissionv1.Create {
		return m.MutateCreate(request)
	}
	if request.Operation == admissionv1.Update {
		return m.MutateUpdate(request)
	}
	return result, nil
}

// MutateCreate is the function executed for every create webhook request.
func (m *Mutator) MutateCreate(request *admissionv1.AdmissionRequest) ([]mutator.PatchOperation, error) {
	var result []mutator.PatchOperation
	var patch []mutator.PatchOperation
	var err error

	awsCluster := &infrastructurev1alpha2.AWSCluster{}
	if _, _, err = mutator.Deserializer.Decode(request.Object.Raw, nil, awsCluster); err != nil {
		return nil, microerror.Maskf(parsingFailedError, "unable to parse AWSCluster: %v", err)
	}

	patch, err = m.MutatePodCIDR(*awsCluster)
	if err != nil {
		return nil, microerror.Mask(err)
	}
	result = append(result, patch...)

	patch, err = m.MutateCredential(*awsCluster)
	if err != nil {
		return nil, microerror.Mask(err)
	}
	result = append(result, patch...)

	patch, err = m.MutateDescription(*awsCluster)
	if err != nil {
		return nil, microerror.Mask(err)
	}
	result = append(result, patch...)

	patch, err = m.MutateDomain(*awsCluster)
	if err != nil {
		return nil, microerror.Mask(err)
	}
	result = append(result, patch...)

	patch, err = m.MutateRegion(*awsCluster)
	if err != nil {
		return nil, microerror.Mask(err)
	}
	result = append(result, patch...)

	patch, err = m.MutateReleaseVersion(*awsCluster)
	if err != nil {
		return nil, microerror.Mask(err)
	}
	releaseVersion, err := aws.ReleaseVersion(awsCluster, patch)
	if err != nil {
		return nil, microerror.Maskf(parsingFailedError, "unable to parse release version from AWSCluster")
	}
	result = append(result, patch...)

	patch, err = m.MutateOperatorVersion(*awsCluster, releaseVersion)
	if err != nil {
		return nil, microerror.Mask(err)
	}
	result = append(result, patch...)

	if !aws.IsHAVersion(releaseVersion) {
		patch, err = m.MutateMasterPreHA(*awsCluster)
		if err != nil {
			return nil, microerror.Mask(err)
		}
		result = append(result, patch...)
	}

	return result, nil
}

// MutateUpdate is the function executed for every update webhook request.
func (m *Mutator) MutateUpdate(request *admissionv1.AdmissionRequest) ([]mutator.PatchOperation, error) {
	var result []mutator.PatchOperation
	var patch []mutator.PatchOperation
	var err error

	awsCluster := &infrastructurev1alpha2.AWSCluster{}
	if _, _, err = mutator.Deserializer.Decode(request.Object.Raw, nil, awsCluster); err != nil {
		return nil, microerror.Maskf(parsingFailedError, "unable to parse AWSCluster: %v", err)
	}
	awsClusterOld := &infrastructurev1alpha2.AWSCluster{}
	if _, _, err = mutator.Deserializer.Decode(request.OldObject.Raw, nil, awsClusterOld); err != nil {
		return nil, microerror.Maskf(parsingFailedError, "unable to parse old AWSCluster: %v", err)
	}
	releaseVersion, err := aws.ReleaseVersion(awsCluster, patch)
	if err != nil {
		return nil, microerror.Maskf(parsingFailedError, "unable to parse release version from AWSCluster")
	}

	patch, err = m.MutatePodCIDR(*awsCluster)
	if err != nil {
		return nil, microerror.Mask(err)
	}
	result = append(result, patch...)

	patch, err = m.MutateCredential(*awsCluster)
	if err != nil {
		return nil, microerror.Mask(err)
	}
	result = append(result, patch...)

	patch, err = m.MutateDescription(*awsCluster)
	if err != nil {
		return nil, microerror.Mask(err)
	}
	result = append(result, patch...)

	patch, err = m.MutateDomain(*awsCluster)
	if err != nil {
		return nil, microerror.Mask(err)
	}
	result = append(result, patch...)

	patch, err = m.MutateRegion(*awsCluster)
	if err != nil {
		return nil, microerror.Mask(err)
	}
	result = append(result, patch...)

	patch, err = m.MutateAnnotationNodeTerminateUnhealthy(*awsCluster)
	if err != nil {
		return nil, microerror.Mask(err)
	}
	result = append(result, patch...)

	if !aws.IsHAVersion(releaseVersion) {
		patch, err = m.MutateMasterPreHA(*awsCluster)
		if err != nil {
			return nil, microerror.Mask(err)
		}
		result = append(result, patch...)
	}

	return result, nil
}

// MutatePodCIDR defaults the Pod CIDR if it is not set.
func (m *Mutator) MutatePodCIDR(awsCluster infrastructurev1alpha2.AWSCluster) ([]mutator.PatchOperation, error) {
	var result []mutator.PatchOperation
	//nolint:staticcheck // SA4022 the address of a variable cannot be nil
	if &awsCluster.Spec.Provider.Pods != nil {
		if awsCluster.Spec.Provider.Pods.CIDRBlock != "" {
			return result, nil
		}
		if awsCluster.Spec.Provider.Pods.ExternalSNAT != nil {
			// If the Pod CIDR is not set but the pods attribute exists, we default here
			m.Log("level", "debug", "message", fmt.Sprintf("AWSCluster %s Pod CIDR Block is not set and will be defaulted to %s",
				awsCluster.ObjectMeta.Name,
				m.podCIDRBlock),
			)
			patch := mutator.PatchAdd("/spec/provider/pods/", "cidrBlock")
			result = append(result, patch)
			patch = mutator.PatchAdd("/spec/provider/pods/cidrBlock", m.podCIDRBlock)
			result = append(result, patch)
			return result, nil
		}
	}
	// If the Pod CIDR is not set we default it here
	m.Log("level", "debug", "message", fmt.Sprintf("AWSCluster %s Pod CIDR Block is not set and will be defaulted to %s",
		awsCluster.ObjectMeta.Name,
		m.podCIDRBlock),
	)
	patch := mutator.PatchAdd("/spec/provider/", "pods")
	result = append(result, patch)
	patch = mutator.PatchAdd("/spec/provider/pods", map[string]string{"cidrBlock": m.podCIDRBlock})
	result = append(result, patch)

	return result, nil
}

// MutateMasterPreHA is there to mutate the master instance attributes of the AWSCluster CR in legacy versions.
// This can be deprecated once no versions < 11.4.0 are in use anymore
func (m *Mutator) MutateMasterPreHA(awsCluster infrastructurev1alpha2.AWSCluster) ([]mutator.PatchOperation, error) {
	var result []mutator.PatchOperation

	var availabilityZone string
	var instanceType string
	{
		//nolint:staticcheck // SA4022 the address of a variable cannot be nil
		if &awsCluster.Spec.Provider.Master != nil {
			if awsCluster.Spec.Provider.Master.AvailabilityZone != "" && awsCluster.Spec.Provider.Master.InstanceType != "" {
				return result, nil
			}
			availabilityZone = awsCluster.Spec.Provider.Master.AvailabilityZone
			instanceType = awsCluster.Spec.Provider.Master.InstanceType
		} else {
			patch := mutator.PatchAdd("/spec/provider/", "master")
			result = append(result, patch)
		}
	}
	if instanceType == "" {
		instanceType = aws.DefaultMasterInstanceType
	}
	if availabilityZone == "" {
		defaultedAZs := aws.GetNavailabilityZones(&aws.Handler{K8sClient: m.k8sClient, Logger: m.logger}, 1, m.validAvailabilityZones)
		availabilityZone = defaultedAZs[0]
	}
	// If the Master attributes are not set, we default them here
	m.Log("level", "debug", "message", fmt.Sprintf("Pre-HA AWSCluster %s Master attributes will be defaulted to availabilityZone %s and instanceType %s",
		awsCluster.ObjectMeta.Name,
		availabilityZone,
		instanceType),
	)
	patch := mutator.PatchAdd("/spec/provider/master", map[string]string{"availabilityZone": availabilityZone, "instanceType": instanceType})
	result = append(result, patch)
	return result, nil
}

//  MutateCredential defaults the cluster credential if it is not set.
func (m *Mutator) MutateCredential(awsCluster infrastructurev1alpha2.AWSCluster) ([]mutator.PatchOperation, error) {
	var result []mutator.PatchOperation
	if awsCluster.Spec.Provider.CredentialSecret.Name != "" && awsCluster.Spec.Provider.CredentialSecret.Namespace != "" {
		return result, nil
	}
	// If the cluster credential secret attribute is not set or incomplete, we default here

	var secretName types.NamespacedName
	{
		secret, err := m.fetchCredentialSecret(key.Organization(&awsCluster))
		if IsNotFound(err) {
			// if the credential secret can not be found we do no fail but use the default one
			m.Log("level", "debug", "message", fmt.Sprintf("Could not fetch credential-secret. Using default secret instead: %v", err))
			secretName = aws.DefaultCredentialSecret()
		} else if err != nil {
			return nil, microerror.Mask(err)
		} else {
			secretName = types.NamespacedName{
				Name:      secret.GetName(),
				Namespace: secret.GetNamespace(),
			}
		}
	}
	m.Log("level", "debug", "message", fmt.Sprintf("AWSCluster %s credential secret is not set and will be defaulted to %s/%s",
		awsCluster.ObjectMeta.Name,
		secretName.Namespace,
		secretName.Name),
	)
	patch := mutator.PatchAdd("/spec/provider/credentialSecret", map[string]string{"name": secretName.Name, "namespace": secretName.Namespace})
	result = append(result, patch)
	return result, nil
}
func (m *Mutator) fetchCredentialSecret(organization string) (corev1.Secret, error) {
	var err error
	secrets := corev1.SecretList{}

	// return early if no org is given
	if organization == "" {
		return corev1.Secret{}, microerror.Maskf(notFoundError, "Could not find secret because organization is unknown.")
	}

	// Fetch the credential secret
	m.Log("level", "debug", "message", fmt.Sprintf("Fetching credential secret for organization %s", organization))
	err = m.k8sClient.CtrlClient().List(
		context.Background(),
		&secrets,
		client.MatchingLabels{label.Organization: organization, label.ManagedBy: "credentiald"},
	)
	if err != nil {
		return corev1.Secret{}, microerror.Maskf(notFoundError, "Failed to fetch credential-secret: %v", err)
	}
	if len(secrets.Items) == 0 {
		return corev1.Secret{}, microerror.Maskf(notFoundError, "Could not find credential-secret for organization %s", organization)
	}
	if len(secrets.Items) > 1 {
		return corev1.Secret{}, microerror.Maskf(notFoundError, "Found %v credential secrets instead of one for organization %s", len(secrets.Items), organization)
	}
	return secrets.Items[0], nil
}

//  MutateDescription defaults the cluster description if it is not set.
func (m *Mutator) MutateDescription(awsCluster infrastructurev1alpha2.AWSCluster) ([]mutator.PatchOperation, error) {
	var result []mutator.PatchOperation
	if awsCluster.Spec.Cluster.Description == "" {
		// If the cluster description is not set, we default here
		m.Log("level", "debug", "message", fmt.Sprintf("AWSCluster %s Description is not set and will be defaulted to %s",
			awsCluster.ObjectMeta.Name,
			aws.DefaultClusterDescription),
		)
		patch := mutator.PatchAdd("/spec/cluster/description", aws.DefaultClusterDescription)
		result = append(result, patch)
	}
	return result, nil
}

//  MutateDomain defaults the cluster dns domain if it is not set.
func (m *Mutator) MutateDomain(awsCluster infrastructurev1alpha2.AWSCluster) ([]mutator.PatchOperation, error) {
	var result []mutator.PatchOperation
	if awsCluster.Spec.Cluster.DNS.Domain == "" {
		// If the dns domain is not set, we default here
		m.Log("level", "debug", "message", fmt.Sprintf("AWSCluster %s DNS domain is not set and will be defaulted to %s",
			awsCluster.ObjectMeta.Name,
			m.dnsDomain),
		)
		patch := mutator.PatchAdd("/spec/cluster/dns/domain", m.dnsDomain)
		result = append(result, patch)
	}
	return result, nil
}

func (m *Mutator) MutateOperatorVersion(awsCluster infrastructurev1alpha2.AWSCluster, releaseVersion *semver.Version) ([]mutator.PatchOperation, error) {
	var result []mutator.PatchOperation
	var patch []mutator.PatchOperation
	var err error

	if key.AWSOperator(&awsCluster) != "" {
		return result, nil
	}
	// Retrieve the `Release` CR.
	release, err := aws.FetchRelease(&aws.Handler{K8sClient: m.k8sClient, Logger: m.logger}, releaseVersion)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	// mutate the operator label
	patch, err = aws.MutateLabelFromRelease(&aws.Handler{K8sClient: m.k8sClient, Logger: m.logger}, &awsCluster, *release, label.AWSOperatorVersion, "aws-operator")
	if err != nil {
		return nil, microerror.Mask(err)
	}
	result = append(result, patch...)

	return result, nil
}

func (m *Mutator) MutateReleaseVersion(awsCluster infrastructurev1alpha2.AWSCluster) ([]mutator.PatchOperation, error) {
	var result []mutator.PatchOperation
	var patch []mutator.PatchOperation
	var err error

	if key.Release(&awsCluster) != "" {
		return result, nil
	}
	// Retrieve the `Cluster` CR related to this object.
	cluster, err := aws.FetchCluster(&aws.Handler{K8sClient: m.k8sClient, Logger: m.logger}, &awsCluster)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	// mutate the release label
	patch, err = aws.MutateLabelFromCluster(&aws.Handler{K8sClient: m.k8sClient, Logger: m.logger}, &awsCluster, *cluster, label.Release)
	if err != nil {
		return nil, microerror.Mask(err)
	}
	result = append(result, patch...)

	return result, nil
}

//  MutateRegion defaults the cluster region if it is not set.
func (m *Mutator) MutateRegion(awsCluster infrastructurev1alpha2.AWSCluster) ([]mutator.PatchOperation, error) {
	var result []mutator.PatchOperation
	if awsCluster.Spec.Provider.Region == "" {
		// If the region is not set, we default here
		m.Log("level", "debug", "message", fmt.Sprintf("AWSCluster %s region is not set and will be defaulted to %s",
			awsCluster.ObjectMeta.Name,
			m.region),
		)
		patch := mutator.PatchAdd("/spec/provider/region", m.region)
		result = append(result, patch)
	}
	return result, nil
}

//MutateAnnotationNodeTerminateUnhealthy migrate NodeTerminateUnhealthy annotations from alpha to stable in case it is configured.
// TODO https://github.com/giantswarm/giantswarm/issues/17395
// this migration code can be removed once all AWS clusters are on release 15.0.0 or newer
func (m *Mutator) MutateAnnotationNodeTerminateUnhealthy(awsCluster infrastructurev1alpha2.AWSCluster) ([]mutator.PatchOperation, error) {
	var result []mutator.PatchOperation

	release, err := semver.New(key.Release(&awsCluster))
	if err != nil {
		return nil, microerror.Maskf(notAllowedError, fmt.Sprintf("AWSCluster release has invalid value '%s'.", key.Release(&awsCluster)))
	}

	// new annotation is available from release >= 15.x.x
	release15 := semver.MustParse("14.99.99")
	if release.GE(release15) {
		//load the old alpha annotation
		if terminateUnhealthy, ok := awsCluster.GetAnnotations()[aws.AnnotationAlphaNodeTerminateUnhealthy]; ok {
			// clean the old annotation
			delete(awsCluster.Annotations, aws.AnnotationAlphaNodeTerminateUnhealthy)

			// set new annotation, any value except 'false' is considered as true
			if terminateUnhealthy == stringFalse {
				awsCluster.Annotations[annotation.NodeTerminateUnhealthy] = stringFalse
			} else {
				awsCluster.Annotations[annotation.NodeTerminateUnhealthy] = stringTrue
			}

			m.Log("level", "debug", "message", fmt.Sprintf("AWSCluster annotation '%s' migrated to '%s'.",
				aws.AnnotationAlphaNodeTerminateUnhealthy,
				annotation.NodeTerminateUnhealthy),
			)
			patch := mutator.PatchAdd("/metadata/annotations", awsCluster.Annotations)
			result = append(result, patch)
		}
	}
	return result, nil
}

func (m *Mutator) Log(keyVals ...interface{}) {
	m.logger.Log(keyVals...)
}

func (m *Mutator) Resource() string {
	return "awscluster"
}
