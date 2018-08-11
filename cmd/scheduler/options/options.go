package options

import (
	"fmt"
	"os"

	"gopkg.in/gin-gonic/gin.v1"

	"yunion.io/x/log"
	"yunion.io/x/pkg/util/version"
	"yunion.io/x/structarg"
)

type SchedulerOptions struct {
	IgnoreNonRunningGuests      bool   `help:"Ignore non running guests when build host memory and cpu size" default:"false" alias:"ignore-nonrunning-guests"`
	IgnoreFakeDeletedGuests     bool   `help:"Ignore fake deleted guests when build host memory and cpu size" default:"false"`
	AlwaysCheckAllPredicates    bool   `help:"Excute all predicates when scheduling" default:"false"`
	DisableBaremetalPredicates  bool   `help:"Switch to trigger baremetal related predicates" default:"false"`
	SchedulerTestLimit          int    `help:"Scheduler test items' limitations" default:"100"`
	SchedulerHistoryLimit       int    `help:"Scheduler history items' limitations" default:"1000"`
	SchedulerHistoryCleanPeriod string `help:"Scheduler history cleanup period" default:"60s"`

	// per isolated device default reserverd resource
	MemoryReservedPerIsolatedDevice  int64 `help:"Per isolated device default reserverd memory size in MB" default:"8192"`    // 8G
	CpuReservedPerIsolatedDevice     int64 `help:"Per isolated device default reserverd CPU count" default:"8"`               // 8 core
	StorageReservedPerIsolatedDevice int64 `help:"Per isolated device default reserverd storage size in MB" default:"102400"` // 100G

	// parallelization options
	HostBuildParallelizeSize int `help:"Number of host description build parallelization" default:"14"`
	PredicateParallelizeSize int `help:"Number of execute predicates parallelization" default:"14"`
	PriorityParallelizeSize  int `help:"Number of execute priority parallelization" default:"14"`

	// overcommit bound options
	DefaultStorageOvercommitBound int `help:"Default storage overcommit bound" default:"1"`
	DefaultCpuOvercommitBound     int `help:"Default cpu overcommit bound" default:"8"`
	DefaultMemoryOvercommitBound  int `help:"Default memory overcommit bound" default:"1"`

	// expire queue options
	ExpireQueueConsumptionPeriod  string `help:"Expire queue consumption period" default:"3s"`
	ExpireQueueConsumptionTimeout string `help:"Expire queue consumption timeout" default:"10s"`
	ExpireQueueMaxLength          int    `help:"Expire queue max length" default:"1000"`
	ExpireQueueDealLength         int    `help:"Expire queue deal length" default:"100"`

	// completed queue options
	CompletedQueueConsumptionPeriod  string `help:"Completed queue consumption period" default:"30s"`
	CompletedQueueConsumptionTimeout string `help:"Completed queue consumption timeout" default:"30s"`
	CompletedQueueMaxLength          int    `help:"Completed queue max length" default:"100"`
	CompletedQueueDealLength         int    `help:"Completed queue deal length" default:"10"`

	// cache options
	HostCandidateCacheTTL         string `help:"Build host description candidate cache TTL" default:"0s"`
	HostCandidateCacheReloadCount int    `help:"Build host description candidate cache reload times count" default:"20"`
	HostCandidateCachePeriod      string `help:"Build host description candidate cache period" default:"30s"`

	BaremetalCandidateCacheTTL         string `help:"Build Baremetal description candidate cache TTL" default:"0s"`
	BaremetalCandidateCacheReloadCount int    `help:"Build Baremetal description candidate cache reload times count" default:"20"`
	BaremetalCandidateCachePeriod      string `help:"Build Baremetal description candidate cache period" default:"30s"`

	NetworkCacheTTL    string `help:"Build network info from database to cache TTL" default:"0s"`
	NetworkCachePeriod string `help:"Build network info from database to cache TTL" default:"1m"`

	ClusterDBCacheTTL    string `help:"Cluster database cache TTL" default:"0s"`
	ClusterDBCachePeriod string `help:"Cluster database cache period" default:"5m"`

	BaremetalAgentDBCacheTTL    string `help:"BaremetalAgent database cache TTL" default:"0s"`
	BaremetalAgentDBCachePeriod string `help:"BaremetalAgent database cache period" default:"5m"`

	AggregateDBCacheTTL    string `help:"Aggregate database cache TTL" default:"0s"`
	AggregateDBCachePeriod string `help:"Aggregate database cache period" default:"30s"`

	AggregateHostDBCacheTTL    string `help:"AggregateHost database cache TTL" default:"0s"`
	AggregateHostDBCachePeriod string `help:"AggregateHost database cache period" default:"30s"`

	NetworksDBCacheTTL    string `help:"Networks database cache TTL" default:"0s"`
	NetworksDBCachePeriod string `help:"Networks database cache period" default:"5m"`

	NetinterfaceDBCacheTTL    string `help:"Netinterfaces database cache TTL" default:"0s"`
	NetinterfaceDBCachePeriod string `help:"Netinterfaces database cache period" default:"5m"`

	WireDBCacheTTL    string `help:"Wire database cache TTL" default:"0s"`
	WireDBCachePeriod string `help:"Wire database cache period" default:"5m"`
}

