// Copyright 2019 Yunion
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package options

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"

	"golang.org/x/net/http/httpproxy"
	"golang.org/x/text/language"

	"yunion.io/x/log"
	"yunion.io/x/log/hooks"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/httputils"
	"yunion.io/x/pkg/util/reflectutils"
	"yunion.io/x/pkg/util/version"
	"yunion.io/x/pkg/utils"
	"yunion.io/x/structarg"

	"yunion.io/x/onecloud/pkg/cloudcommon/consts"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/util/atexit"
	"yunion.io/x/onecloud/pkg/util/fileutils2"
)

const (
	DefaultQuotaUnlimit = "unlimit"
	DefaultQuotaZero    = "zero"
	DefaultQuotaDefault = "default"
)

type BaseOptions struct {
	Region string `help:"Region name or ID" alias:"auth-region"`

	Port    int    `help:"The port that the service runs on" alias:"bind-port"`
	Address string `help:"The IP address to serve on (set to 0.0.0.0 for all interfaces)" default:"0.0.0.0" alias:"bind-host"`

	DebugClient bool `help:"Switch on/off mcclient debugs" default:"false"`

	LogLevel           string `help:"log level" default:"info" choices:"debug|info|warn|error"`
	LogWithTimeZone    string `help:"log time zone" default:"UTC"`
	LogTimestampFormat string `help:"log time format" default:"2006-01-02 15:04:05"`
	LogVerboseLevel    int    `help:"log verbosity level" default:"0"`
	LogFilePrefix      string `help:"prefix of log files"`

	CorsHosts []string `help:"List of hostname that allow CORS"`
	TempPath  string   `help:"Path for store temp file, at least 40G space" default:"/opt/yunion/tmp"`

	ApplicationID      string `help:"Application ID"`
	RequestWorkerCount int    `default:"8" help:"Request worker thread count, default is 8"`
	TaskWorkerCount    int    `default:"4" help:"Task manager worker thread count, default is 4"`

	RequestWorkerQueueSize int `default:"10" help:"Request worker queue size, default is 10"`

	DefaultProcessTimeoutSeconds int `default:"60" help:"request process timeout, default is 60 seconds"`

	EnableSsl   bool   `help:"Enable https"`
	SslCaCerts  string `help:"ssl certificate ca root file, separating ca and cert file is not encouraged" alias:"ca-file"`
	SslCertfile string `help:"ssl certification file, normally combines all the certificates in the chain" alias:"cert-file"`
	SslKeyfile  string `help:"ssl certification private key file" alias:"key-file"`

	NotifyAdminUsers  []string `default:"sysadmin" help:"System administrator user ID or name to notify system events, if domain is not default, specify domain as prefix ending with double backslash, e.g. domain\\\\user"`
	NotifyAdminGroups []string `help:"System administrator group ID or name to notify system events, if domain is not default, specify domain as prefix ending with double backslash, e.g. domain\\\\group"`

	// EnableRbac                       bool `help:"Switch on Role-based Access Control" default:"true"`
	RbacDebug                        bool `help:"turn on rbac debug log" default:"false"`
	RbacPolicyRefreshIntervalSeconds int  `help:"policy refresh interval in seconds, default half a minute" default:"30"`
	// RbacPolicySyncFailedRetrySeconds int  `help:"seconds to wait after a failed sync, default 30 seconds" default:"30"`
	PolicyWorkerCount int `help:"Policy worker count" default:"1"`

	ConfigSyncPeriodSeconds int `help:"service config sync interval in seconds, default 30 minutes" default:"1800"`

	IsSlaveNode        bool `help:"Slave mode"`
	CronJobWorkerCount int  `help:"Cron job worker count" default:"4"`

	EnableQuotaCheck  bool   `help:"enable quota check" default:"false"`
	DefaultQuotaValue string `help:"default quota value" choices:"unlimit|zero|default" default:"default"`

	CalculateQuotaUsageIntervalSeconds int `help:"interval to calculate quota usages, default 30 minutes" default:"900"`

	NonDefaultDomainProjects bool `help:"allow projects in non-default domains" default:"false" json:",allowfalse"`

	TimeZone string `help:"time zone" default:"Asia/Shanghai"`

	DomainizedNamespace bool `help:"turn on global name space, default is on" default:"false" json:"domainized_namespace,allowfalse"`

	ApiServer string `help:"URL to access frontend webconsole"`

	CustomizedPrivatePrefixes []string `help:"customized private prefixes"`

	structarg.BaseOptions

	GlobalHTTPProxy  string `help:"Global http proxy"`
	GlobalHTTPSProxy string `help:"Global https proxy"`

	IgnoreNonrunningGuests bool `default:"true" help:"Count memory for running guests only when do scheduling. Ignore memory allocation for non-running guests"`

	PlatformName  string            `help:"identity name of this platform" default:"Cloudpods"`
	PlatformNames map[string]string `help:"identity name of this platform by language"`

	EnableAppProfiling bool `help:"enable profiling API" default:"false"`
	AllowTLS1x         bool `help:"allow obsolete insecure TLS V1.0&1.1" default:"false" json:"allow_tls1x"`

	EnableChangeOwnerAutoRename bool `help:"Allows renaming when changing names" default:"false"`
	EnableDefaultPolicy         bool `help:"Enable defualt policies" default:"true"`
}

