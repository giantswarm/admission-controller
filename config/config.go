package config

import (
	infrastructurev1alpha2 "github.com/giantswarm/apiextensions/v3/pkg/apis/infrastructure/v1alpha2"
	releasev1alpha1 "github.com/giantswarm/apiextensions/v3/pkg/apis/release/v1alpha1"
	securityv1alpha1 "github.com/giantswarm/apiextensions/v3/pkg/apis/security/v1alpha1"
	"github.com/giantswarm/k8sclient/v5/pkg/k8sclient"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	kingpin "gopkg.in/alecthomas/kingpin.v2"
	restclient "k8s.io/client-go/rest"
	apiv1alpha2 "sigs.k8s.io/cluster-api/api/v1alpha2"
)

const (
	defaultAddress        = ":8443"
	defaultMetricsAddress = ":8080"
)

type Config struct {
	Address                  string
	AdminGroup               string
	AllTargetGroup           string
	MetricsAddress           string
	AvailabilityZones        string
	CertFile                 string
	DockerCIDR               string
	Endpoint                 string
	IPAMNetworkCIDR          string
	KubernetesClusterIPRange string
	MasterInstanceTypes      string
	PodCIDR                  string
	PodSubnet                string
	Region                   string
	WorkerInstanceTypes      string
	Logger                   micrologger.Logger
	K8sClient                k8sclient.Interface
	KeyFile                  string
}

func Parse() (Config, error) {
	var err error
	var config Config

	// Create a new logger that is used by all admitters.
	var newLogger micrologger.Logger
	{
		newLogger, err = micrologger.New(micrologger.Config{})
		if err != nil {
			return Config{}, microerror.Mask(err)
		}
		config.Logger = newLogger
	}

	// Create a new k8sclient that is used by all admitters.
	var k8sClient k8sclient.Interface
	{
		restConfig, err := restclient.InClusterConfig()
		if err != nil {
			return Config{}, microerror.Mask(err)
		}
		c := k8sclient.ClientsConfig{
			SchemeBuilder: k8sclient.SchemeBuilder{
				apiv1alpha2.AddToScheme,
				infrastructurev1alpha2.AddToScheme,
				securityv1alpha1.AddToScheme,
				releasev1alpha1.AddToScheme,
			},
			Logger: config.Logger,

			RestConfig: restConfig,
		}

		k8sClient, err = k8sclient.NewClients(c)
		if err != nil {
			return Config{}, microerror.Mask(err)
		}
		config.K8sClient = k8sClient
	}

	kingpin.Flag("address", "The address to listen on").Default(defaultAddress).StringVar(&config.Address)
	kingpin.Flag("admin-group", "Tenant Admin Target Group").Required().StringVar(&config.AdminGroup)
	kingpin.Flag("all-target-group", "View All Target Group").Required().StringVar(&config.AllTargetGroup)
	kingpin.Flag("availability-zones", "List of AWS availability zones").Required().StringVar(&config.AvailabilityZones)
	kingpin.Flag("docker-cidr", "Default CIDR from Docker").Required().StringVar(&config.DockerCIDR)
	kingpin.Flag("endpoint", "Default kubernetes endpoint").Required().StringVar(&config.Endpoint)
	kingpin.Flag("ipam-network-cidr", "Default CIDR from tenant cluster").Required().StringVar(&config.IPAMNetworkCIDR)
	kingpin.Flag("kubernetes-cluster-ip-range", "Default CIDR from Kubernetes").Required().StringVar(&config.KubernetesClusterIPRange)
	kingpin.Flag("master-instance-types", "List of AWS master instance types").Required().StringVar(&config.MasterInstanceTypes)
	kingpin.Flag("metrics-address", "The metrics address for Prometheus").Default(defaultMetricsAddress).StringVar(&config.MetricsAddress)
	kingpin.Flag("pod-cidr", "Default pod CIDR").Required().StringVar(&config.PodCIDR)
	kingpin.Flag("pod-subnet", "Default pod subnet").Required().StringVar(&config.PodSubnet)
	kingpin.Flag("region", "Default cluster region").Required().StringVar(&config.Region)
	kingpin.Flag("tls-cert-file", "File containing the certificate for HTTPS").Required().StringVar(&config.CertFile)
	kingpin.Flag("tls-key-file", "File containing the private key for HTTPS").Required().StringVar(&config.KeyFile)
	kingpin.Flag("worker-instance-types", "List of AWS worker instance types").Required().StringVar(&config.WorkerInstanceTypes)

	kingpin.Parse()

	return config, nil
}
