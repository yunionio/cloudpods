module yunion.io/x/onecloud

go 1.18

require (
	bazil.org/fuse v0.0.0-20180421153158-65cc252bf669
	cloud.google.com/go/storage v1.10.0
	github.com/360EntSecGroup-Skylar/excelize v1.4.0
	github.com/Azure/azure-sdk-for-go v36.1.0+incompatible
	github.com/Azure/go-autorest/autorest v0.9.6
	github.com/Azure/go-autorest/autorest/azure/auth v0.4.2
	github.com/LeeEirc/terminalparser v0.0.0-20220328021224-de16b7643ea4
	github.com/Masterminds/sprig v2.22.0+incompatible
	github.com/Microsoft/azure-vhd-utils v0.0.0-20181115010904-44cbada2ece3
	github.com/aliyun/alibaba-cloud-sdk-go v1.61.684
	github.com/aliyun/aliyun-oss-go-sdk v2.0.4+incompatible
	github.com/anacrolix/torrent v0.0.0-20181129073333-cc531b8c4a80
	github.com/aokoli/goutils v1.0.1
	github.com/aws/aws-sdk-go v1.39.0
	github.com/basgys/goxml2json v1.1.1-0.20181031222924-996d9fc8d313
	github.com/benbjohnson/clock v1.0.0
	github.com/bitly/go-simplejson v0.5.0
	github.com/c-bata/go-prompt v0.2.4
	github.com/cheggaaa/pb/v3 v3.0.8
	github.com/coredns/coredns v1.3.0
	github.com/creack/pty v1.1.11
	github.com/fatih/color v1.13.0
	github.com/fernet/fernet-go v0.0.0-20180830025343-9eac43b88a5e
	github.com/flynn/go-shlex v0.0.0-20150515145356-3f9db97f8568 // indirect
	github.com/fsnotify/fsnotify v1.4.9
	github.com/ghodss/yaml v1.0.0
	github.com/gin-gonic/gin v1.7.7
	github.com/go-yaml/yaml v2.1.0+incompatible
	github.com/golang-plus/errors v1.0.0
	github.com/golang-plus/uuid v1.0.0
	github.com/golang/mock v1.4.4
	github.com/golang/protobuf v1.5.2
	github.com/google/gopacket v1.1.17
	github.com/googollee/go-socket.io v0.0.0-20181214084611-0ad7206c347a
	github.com/gorilla/mux v1.7.0
	github.com/gorilla/websocket v1.4.1
	github.com/gosuri/uitable v0.0.0-20160404203958-36ee7e946282
	github.com/hako/durafmt v0.0.0-20180520121703-7b7ae1e72ead
	github.com/huaweicloud/huaweicloud-sdk-go v1.0.26
	github.com/jaypipes/ghw v0.9.1
	github.com/jdcloud-api/jdcloud-sdk-go v1.55.0
	github.com/koding/websocketproxy v0.0.0-20181220232114-7ed82d81a28c
	github.com/lestrrat-go/jwx v1.0.2
	github.com/lestrrat/go-jwx v0.0.0-20180221005942-b7d4802280ae
	github.com/libvirt/libvirt-go-xml v5.2.0+incompatible
	github.com/ma314smith/signedxml v0.0.0-20210628192057-abc5b481ae1c
	github.com/mattn/go-sqlite3 v1.14.12
	github.com/mdlayher/arp v0.0.0-20190313224443-98a83c8a2717
	github.com/mdlayher/ethernet v0.0.0-20190606142754-0394541c37b7
	github.com/mdlayher/raw v0.0.0-20190606144222-a54781e5f38f
	github.com/mholt/caddy v0.10.11
	github.com/miekg/dns v1.1.25
	github.com/minio/minio-go v6.0.14+incompatible
	github.com/mozillazg/go-pinyin v0.19.0
	github.com/opentracing-contrib/go-observer v0.0.0-20170622124052-a52f23424492 // indirect
	github.com/opentracing/opentracing-go v1.0.2 // indirect
	github.com/openzipkin/zipkin-go-opentracing v0.3.4 // indirect
	github.com/pierrec/lz4/v4 v4.1.15
	github.com/pkg/errors v0.9.1
	github.com/pquerna/otp v1.2.0
	github.com/sergi/go-diff v1.2.0
	github.com/serialx/hashring v0.0.0-20180504054112-49a4782e9908
	github.com/sevlyar/go-daemon v0.1.5
	github.com/shirou/gopsutil v3.21.11+incompatible
	github.com/shirou/gopsutil/v3 v3.22.10
	github.com/sirupsen/logrus v1.9.0
	github.com/skip2/go-qrcode v0.0.0-20190110000554-dc11ecdae0a9
	github.com/smartystreets/goconvey v1.7.2
	github.com/stretchr/testify v1.8.1
	github.com/tatsushid/go-fastping v0.0.0-20160109021039-d7bb493dee3e
	github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common v1.0.413
	github.com/tencentyun/cos-go-sdk-v5 v0.7.24
	github.com/tjfoc/gmsm v1.4.1
	github.com/tredoe/osutil v0.0.0-20161130133508-7d3ee1afa71c
	github.com/vishvananda/netlink v1.0.0
	github.com/vishvananda/netns v0.0.0-20191106174202-0a2b9b5464df // indirect
	github.com/vmihailenco/msgpack v4.0.4+incompatible
	github.com/vmware/govmomi v0.20.1
	go.etcd.io/etcd/api/v3 v3.5.0
	go.etcd.io/etcd/client/v3 v3.5.0
	golang.org/x/crypto v0.0.0-20220411220226-7b82a4e95df4
	golang.org/x/net v0.0.0-20220418201149-a630d4f3e7a2
	golang.org/x/oauth2 v0.0.0-20210514164344-f6687ab2804c
	golang.org/x/sync v0.0.0-20210220032951-036812b2e83c
	golang.org/x/sys v0.0.0-20220829200755-d48e67d00261
	golang.org/x/text v0.3.7
	golang.org/x/time v0.0.0-20191024005414-555d28b269f0
	golang.org/x/xerrors v0.0.0-20200804184101-5ec99f83aff1
	golang.zx2c4.com/wireguard/wgctrl v0.0.0-20220916014741-473347a5e6e3
	google.golang.org/grpc v1.38.0
	google.golang.org/protobuf v1.27.1
	gopkg.in/fatih/set.v0 v0.2.1
	gopkg.in/ldap.v3 v3.0.3
	gopkg.in/yaml.v2 v2.4.0
	k8s.io/api v0.19.3
	k8s.io/apimachinery v0.19.3
	k8s.io/client-go v0.19.3
	k8s.io/cluster-bootstrap v0.19.3
	moul.io/http2curl/v2 v2.3.0
	yunion.io/x/log v1.0.1-0.20230411060016-feb3f46ab361
	yunion.io/x/ovsdb v0.0.0-20230306173834-f164f413a900
	yunion.io/x/pkg v1.0.1-0.20230912084455-1393f31347db
	yunion.io/x/s3cli v0.0.0-20190917004522-13ac36d8687e
	yunion.io/x/structarg v0.0.0-20220312084958-9c6c79c7d1c6
)