const (
	LockMethodInMemory = "inmemory"
	LockMethodEtcd     = "etcd"
)

type CommonOptions struct {
	AuthURL            string `help:"Keystone auth URL" alias:"auth-uri"`
	AdminUser          string `help:"Admin username"`
	AdminDomain        string `help:"Admin user domain" default:"Default"`
	AdminPassword      string `help:"Admin password" alias:"admin-passwd"`
	AdminProject       string `help:"Admin project" default:"system" alias:"admin-tenant-name"`
	AdminProjectDomain string `help:"Domain of Admin project" default:"Default"`
	AuthTokenCacheSize uint32 `help:"Auth token Cache Size" default:"2048"`

	TenantCacheExpireSeconds int `help:"expire seconds of cached tenant/domain info. defailt 15 minutes" default:"900"`

	SessionEndpointType string `help:"Client session end point type" default:"internal"`

	BaseOptions
}

type HostCommonOptions struct {
	CommonOptions

	ExecutorSocketPath     string `help:"Executor socket path" default:"/var/run/onecloud/exec.sock"`
	DeployServerSocketPath string `help:"Deploy server listen socket path" default:"/var/run/onecloud/deploy.sock"`

	EnableRemoteExecutor bool `help:"Enable remote executor" default:"false"`

	ExecutorConnectTimeoutSeconds int    `help:"executor client connection timeout in seconds, default is 30" default:"30"`
	EnableIsolatedDeviceWhitelist bool   `help:"enable isolated device white list" default:"false"`
	ImageDeployDriver             string `help:"Image deploy driver" default:"qemu-kvm" choices:"qemu-kvm|nbd|libguestfs"`
	DeployConcurrent              int    `help:"qemu-kvm deploy driver concurrent" default:"5"`
	Qcow2Preallocation            string `help:"Qcow2 image create preallocation" default:"metadata" choices:"disable|metadata|falloc|full"`
}

type DBOptions struct {
	SqlConnection string `help:"SQL connection string" alias:"connection"`

	Clickhouse string `help:"Connection string for click house"`

	DbMaxWaitTimeoutSeconds int `help:"max wait timeout for db connection, default 1 hour" default:"3600"`

	OpsLogWithClickhouse   bool `help:"store operation logs with clickhouse" default:"false"`
	EnableDBChecksumTables bool `help:"Enable DB tables with record checksum for consistency"`
	DBChecksumSkipInit     bool `help:"Skip DB tables with record checksum calculation when init" default:"false"`

	DBChecksumHashAlgorithm string `help:"hash algorithm for db checksum hash" choices:"md5|sha256" default:"sha256"`

	AutoSyncTable   bool `help:"Automatically synchronize table changes if differences are detected"`
	ExitAfterDBInit bool `help:"Exit program after db initialization" default:"false"`

	GlobalVirtualResourceNamespace bool `help:"Per project namespace or global namespace for virtual resources" default:"false"`
	DebugSqlchemy                  bool `default:"false" help:"Print SQL executed by sqlchemy"`

	QueryOffsetOptimization bool `help:"apply query offset optimization"`

	HistoricalUniqueName bool `help:"use historically unique name" default:"false"`

	LockmanMethod string `help:"method for lock synchronization" choices:"inmemory|etcd" default:"inmemory"`

	OpsLogMaxKeepMonths int `help:"maximal months of logs to keep, default 6 months" default:"6"`

	SplitableMaxDurationHours int `help:"maximal number of hours that a splitable segement lasts, default 30 days" default:"720"`

	EtcdOptions

	EtcdLockPrefix string `help:"prefix of etcd lock records" default:"/onecloud/lockman"`
	EtcdLockTTL    int    `help:"ttl of etcd lock records" default:"5"`
}

