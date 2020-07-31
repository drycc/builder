module github.com/drycc/builder

go 1.13

require (
	github.com/Sirupsen/logrus v0.0.0-00010101000000-000000000000 // indirect
	github.com/arschles/assert v0.0.0-20150820224400-6882f85ccdc7
	github.com/aws/aws-sdk-go v1.28.2
	github.com/codegangsta/cli v1.9.0
	github.com/docker/distribution v0.0.0-00010101000000-000000000000
	github.com/drycc/controller-sdk-go v0.0.0-20190417134318-39a6c81f21f3
	github.com/drycc/pkg v0.0.0-20190121053802-5c1dfa7b5446
	github.com/gorilla/mux v1.7.4 // indirect
	github.com/goware/urlx v0.3.1 // indirect
	github.com/kelseyhightower/envconfig v1.2.0
	github.com/pborman/uuid v1.2.0
	golang.org/x/crypto v0.0.0-20200220183623-bac4c82f6975
	gopkg.in/yaml.v2 v2.2.8
	k8s.io/api v0.18.6
	k8s.io/apimachinery v0.18.6
	k8s.io/client-go v0.18.6
)

replace (
	bitbucket.org/ww/goautoneg => github.com/rancher/goautoneg v0.0.0-20120707110453-a547fc61f48d
	github.com/Sirupsen/logrus => github.com/sirupsen/logrus v0.7.3
	github.com/docker/distribution => github.com/drycc/distribution v2.1.2-0.20160613220734-0afef00d5764+incompatible
	github.com/emicklei/go-restful => github.com/emicklei/go-restful v1.1.4-0.20151126145626-777bb3f19bca
	github.com/influxdb/influxdb => github.com/influxdata/influxdb v1.8.1
	github.com/microsoft/hcsshim => github.com/Microsoft/hcsshim v0.8.9
	github.com/renstrom/dedent => github.com/lithammer/dedent v1.1.0
	speter.net/go/exp/math/dec/inf => github.com/belua/inf v0.0.0-20151208101502-46a406493388
)
