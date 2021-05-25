module github.com/giantswarm/aws-admission-controller/v2

go 1.15

require (
	github.com/blang/semver v3.5.1+incompatible
	github.com/dylanmei/iso8601 v0.1.0
	github.com/dyson/certman v0.2.1
	github.com/giantswarm/apiextensions/v2 v2.6.2
	github.com/giantswarm/apiextensions/v3 v3.26.0
	github.com/giantswarm/backoff v0.2.0
	github.com/giantswarm/k8sclient/v5 v5.11.0
	github.com/giantswarm/microerror v0.3.0
	github.com/giantswarm/micrologger v0.5.0
	github.com/giantswarm/ruleengine v0.2.0
	github.com/giantswarm/to v0.3.0
	github.com/google/go-cmp v0.5.5
	github.com/prometheus/client_golang v1.10.0
	github.com/stretchr/testify v1.6.1 // indirect
	gopkg.in/alecthomas/kingpin.v2 v2.2.6
	k8s.io/api v0.18.19
	k8s.io/apiextensions-apiserver v0.18.18
	k8s.io/apimachinery v0.18.19
	k8s.io/client-go v0.18.18
	sigs.k8s.io/cluster-api v0.3.16
	sigs.k8s.io/controller-runtime v0.6.4
)

replace (
	github.com/coreos/etcd v3.3.10+incompatible => github.com/coreos/etcd v3.3.25+incompatible
	github.com/gogo/protobuf v1.3.1 => github.com/gogo/protobuf v1.3.2
	github.com/gorilla/websocket v1.4.0 => github.com/gorilla/websocket v1.4.2
	sigs.k8s.io/cluster-api => github.com/giantswarm/cluster-api v0.3.13-gs
)
