package cloudcommon

import (
	"fmt"
	"os"
	"path"

	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/cloudcommon/consts"
	"yunion.io/x/pkg/util/version"
	"yunion.io/x/pkg/utils"
	"yunion.io/x/structarg"
)

type CommonOptions struct {
	Port    int    `help:"The port that the service runs on" alias:"bind-port"`
	Address string `help:"The IP address to serve on (set to 0.0.0.0 for all interfaces)" default:"0.0.0.0" alias:"bind-host"`

	LogLevel        string `help:"log level" default:"info" choices:"debug|info|warn|error"`
	LogVerboseLevel int    `help:"log verbosity level" default:"0"`

	Region             string   `help:"Region name or ID" alias:"auth-region"`
	AuthURL            string   `help:"Keystone auth URL" alias:"auth-uri"`
	AdminUser          string   `help:"Admin username"`
	AdminDomain        string   `help:"Admin user domain"`
	AdminPassword      string   `help:"Admin password" alias:"admin-passwd"`
	AdminProject       string   `help:"Admin project" default:"system" alias:"admin-tenant-name"`
	CorsHosts          []string `help:"List of hostname that allow CORS"`
	AuthTokenCacheSize uint32   `help:"Auth token Cache Size" default:"2048"`
	TempPath           string   `help:"Path for store temp file, at least 40G space" default:"/opt/yunion/tmp"`

	DebugClient bool `help:"Switch on/off mcclient debugs" default:"false"`

	ApplicationID      string `help:"Application ID"`
	RequestWorkerCount int    `default:"4" help:"Request worker thread count, default is 4"`

	NotifyAdminUser string `default:"sysadmin" help:"System administrator user ID or name to notify"`

	EnableSsl   bool   `help:"Enable https"`
	SslCertfile string `help:"ssl certification file"`
	SslKeyfile  string `help:"ssl certification key file"`

	EnableRbac                       bool `help:"Switch on Role-based Access Control" default:"true"`
	RbacDebug                        bool `help:"turn on rbac debug log" default:"false"`
	RbacPolicySyncPeriodSeconds      int  `help:"policy sync interval in seconds, default 15 minutes" default:"900"`
	RbacPolicySyncFailedRetrySeconds int  `help:"seconds to wait after a failed sync, default 30 seconds" default:"30"`

	structarg.BaseOptions
}

type DBOptions struct {
	SqlConnection string `help:"SQL connection string"`
	AutoSyncTable bool   `help:"Automatically synchronize table changes if differences are detected"`

	GlobalVirtualResourceNamespace bool `help:"Per project namespace or global namespace for virtual resources"`
	DebugSqlchemy                  bool `default:"False" help:"Print SQL executed by sqlchemy"`
}

func (this *DBOptions) GetDBConnection() (dialect, connstr string, err error) {
	return utils.TransSQLAchemyURL(this.SqlConnection)
}

func ParseOptions(optStruct interface{}, optionsRef *CommonOptions, args []string, configFileName string) {
	if len(consts.GetServiceType()) == 0 {
		log.Fatalf("ServiceType not initialized!")
	}

	serviceName := path.Base(args[0])
	parser, err := structarg.NewArgumentParser(optStruct,
		serviceName,
		fmt.Sprintf(`Yunion cloud service - %s`, serviceName),
		`Yunion Technology @ 2018`)
	if err != nil {
		log.Fatalf("Error define argument parser: %v", err)
	}

	err = parser.ParseArgs(args[1:], false)
	if err != nil {
		log.Fatalf("Parse arguments error: %v", err)
	}

	if len(optionsRef.Config) == 0 {
		for _, p := range []string{"./etc", "/etc/yunion"} {
			confTmp := path.Join(p, configFileName)
			log.Infof(confTmp)
			if _, err := os.Stat(confTmp); err == nil {
				optionsRef.Config = confTmp
				break
			}
		}
	}

	if len(optionsRef.Config) > 0 {
		log.Infof("Use configuration file: %s", optionsRef.Config)
		err = parser.ParseFile(optionsRef.Config)
		if err != nil {
			log.Fatalf("Parse configuration file: %v", err)
		}
	}

	if optionsRef.Help {
		fmt.Println(parser.HelpString())
		os.Exit(0)
	}

	if optionsRef.Version {
		fmt.Printf("Yunion cloud version:\n%s", version.GetJsonString())
		os.Exit(0)
	}

	if len(optionsRef.ApplicationID) == 0 {
		optionsRef.ApplicationID = serviceName
	}

	// log configuration
	log.SetVerboseLevel(int32(optionsRef.LogVerboseLevel))
	err = log.SetLogLevelByString(log.Logger(), optionsRef.LogLevel)
	if err != nil {
		log.Fatalf("Set log level %q: %v", optionsRef.LogLevel, err)
	}

	log.V(10).Debugf("Parsed options: %#v", optStruct)

	consts.SetRegion(optionsRef.Region)
}