type EtcdOptions struct {
	EtcdEndpoints     []string `help:"endpoints of etcd cluster"`
	EtcdUsername      string   `help:"username of etcd cluster"`
	EtcdPassword      string   `help:"password of etcd cluster"`
	EtcdUseTLS        bool     `help:"use tls transport to connect etcd cluster" default:"false"`
	EtcdSkipTLSVerify bool     `help:"skip tls verification" default:"false"`
	EtcdCacert        string   `help:"path to cacert for connecting to etcd cluster"`
	EtcdCert          string   `help:"path to cert file for connecting to etcd cluster"`
	EtcdKey           string   `help:"path to key file for connecting to etcd cluster"`
}

func (opt *EtcdOptions) GetEtcdTLSConfig() (*tls.Config, error) {
	var (
		cert       tls.Certificate
		certLoaded bool
		capool     *x509.CertPool
	)
	if opt.EtcdCert != "" && opt.EtcdKey != "" {
		var err error
		cert, err = tls.LoadX509KeyPair(opt.EtcdCert, opt.EtcdKey)
		if err != nil {
			return nil, errors.Wrap(err, "load etcd cert and key")
		}
		certLoaded = true
		opt.EtcdUseTLS = true
	}
	if opt.EtcdCacert != "" {
		data, err := os.ReadFile(opt.EtcdCacert)
		if err != nil {
			return nil, errors.Wrap(err, "read cacert file")
		}
		capool = x509.NewCertPool()
		for {
			var block *pem.Block
			block, data = pem.Decode(data)
			if block == nil {
				break
			}
			cacert, err := x509.ParseCertificate(block.Bytes)
			if err != nil {
				return nil, errors.Wrap(err, "parse cacert file")
			}
			capool.AddCert(cacert)
		}
		opt.EtcdUseTLS = true
	}
	if opt.EtcdSkipTLSVerify { // it's false by default, true means user intends to use tls
		opt.EtcdUseTLS = true
	}
	if opt.EtcdUseTLS {
		cfg := &tls.Config{
			RootCAs:            capool,
			InsecureSkipVerify: opt.EtcdSkipTLSVerify,
		}
		if certLoaded {
			cfg.Certificates = []tls.Certificate{cert}
		}
		return cfg, nil
	}
	return nil, nil
}

func (opt *DBOptions) GetDBConnection() (string, string, error) {
	if strings.HasPrefix(opt.SqlConnection, "mysql") {
		return utils.TransSQLAchemyURL(opt.SqlConnection)
	} else {
		pos := strings.Index(opt.SqlConnection, "://")
		if pos > 0 {
			return opt.SqlConnection[:pos], opt.SqlConnection[pos+3:], nil
		} else {
			return "", "", httperrors.ErrNotSupported
		}
	}
}

func (opt *DBOptions) GetClickhouseConnStr() (string, string, error) {
	if len(opt.Clickhouse) == 0 {
		return "", "", errors.ErrNotFound
	}
	return "clickhouse", opt.Clickhouse, nil
}

func ParseOptionsIgnoreNoConfigfile(optStruct interface{}, args []string, configFileName string, serviceType string) {
	parseOptions(optStruct, args, configFileName, serviceType, true)
}

func ParseOptions(optStruct interface{}, args []string, configFileName string, serviceType string) {
	parseOptions(optStruct, args, configFileName, serviceType, false)
}

