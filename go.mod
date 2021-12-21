module yunion.io/x/onecloud

go 1.13

require (
	bazil.org/fuse v0.0.0-20180421153158-65cc252bf669
	cloud.google.com/go/storage v1.5.0
	github.com/360EntSecGroup-Skylar/excelize v1.4.0
	github.com/Azure/azure-sdk-for-go v36.1.0+incompatible
	github.com/Azure/go-autorest/autorest v0.9.6
	github.com/Azure/go-autorest/autorest/azure/auth v0.4.2
	github.com/Azure/go-autorest/autorest/to v0.4.0 // indirect
	github.com/DataDog/dd-trace-go v0.6.1 // indirect
	github.com/DataDog/zstd v1.3.4 // indirect
	github.com/Masterminds/goutils v1.1.0 // indirect
	github.com/Masterminds/semver v1.5.0 // indirect
	github.com/Masterminds/sprig v2.22.0+incompatible
	github.com/Microsoft/azure-vhd-utils v0.0.0-20181115010904-44cbada2ece3
	github.com/RoaringBitmap/roaring v0.4.16 // indirect
	github.com/Shopify/sarama v1.20.0 // indirect
	github.com/Shopify/toxiproxy v2.1.4+incompatible // indirect
	github.com/StackExchange/wmi v0.0.0-20180116203802-5d049714c4a6 // indirect
	github.com/aliyun/alibaba-cloud-sdk-go v1.61.684
	github.com/aliyun/aliyun-oss-go-sdk v2.0.4+incompatible
	github.com/anacrolix/dht v0.0.0-20181129074040-b09db78595aa // indirect
	github.com/anacrolix/go-libutp v0.0.0-20180808010927-aebbeb60ea05 // indirect
	github.com/anacrolix/log v0.0.0-20180808012509-286fcf906b48 // indirect
	github.com/anacrolix/mmsg v0.0.0-20180808012353-5adb2c1127c0 // indirect
	github.com/anacrolix/torrent v0.0.0-20181129073333-cc531b8c4a80
	github.com/aokoli/goutils v1.0.1
	github.com/apache/thrift v0.12.0 // indirect
	github.com/aws/aws-sdk-go v1.39.0
	github.com/baiyubin/aliyun-sts-go-sdk v0.0.0-20180326062324-cfa1a18b161f // indirect
	github.com/beevik/etree v1.1.0 // indirect
	github.com/benbjohnson/clock v1.0.0
	github.com/bitly/go-simplejson v0.5.0
	github.com/bmizerany/assert v0.0.0-20160611221934-b7ed37b82869 // indirect
	github.com/c-bata/go-prompt v0.2.1
	github.com/coredns/coredns v1.3.0
	github.com/coreos/go-systemd v0.0.0-20190620071333-e64a0ec8b42a // indirect
	github.com/coreos/pkg v0.0.0-20180928190104-399ea9e2e55f // indirect
	github.com/creack/pty v1.1.11
	github.com/dnaeon/go-vcr v1.1.0 // indirect
	github.com/dnstap/golang-dnstap v0.0.0-20170829151710-2cf77a2b5e11 // indirect
	github.com/eapache/go-resiliency v1.1.0 // indirect
	github.com/eapache/go-xerial-snappy v0.0.0-20180814174437-776d5712da21 // indirect
	github.com/eapache/queue v1.1.0 // indirect
	github.com/edsrzf/mmap-go v1.0.0 // indirect
	github.com/farsightsec/golang-framestream v0.0.0-20181102145529-8a0cb8ba8710 // indirect
	github.com/fatih/color v1.10.0
	github.com/fernet/fernet-go v0.0.0-20180830025343-9eac43b88a5e
	github.com/flynn/go-shlex v0.0.0-20150515145356-3f9db97f8568 // indirect
	github.com/fsnotify/fsnotify v1.4.9
	github.com/ghodss/yaml v1.0.0
	github.com/gin-gonic/gin v1.7.0
	github.com/glycerine/go-unsnap-stream v0.0.0-20181221182339-f9677308dec2 // indirect
	github.com/go-logfmt/logfmt v0.4.0 // indirect
	github.com/go-ole/go-ole v1.2.2 // indirect
	github.com/go-sql-driver/mysql v1.5.0
	github.com/go-yaml/yaml v2.1.0+incompatible
	github.com/gofrs/uuid v4.1.0+incompatible // indirect
	github.com/golang-plus/errors v1.0.0
	github.com/golang-plus/testing v1.0.0 // indirect
	github.com/golang-plus/uuid v1.0.0
	github.com/golang/mock v1.3.1
	github.com/golang/protobuf v1.4.2
	github.com/google/gopacket v1.1.17
	github.com/googollee/go-engine.io v0.0.0-20180829091931-e2f255711dcb // indirect
	github.com/googollee/go-socket.io v0.0.0-20181214084611-0ad7206c347a
	github.com/gorilla/mux v1.7.0
	github.com/gorilla/websocket v1.4.1
	github.com/gosuri/uitable v0.0.0-20160404203958-36ee7e946282
	github.com/grpc-ecosystem/grpc-opentracing v0.0.0-20180507213350-8e809c8a8645 // indirect
	github.com/hako/durafmt v0.0.0-20180520121703-7b7ae1e72ead
	github.com/huandu/xstrings v1.2.0 // indirect
	github.com/imdario/mergo v0.3.6 // indirect
	github.com/jdcloud-api/jdcloud-sdk-go v1.55.0
	github.com/kardianos/osext v0.0.0-20190222173326-2bc1f35cddc0 // indirect
	github.com/koding/websocketproxy v0.0.0-20181220232114-7ed82d81a28c
	github.com/lestrrat-go/jwx v1.0.2
	github.com/lestrrat/go-jwx v0.0.0-20180221005942-b7d4802280ae
	github.com/lestrrat/go-pdebug v0.0.0-20180220043741-569c97477ae8 // indirect
	github.com/libvirt/libvirt-go-xml v5.2.0+incompatible
	github.com/ma314smith/signedxml v0.0.0-20200410192636-c342a2d0ae60
	github.com/mattn/go-runewidth v0.0.12 // indirect
	github.com/mattn/go-sqlite3 v1.10.0 // indirect
	github.com/mattn/go-tty v0.0.0-20181127064339-e4f871175a2f // indirect
	github.com/mdlayher/arp v0.0.0-20190313224443-98a83c8a2717
	github.com/mdlayher/ethernet v0.0.0-20190606142754-0394541c37b7
	github.com/mdlayher/raw v0.0.0-20190606144222-a54781e5f38f
	github.com/mholt/caddy v0.10.11
	github.com/miekg/dns v1.1.25
	github.com/minio/minio-go v6.0.14+incompatible
	github.com/mitchellh/copystructure v1.0.0 // indirect
	github.com/mohae/deepcopy v0.0.0-20170929034955-c48cc78d4826 // indirect
	github.com/moul/http2curl v1.0.0
	github.com/mozillazg/go-pinyin v0.15.0
	github.com/opentracing-contrib/go-observer v0.0.0-20170622124052-a52f23424492 // indirect
	github.com/opentracing/opentracing-go v1.0.2 // indirect
	github.com/openzipkin/zipkin-go-opentracing v0.3.4 // indirect
	github.com/pierrec/lz4 v2.0.5+incompatible // indirect
	github.com/pierrec/lz4/v4 v4.1.12
	github.com/pkg/errors v0.9.1
	github.com/pkg/term v0.0.0-20181116001808-27bbf2edb814 // indirect
	github.com/pquerna/otp v1.2.0
	github.com/rcrowley/go-metrics v0.0.0-20181016184325-3113b8401b8a // indirect
	github.com/rivo/uniseg v0.2.0 // indirect
	github.com/satori/go.uuid v1.2.0 // indirect
	github.com/serialx/hashring v0.0.0-20180504054112-49a4782e9908
	github.com/sevlyar/go-daemon v0.1.5
	github.com/shirou/gopsutil v2.18.10+incompatible
	github.com/shirou/w32 v0.0.0-20160930032740-bb4de0191aa4 // indirect
	github.com/sirupsen/logrus v1.4.2
	github.com/skip2/go-qrcode v0.0.0-20190110000554-dc11ecdae0a9
	github.com/smartystreets/goconvey v1.6.4
	github.com/spaolacci/murmur3 v1.1.0 // indirect
	github.com/stretchr/testify v1.5.1
	github.com/tatsushid/go-fastping v0.0.0-20160109021039-d7bb493dee3e
	github.com/tencentcloud/tencentcloud-sdk-go v3.0.135+incompatible
	github.com/tencentyun/cos-go-sdk-v5 v0.7.24
	github.com/tinylib/msgp v1.1.0 // indirect
	github.com/tmc/grpc-websocket-proxy v0.0.0-20190109142713-0ad062ec5ee5 // indirect
	github.com/tredoe/osutil v0.0.0-20161130133508-7d3ee1afa71c
	github.com/vishvananda/netlink v1.0.0
	github.com/vishvananda/netns v0.0.0-20191106174202-0a2b9b5464df // indirect
	github.com/vmihailenco/msgpack v4.0.4+incompatible
	github.com/vmware/govmomi v0.20.1
	go.etcd.io/etcd v0.5.0-alpha.5.0.20200819165624-17cef6e3e9d5
	go.uber.org/atomic v1.4.0 // indirect
	golang.org/x/crypto v0.0.0-20200622213623-75b288015ac9
	golang.org/x/net v0.0.0-20201110031124-69a78807bb2b
	golang.org/x/oauth2 v0.0.0-20191202225959-858c2ad4c8b6
	golang.org/x/sync v0.0.0-20190911185100-cd5d95a43a6e
	golang.org/x/sys v0.0.0-20210403161142-5e06dd20ab57
	golang.org/x/text v0.3.3
	golang.org/x/tools v0.0.0-20200515220128-d3bf790afa53 // indirect
	golang.org/x/xerrors v0.0.0-20191204190536-9bdfabe68543
	golang.zx2c4.com/wireguard/wgctrl v0.0.0-20191008142428-8d021180e987
	google.golang.org/genproto v0.0.0-20200526211855-cb27e3aa2013
	google.golang.org/grpc v1.29.0
	google.golang.org/protobuf v1.24.0
	gopkg.in/asn1-ber.v1 v1.0.0-20181015200546-f715ec2f112d // indirect
	gopkg.in/fatih/set.v0 v0.2.1
	gopkg.in/ini.v1 v1.44.0 // indirect
	gopkg.in/ldap.v3 v3.0.3
	gopkg.in/yaml.v2 v2.2.8
	honnef.co/go/tools v0.0.1-2020.1.4 // indirect
	k8s.io/api v0.19.3
	k8s.io/apimachinery v0.19.3
	k8s.io/client-go v0.19.3
	k8s.io/cluster-bootstrap v0.19.3
	yunion.io/x/executor v0.0.0-20211018100936-39a2cd966656
	yunion.io/x/jsonutils v0.0.0-20211105163012-d846c05a3c9a
	yunion.io/x/log v0.0.0-20201210064738-43181789dc74
	yunion.io/x/ovsdb v0.0.0-20200526071744-27bf0940cbc7
	yunion.io/x/pkg v0.0.0-20211116020154-6a76ba2f7e97
	yunion.io/x/s3cli v0.0.0-20190917004522-13ac36d8687e
	yunion.io/x/sqlchemy v1.0.0
	yunion.io/x/structarg v0.0.0-20200720093445-9f850fa222ce
)
