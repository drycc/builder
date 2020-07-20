module github.com/drycc/builder

go 1.13

require (
	github.com/Azure/azure-sdk-for-go v35.0.0+incompatible // indirect
	github.com/PuerkitoBio/purell v1.1.1 // indirect
	github.com/PuerkitoBio/urlesc v0.0.0-20170810143723-de5bf2ad4578 // indirect
	github.com/Sirupsen/logrus v0.7.3 // indirect
	github.com/arschles/assert v0.0.0-20150820224400-6882f85ccdc7
	github.com/aws/aws-sdk-go v1.28.2
	github.com/blang/semver v3.5.0+incompatible // indirect
	github.com/codegangsta/cli v1.9.0
	github.com/cyphar/filepath-securejoin v0.2.2 // indirect
	github.com/docker/distribution v0.0.0-00010101000000-000000000000
	github.com/docker/docker v1.4.2-0.20150722082610-0f5c9d301b9b // indirect
	github.com/docker/go-units v0.4.0 // indirect
	github.com/drycc/controller-sdk-go v0.0.0-20190417134318-39a6c81f21f3
	github.com/drycc/pkg v0.0.0-20190121053802-5c1dfa7b5446
	github.com/emicklei/go-restful v2.9.5+incompatible // indirect
	github.com/ghodss/yaml v1.0.0 // indirect
	github.com/go-ini/ini v1.8.6 // indirect
	github.com/google/cadvisor v0.35.0 // indirect
	github.com/google/gofuzz v1.1.0 // indirect
	github.com/gorilla/context v1.1.1 // indirect
	github.com/gorilla/mux v1.7.0 // indirect
	github.com/goware/urlx v0.0.0-20160722204212-8bb4a2e4339f // indirect
	github.com/juju/ratelimit v0.0.0-20151125201925-77ed1c8a0121 // indirect
	github.com/kelseyhightower/envconfig v1.2.0
	github.com/mitchellh/mapstructure v1.1.2 // indirect
	github.com/ncw/swift v1.0.20-0.20151102203822-c54732e87b0b // indirect
	github.com/opencontainers/runc v0.0.7
	github.com/opencontainers/runtime-spec v1.0.2 // indirect
	github.com/pborman/uuid v1.2.0
	github.com/prometheus/client_golang v1.0.0 // indirect
	github.com/prometheus/client_model v0.2.0 // indirect
	github.com/spf13/pflag v1.0.5 // indirect
	github.com/ugorji/go v0.0.0-20160211161415-f4485b318aad // indirect
	golang.org/x/crypto v0.0.0-20200220183623-bac4c82f6975
	golang.org/x/net v0.0.0-20191004110552-13f9640d40b9 // indirect
	google.golang.org/api v0.6.1-0.20190607001116-5213b8090861 // indirect
	google.golang.org/cloud v0.0.0-20151119220103-975617b05ea8 // indirect
	google.golang.org/grpc v1.26.0 // indirect
	gopkg.in/yaml.v2 v2.2.8
	k8s.io/kubernetes v1.2.4
	speter.net/go/exp/math/dec/inf v0.0.0-00010101000000-000000000000 // indirect
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
