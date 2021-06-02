module github.com/harmonicinc-com/joebot

replace (
	github.com/filebrowser/filebrowser/v2 => ../filebrowser
	github.com/ginuerzh/gost => ../ginuerzh/gost
	github.com/yudai/gotty => ../yudai/gotty
	github.com/yudai/hcl => ../yudai/hcl
)

go 1.16

require (
	github.com/DataDog/zstd v1.4.8 // indirect
	github.com/Sereal/Sereal v0.0.0-20200820125258-a016b7cda3f3 // indirect
	github.com/alecthomas/template v0.0.0-20190718012654-fb15b899a751 // indirect
	github.com/alecthomas/units v0.0.0-20210208195552-ff826a37aa15 // indirect
	github.com/asdine/storm v2.1.2+incompatible
	github.com/filebrowser/filebrowser/v2 v2.0.0-00010101000000-000000000000
	github.com/ginuerzh/gost v0.0.0-00010101000000-000000000000
	github.com/golang/snappy v0.0.3 // indirect
	github.com/hashicorp/yamux v0.0.0-20210316155119-a95892c5f864
	github.com/labstack/echo v3.3.10+incompatible
	github.com/labstack/gommon v0.3.0 // indirect
	github.com/myesui/uuid v1.0.0 // indirect
	github.com/onsi/ginkgo v1.16.3 // indirect
	github.com/onsi/gomega v1.13.0 // indirect
	github.com/pkg/errors v0.9.1
	github.com/pkg/sftp v1.13.0
	github.com/sirupsen/logrus v1.8.1
	github.com/spf13/afero v1.2.2
	github.com/twinj/uuid v1.0.0
	github.com/valyala/fasttemplate v1.2.1 // indirect
	github.com/xtaci/lossyconn v0.0.0-20200209145036-adba10fffc37 // indirect
	github.com/yudai/gotty v0.0.0-00010101000000-000000000000
	go.etcd.io/bbolt v1.3.6 // indirect
	golang.org/x/crypto v0.0.0-20210513164829-c07d793c2f9a
	golang.org/x/net v0.0.0-20210525063256-abc453219eb5
	gopkg.in/alecthomas/kingpin.v2 v2.2.6
	gopkg.in/stretchr/testify.v1 v1.2.2 // indirect
)