require (
	github.com/google/uuid v1.3.0
	yunion.io/x/executor v0.0.0-20230705125604-c5ac3141db32
	yunion.io/x/jsonutils v1.0.1-0.20230613121553-0f3b41e2ef19
	yunion.io/x/sqlchemy v1.1.2-0.20231011114043-0d203453d3e2
)

require (
	cloud.google.com/go v0.65.0 // indirect
	github.com/Azure/go-autorest/autorest/adal v0.8.2 // indirect
	github.com/Azure/go-autorest/autorest/azure/cli v0.3.1 // indirect
	github.com/Azure/go-autorest/autorest/date v0.2.0 // indirect
	github.com/Azure/go-autorest/autorest/to v0.4.0 // indirect
	github.com/Azure/go-autorest/logger v0.1.0 // indirect
	github.com/Azure/go-autorest/tracing v0.5.0 // indirect
	github.com/ClickHouse/clickhouse-go v1.5.4 // indirect
	github.com/DataDog/dd-trace-go v0.6.1 // indirect
	github.com/DataDog/zstd v1.3.4 // indirect
	github.com/Masterminds/goutils v1.1.0 // indirect
	github.com/Masterminds/semver v1.5.0 // indirect
	github.com/RoaringBitmap/roaring v0.4.16 // indirect
	github.com/Shopify/sarama v1.20.0 // indirect
	github.com/Shopify/toxiproxy v2.1.4+incompatible // indirect
	github.com/StackExchange/wmi v1.2.1 // indirect
	github.com/VividCortex/ewma v1.1.1 // indirect
	github.com/anacrolix/dht v0.0.0-20181129074040-b09db78595aa // indirect
	github.com/anacrolix/go-libutp v0.0.0-20180808010927-aebbeb60ea05 // indirect
	github.com/anacrolix/log v0.0.0-20180808012509-286fcf906b48 // indirect
	github.com/anacrolix/missinggo v0.0.0-20181129073415-3237bf955fed // indirect
	github.com/anacrolix/mmsg v0.0.0-20180808012353-5adb2c1127c0 // indirect
	github.com/anacrolix/sync v0.0.0-20180808010631-44578de4e778 // indirect
	github.com/anacrolix/utp v0.0.0-20180219060659-9e0e1d1d0572 // indirect
	github.com/apache/thrift v0.12.0 // indirect
	github.com/baiyubin/aliyun-sts-go-sdk v0.0.0-20180326062324-cfa1a18b161f // indirect
	github.com/beevik/etree v1.1.0 // indirect
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/bmizerany/assert v0.0.0-20160611221934-b7ed37b82869 // indirect
	github.com/boltdb/bolt v1.3.1 // indirect
	github.com/boombuler/barcode v1.0.1-0.20190219062509-6c824513bacc // indirect
	github.com/bradfitz/iter v0.0.0-20140124041915-454541ec3da2 // indirect
	github.com/cespare/xxhash/v2 v2.1.2 // indirect
	github.com/cloudflare/golz4 v0.0.0-20150217214814-ef862a3cdc58 // indirect
	github.com/coreos/go-semver v0.3.0 // indirect
	github.com/coreos/go-systemd/v22 v22.3.2 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/dgrijalva/jwt-go v3.2.0+incompatible // indirect
	github.com/dimchansky/utfbom v1.1.0 // indirect
	github.com/dnaeon/go-vcr v1.1.0 // indirect
	github.com/dnstap/golang-dnstap v0.0.0-20170829151710-2cf77a2b5e11 // indirect
	github.com/dustin/go-humanize v1.0.0 // indirect
	github.com/eapache/go-resiliency v1.1.0 // indirect
	github.com/eapache/go-xerial-snappy v0.0.0-20180814174437-776d5712da21 // indirect
	github.com/eapache/queue v1.1.0 // indirect
	github.com/edsrzf/mmap-go v1.0.0 // indirect
	github.com/elgatito/upnp v0.0.0-20180711183757-2f244d205f9a // indirect
	github.com/farsightsec/golang-framestream v0.0.0-20181102145529-8a0cb8ba8710 // indirect
	github.com/frankban/quicktest v1.14.3 // indirect
	github.com/gin-contrib/sse v0.1.0 // indirect
	github.com/glycerine/go-unsnap-stream v0.0.0-20181221182339-f9677308dec2 // indirect
	github.com/go-logfmt/logfmt v0.5.0 // indirect
	github.com/go-logr/logr v1.2.3 // indirect
	github.com/go-ole/go-ole v1.2.6 // indirect
	github.com/go-playground/locales v0.13.0 // indirect
	github.com/go-playground/universal-translator v0.17.0 // indirect
	github.com/go-playground/validator/v10 v10.4.1 // indirect
	github.com/go-sql-driver/mysql v1.6.0 // indirect
	github.com/gofrs/uuid v4.1.0+incompatible // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/golang/groupcache v0.0.0-20200121045136-8c9f03a8e57e // indirect
	github.com/golang/snappy v0.0.0-20180518054509-2e65f85255db // indirect
	github.com/google/btree v1.0.0 // indirect
	github.com/google/go-querystring v1.0.0 // indirect
	github.com/google/gofuzz v1.1.0 // indirect
	github.com/googleapis/gax-go/v2 v2.0.5 // indirect
	github.com/googleapis/gnostic v0.4.1 // indirect
	github.com/googollee/go-engine.io v0.0.0-20180829091931-e2f255711dcb // indirect
	github.com/gopherjs/gopherjs v0.0.0-20181017120253-0766667cb4d1 // indirect
	github.com/grpc-ecosystem/grpc-opentracing v0.0.0-20180507213350-8e809c8a8645 // indirect
	github.com/huandu/xstrings v1.2.0 // indirect
	github.com/imdario/mergo v0.3.6 // indirect
	github.com/jmespath/go-jmespath v0.4.0 // indirect
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/jstemmer/go-junit-report v0.9.1 // indirect
	github.com/jtolds/gls v4.20.0+incompatible // indirect
	github.com/kardianos/osext v0.0.0-20190222173326-2bc1f35cddc0 // indirect
	github.com/leodido/go-urn v1.2.0 // indirect
	github.com/lestrrat-go/iter v0.0.0-20200422075355-fc1769541911 // indirect
	github.com/lestrrat-go/pdebug v0.0.0-20200204225717-4d6bd78da58d // indirect
	github.com/lestrrat/go-pdebug v0.0.0-20180220043741-569c97477ae8 // indirect
	github.com/lufia/plan9stats v0.0.0-20211012122336-39d0f177ccd0 // indirect
	github.com/mattn/go-colorable v0.1.9 // indirect
	github.com/mattn/go-isatty v0.0.14 // indirect
	github.com/mattn/go-runewidth v0.0.13 // indirect
	github.com/mattn/go-tty v0.0.0-20181127064339-e4f871175a2f // indirect
	github.com/matttproud/golang_protobuf_extensions v1.0.1 // indirect
	github.com/mgutz/ansi v0.0.0-20170206155736-9520e82c474b // indirect
	github.com/minio/minio-go/v6 v6.0.33 // indirect
	github.com/mitchellh/copystructure v1.0.0 // indirect
	github.com/mitchellh/go-homedir v1.1.0 // indirect
	github.com/mitchellh/reflectwalk v1.0.0 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.2 // indirect
	github.com/mohae/deepcopy v0.0.0-20170929034955-c48cc78d4826 // indirect
	github.com/mozillazg/go-httpheader v0.2.1 // indirect
	github.com/mschoch/smat v0.0.0-20160514031455-90eadee771ae // indirect
	github.com/op/go-logging v0.0.0-20160315200505-970db520ece7 // indirect
	github.com/philhofer/fwd v1.0.0 // indirect
	github.com/pierrec/lz4 v2.6.1+incompatible // indirect
	github.com/pkg/term v1.0.0 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/power-devops/perfstat v0.0.0-20210106213030-5aafc221ea8c // indirect
	github.com/prometheus/client_golang v1.12.1 // indirect
	github.com/prometheus/client_model v0.2.0 // indirect
	github.com/prometheus/common v0.32.1 // indirect
	github.com/prometheus/procfs v0.7.3 // indirect
	github.com/rcrowley/go-metrics v0.0.0-20181016184325-3113b8401b8a // indirect
	github.com/rivo/uniseg v0.2.0 // indirect
	github.com/ryszard/goskiplist v0.0.0-20150312221310-2dfbae5fcf46 // indirect
	github.com/satori/go.uuid v1.2.0 // indirect
	github.com/smartystreets/assertions v1.2.0 // indirect
	github.com/spaolacci/murmur3 v1.1.0 // indirect
	github.com/spf13/pflag v1.0.5 // indirect
	github.com/stretchr/objx v0.5.0 // indirect
	github.com/texttheater/golang-levenshtein v0.0.0-20180516184445-d188e65d659e // indirect
	github.com/tinylib/msgp v1.1.0 // indirect
	github.com/tklauser/go-sysconf v0.3.10 // indirect
	github.com/tklauser/numcpus v0.4.0 // indirect
	github.com/tredoe/osutil/v2 v2.0.0-rc.16 // indirect
	github.com/ugorji/go/codec v1.1.7 // indirect
	github.com/willf/bitset v1.1.9 // indirect
	github.com/willf/bloom v2.0.3+incompatible // indirect
	github.com/yusufpapurcu/wmi v1.2.2 // indirect
	go.etcd.io/etcd/client/pkg/v3 v3.5.0 // indirect
	go.opencensus.io v0.22.4 // indirect
	go.uber.org/atomic v1.7.0 // indirect
	go.uber.org/multierr v1.6.0 // indirect
	go.uber.org/zap v1.17.0 // indirect
	golang.org/x/lint v0.0.0-20210508222113-6edffad5e616 // indirect
	golang.org/x/mod v0.4.2 // indirect
	golang.org/x/term v0.0.0-20210927222741-03fcf44c2211 // indirect
	golang.org/x/tools v0.1.2 // indirect
	google.golang.org/api v0.30.0 // indirect
	google.golang.org/appengine v1.6.6 // indirect
	google.golang.org/genproto v0.0.0-20210602131652-f16073e35f0c // indirect
	gopkg.in/asn1-ber.v1 v1.0.0-20181015200546-f715ec2f112d // indirect
	gopkg.in/check.v1 v1.0.0-20201130134442-10cb98267c6c // indirect
	gopkg.in/inf.v0 v0.9.1 // indirect
	gopkg.in/ini.v1 v1.62.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
	k8s.io/klog/v2 v2.2.0 // indirect
	k8s.io/utils v0.0.0-20200729134348-d5654de09c73 // indirect
	sigs.k8s.io/structured-merge-diff/v4 v4.0.1 // indirect
	sigs.k8s.io/yaml v1.2.0 // indirect
)

replace (
	github.com/go-logr/logr => github.com/go-logr/logr v0.4.0
	github.com/jaypipes/ghw => github.com/zexi/ghw v0.9.1
)
