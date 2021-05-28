module gitlab.eng.vmware.com/marketplace-partner-eng/relok8s/v2

go 1.16

require (
	github.com/bunniesandbeatings/goerkin v0.1.4-beta
	github.com/divideandconquer/go-merge v0.0.0-20160829212531-bc6b3a394b4e
	github.com/golang/protobuf v1.5.2 // indirect
	github.com/google/go-containerregistry v0.5.1
	github.com/maxbrunsfeld/counterfeiter/v6 v6.4.1 // indirect
	github.com/onsi/ginkgo v1.15.0
	github.com/onsi/gomega v1.11.0
	github.com/pkg/errors v0.9.1
	github.com/spf13/cobra v1.1.3
	golang.org/x/sys v0.0.0-20210525143221-35b2ab0089ea // indirect
	golang.org/x/tools v0.1.2 // indirect
	gopkg.in/yaml.v2 v2.4.0
	gopkg.in/yaml.v3 v3.0.0-20200313102051-9f266ea9e77c
	helm.sh/helm/v3 v3.5.3
)

replace (
	github.com/docker/distribution => github.com/docker/distribution v0.0.0-20191216044856-a8371794149d
	github.com/docker/docker => github.com/docker/docker v17.12.0-ce-rc1.0.20200618181300-9dc6525e6118+incompatible
	gopkg.in/yaml.v3 => github.com/atomatt/yaml v0.0.0-20200403124456-7b932d16ab90
)
