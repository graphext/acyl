module github.com/dollarshaveclub/acyl

go 1.16

require (
	github.com/DataDog/datadog-go v4.4.0+incompatible
	github.com/DavidHuie/gomigrate v0.0.0-20160809001028-4004e6142040
	github.com/MakeNowJust/heredoc v0.0.0-20171113091838-e9091a26100e // indirect
	github.com/alecthomas/chroma v0.6.2
	github.com/docker/cli v20.10.7+incompatible
	github.com/docker/distribution v2.7.1+incompatible
	github.com/docker/docker v20.10.2+incompatible
	github.com/dollarshaveclub/furan v0.6.1-0.20210604153750-8def5a45ce21
	github.com/dollarshaveclub/furan/v2 v2.0.1
	github.com/dollarshaveclub/line v0.0.0-20171219191008-fc7a351a8b58
	github.com/dollarshaveclub/metahelm v0.7.1
	github.com/dollarshaveclub/pvc v1.0.0
	github.com/emirpasic/gods v1.12.0 // indirect
	github.com/gdamore/tcell v1.1.1
	github.com/ghodss/yaml v1.0.0
	github.com/go-pg/pg v6.6.7+incompatible
	github.com/golang/mock v1.5.0
	github.com/golang/snappy v0.0.3 // indirect
	github.com/google/go-cmp v0.5.6
	github.com/google/go-github v17.0.0+incompatible
	github.com/google/go-github/v30 v30.1.0 // indirect
	github.com/google/go-github/v38 v38.1.0
	github.com/google/uuid v1.2.0
	github.com/gorilla/mux v1.8.0
	github.com/gorilla/securecookie v1.1.1
	github.com/gorilla/sessions v1.2.1
	github.com/hashicorp/errwrap v1.1.0 // indirect
	github.com/hashicorp/go-cleanhttp v0.5.2 // indirect
	github.com/hashicorp/go-multierror v1.1.1
	github.com/hashicorp/go-retryablehttp v0.6.8 // indirect
	github.com/hashicorp/vault/api v1.0.5-0.20210318200847-4cf7593de37a // indirect
	github.com/hashicorp/vault/sdk v0.1.14-0.20210318200847-4cf7593de37a // indirect
	github.com/imdario/mergo v0.3.11
	github.com/jinzhu/inflection v0.0.0-20170102125226-1c35d901db3d // indirect
	github.com/jmoiron/sqlx v1.3.1
	github.com/lib/pq v1.10.0
	github.com/mattn/go-runewidth v0.0.9
	github.com/mholt/archiver v3.1.1+incompatible
	github.com/mitchellh/go-homedir v1.1.0
	github.com/nlopes/slack v0.1.0
	github.com/palantir/go-githubapp v0.9.2-0.20210830144646-08ca97a77f90
	github.com/pierrec/lz4 v2.6.0+incompatible // indirect
	github.com/pkg/errors v0.9.1
	github.com/rivo/tview v0.0.0-20190113120821-e5e361b9d790
	github.com/rs/zerolog v1.18.0
	github.com/shurcooL/githubv4 v0.0.0-20191127044304-8f68eb5628d0
	github.com/spf13/afero v1.6.0
	github.com/spf13/cobra v1.2.1
	github.com/spf13/pflag v1.0.5
	github.com/ulikunitz/xz v0.5.8 // indirect
	golang.org/x/crypto v0.0.0-20210513164829-c07d793c2f9a
	golang.org/x/net v0.0.0-20210520170846-37e1c6afe023
	golang.org/x/oauth2 v0.0.0-20210402161424-2e8d93401602
	golang.org/x/sync v0.0.0-20210220032951-036812b2e83c
	golang.org/x/sys v0.0.0-20210915083310-ed5796bab164 // indirect
	google.golang.org/grpc v1.38.0
	gopkg.in/DataDog/dd-trace-go.v1 v1.29.0
	gopkg.in/check.v1 v1.0.0-20201130134442-10cb98267c6c
	gopkg.in/jfontan/go-billy-desfacer.v0 v0.0.0-20190109211405-e5f0f2cddac1
	gopkg.in/src-d/go-billy.v4 v4.3.0
	gopkg.in/src-d/go-git.v4 v4.8.1
	gopkg.in/yaml.v2 v2.4.0
	helm.sh/helm/v3 v3.7.0
	k8s.io/api v0.22.1
	k8s.io/apimachinery v0.22.1
	k8s.io/cli-runtime v0.22.1
	k8s.io/client-go v0.22.1
	sigs.k8s.io/kustomize/kyaml v0.11.0
)

replace (
	k8s.io/api => k8s.io/api v0.22.1
	k8s.io/apiextensions-apiserver => k8s.io/apiextensions-apiserver v0.22.1
	k8s.io/apimachinery => k8s.io/apimachinery v0.22.1
	k8s.io/apiserver => k8s.io/apiserver v0.22.1
	k8s.io/cli-runtime => k8s.io/cli-runtime v0.22.1
	k8s.io/client-go => k8s.io/client-go v0.22.1
	k8s.io/cloud-provider => k8s.io/cloud-provider v0.22.1
	k8s.io/cluster-bootstrap => k8s.io/cluster-bootstrap v0.22.1
	k8s.io/code-generator => k8s.io/code-generator v0.22.1
	k8s.io/component-base => k8s.io/component-base v0.22.1
	k8s.io/component-helpers => k8s.io/component-helpers v0.22.1
	k8s.io/controller-manager => k8s.io/controller-manager v0.22.1
	k8s.io/cri-api => k8s.io/cri-api v0.22.1
	k8s.io/csi-translation-lib => k8s.io/csi-translation-lib v0.22.1
	k8s.io/kube-aggregator => k8s.io/kube-aggregator v0.22.1
	k8s.io/kube-controller-manager => k8s.io/kube-controller-manager v0.22.1
	k8s.io/kube-proxy => k8s.io/kube-proxy v0.22.1
	k8s.io/kube-scheduler => k8s.io/kube-scheduler v0.22.1
	k8s.io/kubectl => k8s.io/kubectl v0.22.1
	k8s.io/kubelet => k8s.io/kubelet v0.22.1
	k8s.io/kubernetes => k8s.io/kubernetes v1.21.3
	k8s.io/legacy-cloud-providers => k8s.io/legacy-cloud-providers v0.22.1
	k8s.io/metrics => k8s.io/metrics v0.22.1
	k8s.io/mount-utils => k8s.io/mount-utils v0.22.1
	k8s.io/sample-apiserver => k8s.io/sample-apiserver v0.22.1
)

replace (
	github.com/docker/distribution => github.com/distribution/distribution v2.7.1+incompatible
	github.com/docker/docker => github.com/docker/docker v20.10.5+incompatible
	github.com/google/go-github => github.com/google/go-github/v30 v30.1.1-0.20200328133946-34cb1d623f03
	github.com/hashicorp/vault/api => github.com/hashicorp/vault/api v1.0.5-0.20200818184811-84f6d9a065c2
	gonum.org/v1/gonum => gonum.org/v1/gonum v0.9.1
	gopkg.in/jfontan/go-billy-desfacer.v0 v0.0.0-20190109211405-e5f0f2cddac1 => github.com/bkeroackdsc/go-billy-desfacer v0.0.0-20190109211405-e5f0f2cddac1
)

// for Furan 2
replace github.com/containerd/containerd v1.4.0-0 => github.com/containerd/containerd v1.4.0-beta.1.0.20200624184620-1127ffc7400e
