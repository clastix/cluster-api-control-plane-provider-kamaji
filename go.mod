module github.com/clastix/cluster-api-control-plane-provider-kamaji

go 1.22.0

toolchain go1.22.1

require (
	github.com/clastix/kamaji v1.0.1-0.20241021192153-9ca69e91f9c9
	github.com/onsi/ginkgo/v2 v2.20.2
	github.com/onsi/gomega v1.34.2
	github.com/pkg/errors v0.9.1
	github.com/spf13/pflag v1.0.5
	k8s.io/api v0.31.1
	k8s.io/apimachinery v0.31.1
	k8s.io/client-go v0.31.1
	k8s.io/component-base v0.31.1
	sigs.k8s.io/cluster-api v1.8.4
	sigs.k8s.io/controller-runtime v0.19.0
)

require (
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/blang/semver/v4 v4.0.0 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/davecgh/go-spew v1.1.2-0.20180830191138-d8f796af33cc // indirect
	github.com/emicklei/go-restful/v3 v3.12.1 // indirect
	github.com/evanphx/json-patch/v5 v5.9.0 // indirect
	github.com/fsnotify/fsnotify v1.7.0 // indirect
	github.com/fxamacker/cbor/v2 v2.7.0 // indirect
	github.com/go-logr/logr v1.4.2 // indirect
	github.com/go-logr/zapr v1.3.0 // indirect
	github.com/go-openapi/jsonpointer v0.19.6 // indirect
	github.com/go-openapi/jsonreference v0.20.2 // indirect
	github.com/go-openapi/swag v0.22.4 // indirect
	github.com/go-task/slim-sprig/v3 v3.0.0 // indirect
	github.com/gobuffalo/flect v1.0.2 // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/golang/groupcache v0.0.0-20210331224755-41bb18bfe9da // indirect
	github.com/golang/protobuf v1.5.4 // indirect
	github.com/google/gnostic-models v0.6.8 // indirect
	github.com/google/go-cmp v0.6.0 // indirect
	github.com/google/gofuzz v1.2.0 // indirect
	github.com/google/pprof v0.0.0-20240827171923-fa2c70bbbfe5 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/imdario/mergo v0.3.13 // indirect
	github.com/josharian/intern v1.0.0 // indirect
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/mailru/easyjson v0.7.7 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.2 // indirect
	github.com/munnerz/goautoneg v0.0.0-20191010083416-a7dc8b61c822 // indirect
	github.com/prometheus/client_golang v1.19.1 // indirect
	github.com/prometheus/client_model v0.6.1 // indirect
	github.com/prometheus/common v0.55.0 // indirect
	github.com/prometheus/procfs v0.15.1 // indirect
	github.com/x448/float16 v0.8.4 // indirect
	go.uber.org/multierr v1.11.0 // indirect
	go.uber.org/zap v1.27.0 // indirect
	golang.org/x/exp v0.0.0-20240719175910-8a7402abbf56 // indirect
	golang.org/x/net v0.28.0 // indirect
	golang.org/x/oauth2 v0.21.0 // indirect
	golang.org/x/sys v0.24.0 // indirect
	golang.org/x/term v0.23.0 // indirect
	golang.org/x/text v0.17.0 // indirect
	golang.org/x/time v0.5.0 // indirect
	golang.org/x/tools v0.24.0 // indirect
	gomodules.xyz/jsonpatch/v2 v2.4.0 // indirect
	google.golang.org/protobuf v1.34.2 // indirect
	gopkg.in/inf.v0 v0.9.1 // indirect
	gopkg.in/yaml.v2 v2.4.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
	k8s.io/apiextensions-apiserver v0.31.1 // indirect
	k8s.io/klog/v2 v2.130.1 // indirect
	k8s.io/kube-openapi v0.0.0-20240228011516-70dd3763d340 // indirect
	k8s.io/utils v0.0.0-20240711033017-18e509b52bc8 // indirect
	sigs.k8s.io/json v0.0.0-20221116044647-bc3834ca7abd // indirect
	sigs.k8s.io/structured-merge-diff/v4 v4.4.1 // indirect
	sigs.k8s.io/yaml v1.4.0 // indirect
)

replace (
	k8s.io/api => k8s.io/api v0.31.1
	k8s.io/apiextensions-apiserver => k8s.io/apiextensions-apiserver v0.31.1
	k8s.io/apimachinery => k8s.io/apimachinery v0.31.1
	k8s.io/apiserver => k8s.io/apiserver v0.31.1
	k8s.io/cli-runtime => k8s.io/cli-runtime v0.31.1
	k8s.io/client-go => k8s.io/client-go v0.31.1
	k8s.io/cloud-provider => k8s.io/cloud-provider v0.31.1
	k8s.io/cluster-bootstrap => k8s.io/cluster-bootstrap v0.31.1
	k8s.io/code-generator => k8s.io/code-generator v0.31.1
	k8s.io/component-base => k8s.io/component-base v0.31.1
	k8s.io/component-helpers => k8s.io/component-helpers v0.31.1
	k8s.io/controller-manager => k8s.io/controller-manager v0.31.1
	k8s.io/cri-api => k8s.io/cri-api v0.31.1
	k8s.io/cri-client => k8s.io/cri-client v0.31.1
	k8s.io/csi-translation-lib => k8s.io/csi-translation-lib v0.31.1
	k8s.io/kube-aggregator => k8s.io/kube-aggregator v0.31.1
	k8s.io/kube-controller-manager => k8s.io/kube-controller-manager v0.31.1
	k8s.io/kube-proxy => k8s.io/kube-proxy v0.31.1
	k8s.io/kube-scheduler => k8s.io/kube-scheduler v0.31.1
	k8s.io/kubectl => k8s.io/kubectl v0.31.1
	k8s.io/kubelet => k8s.io/kubelet v0.31.1
	k8s.io/legacy-cloud-providers => k8s.io/legacy-cloud-providers v0.31.1
	k8s.io/metrics => k8s.io/metrics v0.31.1
	k8s.io/mount-utils => k8s.io/mount-utils v0.31.1
	k8s.io/pod-security-admission => k8s.io/pod-security-admission v0.31.1
	k8s.io/sample-apiserver => k8s.io/sample-apiserver v0.31.1
)