type Options struct {
	// common options
	structarg.BaseOptions
	Port    int    `help:"The port that the scheduler's http service runs on" default:"8897" alias:"scheduler-port"`
	Address string `help:"The IP address to serve on (set to 0.0.0.0 for all interfaces)" default:"0.0.0.0"`

	// mysql options
	SqlConnection string `help:"SQL connection string" default:"root:root@tcp(127.0.0.1:3306)/mclouds?charset=utf8&parseTime=True"`

	// log options
	LogLevel        string `help:"log level" default:"info" choices:"debug|info|warn|error"`
	LogVerboseLevel int    `help:"log verbosity level" default:"0"`

	// gin http framework mode
	GinMode string `help:"gin http framework work mode" default:"debug" choices:"debug|release"`

	// cloud auth options
	Region      string `help:"Region name" default:"Beijing"`
	AuthURL     string `help:"Keystone auth URL" default:"http://10.168.26.241:35357/v2.0" alias:"auth-uri"`
	AdminUser   string `help:"Admin username" default:"regionadmin"`
	AdminPasswd string `help:"Admin password" default:"eBVVSNaMeyzDnD8F" alias:"admin-password"`
	AdminTenant string `help:"Admin tenant" default:"system" alias:"admin-tenant-name"`

	// scheduler options
	SchedulerOptions
}

var options Options

func GetOptions() *Options {
	return &options
}

func Parse() {
	parser, e := structarg.NewArgumentParser(&options,
		"scheduler",
		`Yunion cloud scheduler`,
		`Yunion Technology @ 2018`)
	if e != nil {
		log.Fatalf("Error define argument parser: %v", e)
	}

	e = parser.ParseArgs(os.Args[1:], false)
	if e != nil {
		log.Fatalf("Parse arguments error: %v", e)
	}

	if len(options.Config) > 0 {
		e := parser.ParseFile(options.Config)
		if e != nil {
			log.Fatalf("Parse configuration file: %v", e)
		}
	}

	if options.Help {
		fmt.Println(parser.HelpString())
		os.Exit(0)
	}

	if options.Version {
		fmt.Printf("Yunion cloud version:\n%s", version.GetJsonString())
		os.Exit(0)
	}

	// log configuration
	log.SetVerboseLevel(int32(options.LogVerboseLevel))
	e = log.SetLogLevelByString(log.Logger(), options.LogLevel)
	if e != nil {
		log.Fatalf("Set log level %q: %v", options.LogLevel, e)
	}

	log.V(10).Debugf("Parsed options: %#v", options)

	// gin http framework mode configuration
	gin.SetMode(options.GinMode)
}
