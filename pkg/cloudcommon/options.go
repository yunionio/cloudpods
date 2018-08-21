package cloudcommon

import (
	"fmt"
	"os"
	"path"

	"yunion.io/x/log"
	"yunion.io/x/pkg/util/version"
	"yunion.io/x/pkg/utils"
	"yunion.io/x/structarg"
)

type Options struct {
	Port    int    `help:"The port that the service runs on"`
	Address string `help:"The IP address to serve on (set to 0.0.0.0 for all interfaces)" default:"0.0.0.0"`

	LogLevel        string `help:"log level" default:"info" choices:"debug|info|warn|error"`
	LogVerboseLevel int    `help:"log verbosity level" default:"0"`

	Region             string   `help:"Region name or ID"`
	AuthURL            string   `help:"Keystone auth URL" alias:"auth-uri"`
	AdminUser          string   `help:"Admin username"`
	AdminDomain        string   `help:"Admin user domain"`
	AdminPassword      string   `help:"Admin password"`
	AdminProject       string   `help:"Admin project" default:"system" alias:"admin-tenant-name"`
	CorsHosts          []string `help:"List of hostname that allow CORS"`
	AuthTokenCacheSize uint32   `help:"Auth token Cache Size" default:"2048"`

	ApplicationID      string `help:"Application ID"`
	RequestWorkerCount int    `default:"4" help:"Request worker thread count, default is 4"`

	NotifyAdminUser string `default:"sysadmin" help:"System administrator user ID or name to notify"`

	GlobalVirtualResourceNamespace bool `help:"Per project namespace or global namespace for virtual resources"`
	DebugSqlchemy                  bool `default:"False" help:"Print SQL executed by sqlchemy"`

	structarg.BaseOptions
}

type DBOptions struct {
	SqlConnection string `help:"SQL connection string"`
	AutoSyncTable bool   `help:"Automatically synchronize table changes if differences are detected"`

	Options
}

func (this *DBOptions) GetDBConnection() (dialect, connstr string, err error) {
	return utils.TransSQLAchemyURL(this.SqlConnection)
}

func ParseOptions(optStruct interface{}, optionsRef *Options, args []string, configFileName string) {
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
}
