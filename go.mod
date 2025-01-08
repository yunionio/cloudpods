module yunion.io/x/onecloud

go 1.21

require (
	bazil.org/fuse v0.0.0-20180421153158-65cc252bf669
	github.com/360EntSecGroup-Skylar/excelize v1.4.0
	github.com/LeeEirc/terminalparser v0.0.0-20240205084113-fbf78c8480f2
	github.com/aliyun/alibaba-cloud-sdk-go v1.61.684
	github.com/anacrolix/torrent v0.0.0-20181129073333-cc531b8c4a80
	github.com/benbjohnson/clock v1.0.0
	github.com/bitly/go-simplejson v0.5.0
	github.com/c-bata/go-prompt v0.2.4
	github.com/cheggaaa/pb/v3 v3.0.8
	github.com/coredns/coredns v1.3.0
	github.com/coreos/go-iptables v0.6.0
	github.com/creack/pty v1.1.18
	github.com/docker/docker v17.12.0-ce-rc1.0.20200916142827-bd33bbf0497b+incompatible
	github.com/docker/spdystream v0.0.0-20160310174837-449fdfce4d96
	github.com/fernet/fernet-go v0.0.0-20180830025343-9eac43b88a5e
	github.com/fsnotify/fsnotify v1.4.9
	github.com/ghodss/yaml v1.0.0
	github.com/gin-gonic/gin v1.7.7
	github.com/go-yaml/yaml v2.1.0+incompatible
	github.com/golang-plus/uuid v1.0.0
	github.com/golang/mock v1.4.4
	github.com/google/cadvisor v0.38.5
	github.com/google/gopacket v1.1.17
	github.com/google/uuid v1.6.0
	github.com/googollee/go-socket.io v0.0.0-20181214084611-0ad7206c347a
	github.com/gorilla/mux v1.8.0
	github.com/gorilla/websocket v1.4.1
	github.com/gosuri/uitable v0.0.0-20160404203958-36ee7e946282
	github.com/hako/durafmt v0.0.0-20180520121703-7b7ae1e72ead
	github.com/hugozhu/godingtalk v1.0.6
	github.com/influxdata/influxql v1.1.0
	github.com/influxdata/promql/v2 v2.12.0
	github.com/jaypipes/ghw v0.11.0
	github.com/koding/websocketproxy v0.0.0-20181220232114-7ed82d81a28c
	github.com/lestrrat-go/jwx v1.0.2
	github.com/lestrrat/go-jwx v0.0.0-20180221005942-b7d4802280ae
	github.com/libvirt/libvirt-go-xml v5.2.0+incompatible
	github.com/mattn/go-sqlite3 v1.14.19
	github.com/mdlayher/arp v0.0.0-20190313224443-98a83c8a2717
	github.com/mdlayher/ethernet v0.0.0-20190606142754-0394541c37b7
	github.com/mdlayher/packet v1.1.2
	github.com/mholt/caddy v0.10.11
	github.com/miekg/dns v1.1.25
	github.com/minio/minio-go v6.0.14+incompatible
	github.com/mitchellh/go-wordwrap v1.0.1
	github.com/pierrec/lz4/v4 v4.1.15
	github.com/pkg/errors v0.9.1
	github.com/pkg/sftp v1.13.6
	github.com/pquerna/otp v1.2.0
	github.com/satori/go.uuid v1.2.0
	github.com/sergi/go-diff v1.2.0
	github.com/serialx/hashring v0.0.0-20180504054112-49a4782e9908
	github.com/sevlyar/go-daemon v0.1.5
	github.com/shirou/gopsutil v3.21.11+incompatible
	github.com/shirou/gopsutil/v3 v3.22.10
	github.com/sirupsen/logrus v1.9.0
	github.com/skip2/go-qrcode v0.0.0-20190110000554-dc11ecdae0a9
	github.com/smartystreets/goconvey v1.7.2
	github.com/stretchr/testify v1.9.0
	github.com/tatsushid/go-fastping v0.0.0-20160109021039-d7bb493dee3e
	github.com/tjfoc/gmsm v1.4.1
	github.com/tredoe/osutil v1.5.0
	github.com/vishvananda/netlink v1.1.0
	github.com/vishvananda/netns v0.0.5-0.20240412164733-9469873f4601
	github.com/vmihailenco/msgpack v4.0.4+incompatible
	github.com/xuri/excelize/v2 v2.7.1
	github.com/zexi/influxql-to-metricsql v0.1.0
	go.etcd.io/etcd/api/v3 v3.5.0
	go.etcd.io/etcd/client/v3 v3.5.0
	golang.org/x/crypto v0.19.0
	golang.org/x/net v0.21.0
	golang.org/x/sync v0.6.0
	golang.org/x/sys v0.17.0
	golang.org/x/text v0.14.0
	golang.org/x/time v0.5.0
	golang.org/x/xerrors v0.0.0-20231012003039-104605ab7028
	golang.zx2c4.com/wireguard/wgctrl v0.0.0-20230215201556-9c5414ab4bde
	google.golang.org/grpc v1.62.0
	google.golang.org/protobuf v1.32.0
	gopkg.in/fatih/set.v0 v0.2.1
	gopkg.in/ldap.v3 v3.0.3
	gopkg.in/mail.v2 v2.3.1
	gopkg.in/yaml.v2 v2.4.0
	k8s.io/api v0.19.3
	k8s.io/apimachinery v0.19.3
	k8s.io/client-go v0.19.3
	k8s.io/cluster-bootstrap v0.19.3
	k8s.io/cri-api v0.22.17
	k8s.io/klog/v2 v2.20.0
	moul.io/http2curl/v2 v2.3.0
	yunion.io/x/cloudmux v0.3.10-0-alpha.1.0.20250108102611-1b422fb27e8b
	yunion.io/x/executor v0.0.0-20241205080005-48f5b1212256
	yunion.io/x/jsonutils v1.0.1-0.20240930100528-1671a2d0d22f
	yunion.io/x/log v1.0.1-0.20240305175729-7cf2d6cd5a91
	yunion.io/x/ovsdb v0.0.0-20230306173834-f164f413a900
	yunion.io/x/pkg v1.10.3
	yunion.io/x/s3cli v0.0.0-20241221171442-1c11599d28e1
	yunion.io/x/sqlchemy v1.1.3-0.20240926163039-d41512b264e1
	yunion.io/x/structarg v0.0.0-20231017124457-df4d5009457c
)