func parseOptions(optStruct interface{}, args []string, configFileName string, serviceType string, ignoreNoConfigfile bool) {
	if len(serviceType) == 0 {
		log.Fatalf("ServiceType must provided!")
	}

	consts.SetServiceType(serviceType)

	serviceName := path.Base(args[0])

	parser, err := structarg.NewArgumentParser(optStruct,
		serviceName,
		fmt.Sprintf(`Yunion cloud service - %s`, serviceName),
		`Yunion Technology Co. Ltd. @ 2018-2019`)
	if err != nil {
		log.Fatalf("Error define argument parser: %v", err)
	}

	err = parser.ParseArgs2(args[1:], false, false)
	if err != nil {
		log.Fatalf("Parse arguments error: %v", err)
	}

	var optionsRef *BaseOptions

	err = reflectutils.FindAnonymouStructPointer(optStruct, &optionsRef)
	if err != nil {
		log.Fatalf("Find common options fail: %s", err)
	}

	if optionsRef.Help {
		fmt.Println(parser.HelpString())
		os.Exit(0)
	}

	if optionsRef.Version {
		fmt.Printf("Yunion cloud version:\n%s", version.GetJsonString())
		os.Exit(0)
	}

	if len(optionsRef.Config) == 0 {
		for _, p := range []string{"./etc", "/etc/yunion"} {
			confTmp := path.Join(p, configFileName)
			if _, err := os.Stat(confTmp); err == nil {
				optionsRef.Config = confTmp
				break
			}
		}
	}

	if len(optionsRef.Config) > 0 {
		if !fileutils2.Exists(optionsRef.Config) && !ignoreNoConfigfile {
			log.Fatalf("Configuration file %s not exist", optionsRef.Config)
		} else if fileutils2.Exists(optionsRef.Config) {
			log.Infof("Use configuration file: %s", optionsRef.Config)
			err = parser.ParseFile(optionsRef.Config)
			if err != nil {
				log.Fatalf("Parse configuration file: %v", err)
			}
		}
	}

	parser.SetDefault()

	if len(optionsRef.ApplicationID) == 0 {
		optionsRef.ApplicationID = serviceName
	}

	consts.SetServiceName(optionsRef.ApplicationID)
	httperrors.SetTimeZone(optionsRef.TimeZone)

	// log configuration
	log.SetVerboseLevel(int32(optionsRef.LogVerboseLevel))
	err = log.SetLogLevelByString(log.Logger(), optionsRef.LogLevel)
	if err != nil {
		log.Fatalf("Set log level %q: %v", optionsRef.LogLevel, err)
	}
	log.Infof("Set log level to %q", optionsRef.LogLevel)
	log.Logger().Formatter = &log.TextFormatter{
		TimeZone:        optionsRef.LogWithTimeZone,
		TimestampFormat: optionsRef.LogTimestampFormat,
	}
	if optionsRef.LogFilePrefix != "" {
		dir, name := filepath.Split(optionsRef.LogFilePrefix)
		h := &hooks.LogFileRotateHook{
			RotateNum:  10,
			RotateSize: 100 * 1024 * 1024,
			LogFileHook: hooks.LogFileHook{
				FileDir:  dir,
				FileName: name,
			},
		}
		h.Init()
		log.DisableColors()
		log.Logger().AddHook(h)
		log.Logger().Out = io.Discard
		atexit.Register(atexit.ExitHandler{
			Prio:   atexit.PRIO_LOG_CLOSE,
			Reason: "deinit log rotate hook",
			Func: func(atexit.ExitHandler) {
				h.DeInit()
			},
		})
	}

	log.V(10).Debugf("Parsed options: %#v", optStruct)

	if len(optionsRef.Region) > 0 {
		consts.SetRegion(optionsRef.Region)
	}

	consts.SetDefaultPolicy(optionsRef.EnableDefaultPolicy)
	consts.SetDomainizedNamespace(optionsRef.DomainizedNamespace)
}

func (self *BaseOptions) HttpTransportProxyFunc() httputils.TransportProxyFunc {
	cfg := &httpproxy.Config{
		HTTPProxy:  self.GlobalHTTPProxy,
		HTTPSProxy: self.GlobalHTTPSProxy,
	}
	proxyFunc := cfg.ProxyFunc()
	return func(req *http.Request) (*url.URL, error) {
		return proxyFunc(req.URL)
	}
}

func (opt *BaseOptions) GetPlatformName(lang language.Tag) string {
	if len(opt.PlatformNames) > 0 {
		if name, ok := opt.PlatformNames[lang.String()]; ok {
			return name
		}
	}
	return opt.PlatformName
}
