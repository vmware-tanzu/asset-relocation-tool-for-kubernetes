module gitlab.eng.vmware.com/marketplace-partner-eng/chart-mover/v2

go 1.16

require (
	github.com/bunniesandbeatings/goerkin v0.1.4-beta
	github.com/divideandconquer/go-merge v0.0.0-20160829212531-bc6b3a394b4e
	github.com/golang/protobuf v1.5.2 // indirect
	github.com/novln/docker-parser v1.0.0
	github.com/onsi/ginkgo v1.15.0
	github.com/onsi/gomega v1.10.5
	github.com/pkg/errors v0.9.1
	github.com/spf13/cobra v1.1.3
	gopkg.in/yaml.v2 v2.4.0
	helm.sh/helm/v3 v3.5.3
)

replace (
	github.com/docker/distribution => github.com/docker/distribution v0.0.0-20191216044856-a8371794149d
	github.com/docker/docker => github.com/docker/docker v17.12.0-ce-rc1.0.20200618181300-9dc6525e6118+incompatible

)