require (
	cloud.google.com/go v0.112.1 // indirect
	cloud.google.com/go/compute v1.24.0 // indirect
	cloud.google.com/go/compute/metadata v0.2.3 // indirect
	cloud.google.com/go/iam v1.1.6 // indirect
	cloud.google.com/go/storage v1.39.1 // indirect
	gitee.com/chunanyong/dm v1.8.14 // indirect
	github.com/Azure/azure-sdk-for-go v36.1.0+incompatible // indirect
	github.com/Azure/go-ansiterm v0.0.0-20230124172434-306776ec8161 // indirect
	github.com/Azure/go-autorest v14.2.0+incompatible // indirect
	github.com/Azure/go-autorest/autorest v0.9.6 // indirect
	github.com/Azure/go-autorest/autorest/adal v0.8.2 // indirect
	github.com/Azure/go-autorest/autorest/azure/auth v0.4.2 // indirect
	github.com/Azure/go-autorest/autorest/azure/cli v0.3.1 // indirect
	github.com/Azure/go-autorest/autorest/date v0.2.0 // indirect
	github.com/Azure/go-autorest/logger v0.1.0 // indirect
	github.com/Azure/go-autorest/tracing v0.5.0 // indirect
	github.com/ClickHouse/clickhouse-go v1.5.4 // indirect
	github.com/DataDog/dd-trace-go v0.6.1 // indirect
	github.com/DataDog/zstd v1.3.4 // indirect
	github.com/Microsoft/azure-vhd-utils v0.0.0-20181115010904-44cbada2ece3 // indirect
	github.com/Microsoft/go-winio v0.4.15 // indirect
	github.com/RoaringBitmap/roaring v0.4.16 // indirect
	github.com/Shopify/sarama v1.20.0 // indirect
	github.com/Shopify/toxiproxy v2.1.4+incompatible // indirect
	github.com/StackExchange/wmi v1.2.1 // indirect
	github.com/VividCortex/ewma v1.1.1 // indirect
	github.com/aliyun/aliyun-oss-go-sdk v2.0.4+incompatible // indirect
	github.com/anacrolix/dht v0.0.0-20181129074040-b09db78595aa // indirect
	github.com/anacrolix/go-libutp v0.0.0-20180808010927-aebbeb60ea05 // indirect
	github.com/anacrolix/log v0.0.0-20180808012509-286fcf906b48 // indirect
	github.com/anacrolix/missinggo v0.0.0-20181129073415-3237bf955fed // indirect
	github.com/anacrolix/mmsg v0.0.0-20180808012353-5adb2c1127c0 // indirect
	github.com/anacrolix/sync v0.0.0-20180808010631-44578de4e778 // indirect
	github.com/anacrolix/utp v0.0.0-20180219060659-9e0e1d1d0572 // indirect
	github.com/aokoli/goutils v1.0.1 // indirect
	github.com/apache/thrift v0.13.0 // indirect
	github.com/aws/aws-sdk-go v1.39.0 // indirect
	github.com/basgys/goxml2json v1.1.1-0.20181031222924-996d9fc8d313 // indirect
	github.com/beevik/etree v1.1.0 // indirect
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/blang/semver v3.5.1+incompatible // indirect
	github.com/bmizerany/assert v0.0.0-20160611221934-b7ed37b82869 // indirect
	github.com/boltdb/bolt v1.3.1 // indirect
	github.com/boombuler/barcode v1.0.1-0.20190219062509-6c824513bacc // indirect
	github.com/bradfitz/iter v0.0.0-20140124041915-454541ec3da2 // indirect
	github.com/cespare/xxhash v1.1.0 // indirect
	github.com/cespare/xxhash/v2 v2.2.0 // indirect
	github.com/checkpoint-restore/go-criu/v4 v4.1.0 // indirect
	github.com/cilium/ebpf v0.0.0-20200702112145-1c8d4c9ef775 // indirect
	github.com/cloudflare/golz4 v0.0.0-20150217214814-ef862a3cdc58 // indirect
	github.com/containerd/console v1.0.0 // indirect
	github.com/containerd/containerd v1.4.1 // indirect
	github.com/containerd/ttrpc v1.0.2 // indirect
	github.com/coreos/go-semver v0.3.0 // indirect
	github.com/coreos/go-systemd/v22 v22.3.2 // indirect
	github.com/cyphar/filepath-securejoin v0.2.2 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/dgrijalva/jwt-go v3.2.0+incompatible // indirect
	github.com/dimchansky/utfbom v1.1.0 // indirect
	github.com/dnstap/golang-dnstap v0.0.0-20170829151710-2cf77a2b5e11 // indirect
	github.com/docker/distribution v2.7.1-0.20190205005809-0d3efadf0154+incompatible // indirect
	github.com/docker/go-connections v0.4.0 // indirect
	github.com/docker/go-units v0.4.0 // indirect
	github.com/dustin/go-humanize v1.0.0 // indirect
	github.com/eapache/go-resiliency v1.1.0 // indirect
	github.com/eapache/go-xerial-snappy v0.0.0-20180814174437-776d5712da21 // indirect
	github.com/eapache/queue v1.1.0 // indirect
	github.com/edsrzf/mmap-go v1.0.0 // indirect
	github.com/elgatito/upnp v0.0.0-20180711183757-2f244d205f9a // indirect
	github.com/euank/go-kmsg-parser v2.0.0+incompatible // indirect
	github.com/farsightsec/golang-framestream v0.0.0-20181102145529-8a0cb8ba8710 // indirect
	github.com/fatih/color v1.13.0 // indirect
	github.com/felixge/httpsnoop v1.0.4 // indirect
	github.com/flynn/go-shlex v0.0.0-20150515145356-3f9db97f8568 // indirect
	github.com/frankban/quicktest v1.14.3 // indirect
	github.com/gin-contrib/sse v0.1.0 // indirect
	github.com/glycerine/go-unsnap-stream v0.0.0-20181221182339-f9677308dec2 // indirect
	github.com/go-logfmt/logfmt v0.5.0 // indirect
	github.com/go-logr/logr v1.4.2 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/go-ole/go-ole v1.2.6 // indirect
	github.com/go-playground/locales v0.13.0 // indirect
	github.com/go-playground/universal-translator v0.17.0 // indirect
	github.com/go-playground/validator/v10 v10.4.1 // indirect
	github.com/go-sql-driver/mysql v1.6.0 // indirect
	github.com/godbus/dbus/v5 v5.0.4 // indirect
	github.com/gofrs/uuid v4.1.0+incompatible // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/golang-plus/errors v1.0.0 // indirect
	github.com/golang/groupcache v0.0.0-20210331224755-41bb18bfe9da // indirect
	github.com/golang/protobuf v1.5.3 // indirect
	github.com/golang/snappy v0.0.4 // indirect
	github.com/google/btree v1.0.1 // indirect
	github.com/google/go-querystring v1.0.0 // indirect
	github.com/google/gofuzz v1.1.0 // indirect
	github.com/google/s2a-go v0.1.7 // indirect
	github.com/googleapis/enterprise-certificate-proxy v0.3.2 // indirect
	github.com/googleapis/gax-go/v2 v2.12.2 // indirect
	github.com/googleapis/gnostic v0.4.1 // indirect
	github.com/googollee/go-engine.io v0.0.0-20180829091931-e2f255711dcb // indirect
	github.com/gopherjs/gopherjs v0.0.0-20181017120253-0766667cb4d1 // indirect
	github.com/grpc-ecosystem/grpc-opentracing v0.0.0-20180507213350-8e809c8a8645 // indirect
	github.com/huandu/xstrings v1.2.0 // indirect
	github.com/huaweicloud/huaweicloud-sdk-go v1.0.26 // indirect
	github.com/imdario/mergo v0.3.6 // indirect
	github.com/jdcloud-api/jdcloud-sdk-go v1.55.0 // indirect
	github.com/jmespath/go-jmespath v0.4.0 // indirect
	github.com/josharian/native v1.1.0 // indirect
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/jtolds/gls v4.20.0+incompatible // indirect
	github.com/kardianos/osext v0.0.0-20190222173326-2bc1f35cddc0 // indirect
	github.com/karrick/godirwalk v1.16.1 // indirect
	github.com/kr/fs v0.1.0 // indirect
	github.com/leodido/go-urn v1.2.0 // indirect
	github.com/lestrrat-go/iter v0.0.0-20200422075355-fc1769541911 // indirect
	github.com/lestrrat-go/pdebug v0.0.0-20200204225717-4d6bd78da58d // indirect
	github.com/lestrrat/go-pdebug v0.0.0-20180220043741-569c97477ae8 // indirect
	github.com/lufia/plan9stats v0.0.0-20211012122336-39d0f177ccd0 // indirect
	github.com/ma314smith/signedxml v0.0.0-20210628192057-abc5b481ae1c // indirect
	github.com/mattn/go-colorable v0.1.9 // indirect
	github.com/mattn/go-isatty v0.0.14 // indirect
	github.com/mattn/go-runewidth v0.0.13 // indirect
	github.com/mattn/go-tty v0.0.0-20181127064339-e4f871175a2f // indirect
	github.com/matttproud/golang_protobuf_extensions v1.0.2-0.20181231171920-c182affec369 // indirect
	github.com/mdlayher/raw v0.0.0-20190606142536-fef19f00fc18 // indirect
	github.com/mdlayher/socket v0.4.1 // indirect
	github.com/mgutz/ansi v0.0.0-20170206155736-9520e82c474b // indirect
	github.com/mindprince/gonvml v0.0.0-20190828220739-9ebdce4bb989 // indirect
	github.com/minio/minio-go/v6 v6.0.33 // indirect
	github.com/mistifyio/go-zfs v2.1.2-0.20190413222219-f784269be439+incompatible // indirect
	github.com/mitchellh/go-homedir v1.1.0 // indirect
	github.com/moby/sys/mountinfo v0.1.3 // indirect
	github.com/moby/term v0.5.0 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.2 // indirect
	github.com/mohae/deepcopy v0.0.0-20170929034955-c48cc78d4826 // indirect
	github.com/mozillazg/go-httpheader v0.2.1 // indirect
	github.com/mozillazg/go-pinyin v0.19.0 // indirect
	github.com/mrunalp/fileutils v0.0.0-20200520151820-abd8a0e76976 // indirect
	github.com/mschoch/smat v0.0.0-20160514031455-90eadee771ae // indirect
	github.com/mxk/go-flowrate v0.0.0-20140419014527-cca7078d478f // indirect
	github.com/op/go-logging v0.0.0-20160315200505-970db520ece7 // indirect
	github.com/opencontainers/go-digest v1.0.0 // indirect
	github.com/opencontainers/image-spec v1.0.1 // indirect
	github.com/opencontainers/runc v1.0.0-rc92 // indirect
	github.com/opencontainers/runtime-spec v1.0.3-0.20200728170252-4d89ac9fbff6 // indirect
	github.com/opencontainers/selinux v1.6.0 // indirect
	github.com/opentracing-contrib/go-observer v0.0.0-20170622124052-a52f23424492 // indirect
	github.com/opentracing/opentracing-go v1.0.2 // indirect
	github.com/openzipkin/zipkin-go-opentracing v0.3.4 // indirect
	github.com/oracle/oci-go-sdk v24.3.0+incompatible // indirect
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
	github.com/richardlehane/mscfb v1.0.4 // indirect
	github.com/richardlehane/msoleps v1.0.3 // indirect
	github.com/rivo/uniseg v0.2.0 // indirect
	github.com/ryszard/goskiplist v0.0.0-20150312221310-2dfbae5fcf46 // indirect
	github.com/seccomp/libseccomp-golang v0.9.1 // indirect
	github.com/smartystreets/assertions v1.2.0 // indirect
	github.com/spaolacci/murmur3 v1.1.0 // indirect
	github.com/spf13/pflag v1.0.5 // indirect
	github.com/stretchr/objx v0.5.2 // indirect
	github.com/syndtr/gocapability v0.0.0-20180916011248-d98352740cb2 // indirect
	github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common v1.0.413 // indirect
	github.com/tencentyun/cos-go-sdk-v5 v0.7.24 // indirect
	github.com/texttheater/golang-levenshtein v0.0.0-20180516184445-d188e65d659e // indirect
	github.com/tinylib/msgp v1.1.0 // indirect
	github.com/tklauser/go-sysconf v0.3.10 // indirect
	github.com/tklauser/numcpus v0.4.0 // indirect
	github.com/ugorji/go/codec v1.1.7 // indirect
	github.com/ulikunitz/xz v0.5.12 // indirect
	github.com/vmware/govmomi v0.37.1 // indirect
	github.com/volcengine/ve-tos-golang-sdk/v2 v2.6.2 // indirect
	github.com/volcengine/volc-sdk-golang v1.0.23 // indirect
	github.com/willf/bitset v1.1.11-0.20200630133818-d5bec3311243 // indirect
	github.com/willf/bloom v2.0.3+incompatible // indirect
	github.com/xuri/efp v0.0.0-20220603152613-6918739fd470 // indirect
	github.com/xuri/nfp v0.0.0-20220409054826-5e722a1d9e22 // indirect
	github.com/yusufpapurcu/wmi v1.2.2 // indirect
	go.etcd.io/etcd/client/pkg/v3 v3.5.0 // indirect
	go.opencensus.io v0.24.0 // indirect
	go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc v0.48.0 // indirect
	go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp v0.48.0 // indirect
	go.opentelemetry.io/otel v1.23.0 // indirect
	go.opentelemetry.io/otel/metric v1.23.0 // indirect
	go.opentelemetry.io/otel/trace v1.23.0 // indirect
	go.uber.org/atomic v1.7.0 // indirect
	go.uber.org/multierr v1.6.0 // indirect
	go.uber.org/zap v1.17.0 // indirect
	golang.org/x/oauth2 v0.17.0 // indirect
	golang.org/x/term v0.17.0 // indirect
	google.golang.org/api v0.167.0 // indirect
	google.golang.org/appengine v1.6.8 // indirect
	google.golang.org/genproto v0.0.0-20240213162025-012b6fc9bca9 // indirect
	google.golang.org/genproto/googleapis/api v0.0.0-20240304161311-37d4d3c04a78 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20240228224816-df926f6c8641 // indirect
	gopkg.in/alexcesaro/quotedprintable.v3 v3.0.0-20150716171945-2caba252f4dc // indirect
	gopkg.in/asn1-ber.v1 v1.0.0-20181015200546-f715ec2f112d // indirect
	gopkg.in/inf.v0 v0.9.1 // indirect
	gopkg.in/ini.v1 v1.62.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
	gotest.tools/v3 v3.5.1 // indirect
	k8s.io/utils v0.0.0-20201110183641-67b214c5f920 // indirect
	sigs.k8s.io/structured-merge-diff/v4 v4.0.1 // indirect
	sigs.k8s.io/yaml v1.2.0 // indirect
)

replace github.com/influxdata/promql/v2 => github.com/zexi/promql/v2 v2.12.1

replace github.com/Azure/azure-sdk-for-go => github.com/Azure/azure-sdk-for-go v36.1.0+incompatible

replace github.com/docker/docker => github.com/docker/docker v20.10.27+incompatible
