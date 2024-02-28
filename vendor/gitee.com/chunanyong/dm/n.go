/*
 * Copyright (c) 2000-2018, 达梦数据库有限公司.
 * All rights reserved.
 */
package dm

import (
	"bytes"
	"context"
	"database/sql/driver"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"gitee.com/chunanyong/dm/util"
)

const (
	TimeZoneKey              = "timeZone"
	EnRsCacheKey             = "enRsCache"
	RsCacheSizeKey           = "rsCacheSize"
	RsRefreshFreqKey         = "rsRefreshFreq"
	LoginPrimary             = "loginPrimary"
	LoginModeKey             = "loginMode"
	LoginStatusKey           = "loginStatus"
	LoginDscCtrlKey          = "loginDscCtrl"
	SwitchTimesKey           = "switchTimes"
	SwitchIntervalKey        = "switchInterval"
	EpSelectorKey            = "epSelector"
	PrimaryKey               = "primaryKey"
	KeywordsKey              = "keywords"
	CompressKey              = "compress"
	CompressIdKey            = "compressId"
	LoginEncryptKey          = "loginEncrypt"
	CommunicationEncryptKey  = "communicationEncrypt"
	DirectKey                = "direct"
	Dec2DoubleKey            = "dec2double"
	RwSeparateKey            = "rwSeparate"
	RwPercentKey             = "rwPercent"
	RwAutoDistributeKey      = "rwAutoDistribute"
	CompatibleModeKey        = "compatibleMode"
	CompatibleOraKey         = "comOra"
	CipherPathKey            = "cipherPath"
	DoSwitchKey              = "doSwitch"
	DriverReconnectKey       = "driverReconnect"
	ClusterKey               = "cluster"
	LanguageKey              = "language"
	DbAliveCheckFreqKey      = "dbAliveCheckFreq"
	RwStandbyRecoverTimeKey  = "rwStandbyRecoverTime"
	LogLevelKey              = "logLevel"
	LogDirKey                = "logDir"
	LogBufferPoolSizeKey     = "logBufferPoolSize"
	LogBufferSizeKey         = "logBufferSize"
	LogFlusherQueueSizeKey   = "logFlusherQueueSize"
	LogFlushFreqKey          = "logFlushFreq"
	StatEnableKey            = "statEnable"
	StatDirKey               = "statDir"
	StatFlushFreqKey         = "statFlushFreq"
	StatHighFreqSqlCountKey  = "statHighFreqSqlCount"
	StatSlowSqlCountKey      = "statSlowSqlCount"
	StatSqlMaxCountKey       = "statSqlMaxCount"
	StatSqlRemoveModeKey     = "statSqlRemoveMode"
	AddressRemapKey          = "addressRemap"
	UserRemapKey             = "userRemap"
	ConnectTimeoutKey        = "connectTimeout"
	LoginCertificateKey      = "loginCertificate"
	UrlKey                   = "url"
	HostKey                  = "host"
	PortKey                  = "port"
	UserKey                  = "user"
	PasswordKey              = "password"
	DialNameKey              = "dialName"
	RwStandbyKey             = "rwStandby"
	IsCompressKey            = "isCompress"
	RwHAKey                  = "rwHA"
	RwIgnoreSqlKey           = "rwIgnoreSql"
	AppNameKey               = "appName"
	OsNameKey                = "osName"
	MppLocalKey              = "mppLocal"
	SocketTimeoutKey         = "socketTimeout"
	SessionTimeoutKey        = "sessionTimeout"
	ContinueBatchOnErrorKey  = "continueBatchOnError"
	BatchAllowMaxErrorsKey   = "batchAllowMaxErrors"
	EscapeProcessKey         = "escapeProcess"
	AutoCommitKey            = "autoCommit"
	MaxRowsKey               = "maxRows"
	RowPrefetchKey           = "rowPrefetch"
	BufPrefetchKey           = "bufPrefetch"
	LobModeKey               = "LobMode"
	StmtPoolSizeKey          = "StmtPoolSize"
	IgnoreCaseKey            = "ignoreCase"
	AlwayseAllowCommitKey    = "AlwayseAllowCommit"
	BatchTypeKey             = "batchType"
	BatchNotOnCallKey        = "batchNotOnCall"
	IsBdtaRSKey              = "isBdtaRS"
	ClobAsStringKey          = "clobAsString"
	SslCertPathKey           = "sslCertPath"
	SslKeyPathKey            = "sslKeyPath"
	SslFilesPathKey          = "sslFilesPath"
	KerberosLoginConfPathKey = "kerberosLoginConfPath"
	UKeyNameKey              = "uKeyName"
	UKeyPinKey               = "uKeyPin"
	ColumnNameUpperCaseKey   = "columnNameUpperCase"
	ColumnNameCaseKey        = "columnNameCase"
	DatabaseProductNameKey   = "databaseProductName"
	OsAuthTypeKey            = "osAuthType"
	SchemaKey                = "schema"

	DO_SWITCH_OFF             int32 = 0
	DO_SWITCH_WHEN_CONN_ERROR int32 = 1
	DO_SWITCH_WHEN_EP_RECOVER int32 = 2

	CLUSTER_TYPE_NORMAL int32 = 0
	CLUSTER_TYPE_RW     int32 = 1
	CLUSTER_TYPE_DW     int32 = 2
	CLUSTER_TYPE_DSC    int32 = 3
	CLUSTER_TYPE_MPP    int32 = 4

	EP_STATUS_OK    int32 = 1
	EP_STATUS_ERROR int32 = 2

	LOGIN_MODE_PRIMARY_FIRST int32 = 0

	LOGIN_MODE_PRIMARY_ONLY int32 = 1

	LOGIN_MODE_STANDBY_ONLY int32 = 2

	LOGIN_MODE_STANDBY_FIRST int32 = 3

	LOGIN_MODE_NORMAL_FIRST int32 = 4

	SERVER_MODE_NORMAL int32 = 0

	SERVER_MODE_PRIMARY int32 = 1

	SERVER_MODE_STANDBY int32 = 2

	SERVER_STATUS_MOUNT int32 = 3

	SERVER_STATUS_OPEN int32 = 4

	SERVER_STATUS_SUSPEND int32 = 5

	COMPATIBLE_MODE_ORACLE int = 1

	COMPATIBLE_MODE_MYSQL int = 2

	LANGUAGE_CN int = 0

	LANGUAGE_EN int = 1

	LANGUAGE_CNT_HK = 2

	COLUMN_NAME_NATURAL_CASE = 0

	COLUMN_NAME_UPPER_CASE = 1

	COLUMN_NAME_LOWER_CASE = 2

	compressDef   = Dm_build_91
	compressIDDef = Dm_build_92

	charCodeDef = ""

	enRsCacheDef = false

	rsCacheSizeDef = 20

	rsRefreshFreqDef = 10

	loginModeDef = LOGIN_MODE_NORMAL_FIRST

	loginStatusDef = 0

	loginEncryptDef = true

	loginCertificateDef = ""

	dec2DoubleDef = false

	rwHADef = false

	rwStandbyDef = false

	rwSeparateDef = false

	rwPercentDef = 25

	rwAutoDistributeDef = true

	rwStandbyRecoverTimeDef = 1000

	cipherPathDef = ""

	urlDef = ""

	userDef = "SYSDBA"

	passwordDef = "SYSDBA"

	hostDef = "localhost"

	portDef = DEFAULT_PORT

	appNameDef = ""

	mppLocalDef = false

	socketTimeoutDef = 0

	connectTimeoutDef = 5000

	sessionTimeoutDef = 0

	osAuthTypeDef = Dm_build_74

	continueBatchOnErrorDef = false

	escapeProcessDef = false

	autoCommitDef = true

	maxRowsDef = 0

	rowPrefetchDef = Dm_build_75

	bufPrefetchDef = 0

	lobModeDef = 1

	stmtPoolMaxSizeDef = 15

	ignoreCaseDef = true

	alwayseAllowCommitDef = true

	isBdtaRSDef = false

	kerberosLoginConfPathDef = ""

	uKeyNameDef = ""

	uKeyPinDef = ""

	databaseProductNameDef = ""

	caseSensitiveDef = true

	compatibleModeDef = 0
)

type DmConnector struct {
	filterable

	mu sync.Mutex

	dmDriver *DmDriver

	compress int

	compressID int8

	newClientType bool

	charCode string

	enRsCache bool

	rsCacheSize int

	rsRefreshFreq int

	loginMode int32

	loginStatus int

	loginDscCtrl bool

	switchTimes int32

	switchInterval int

	epSelector int32

	keyWords []string

	loginEncrypt bool

	loginCertificate string

	dec2Double bool

	rwHA bool

	rwStandby bool

	rwSeparate bool

	rwPercent int32

	rwAutoDistribute bool

	rwStandbyRecoverTime int

	rwIgnoreSql bool

	doSwitch int32

	driverReconnect bool

	cluster int32

	cipherPath string

	url string

	user string

	password string

	dialName string

	host string

	group *epGroup

	port int32

	appName string

	osName string

	mppLocal bool

	socketTimeout int

	connectTimeout int

	sessionTimeout int

	osAuthType byte

	continueBatchOnError bool

	batchAllowMaxErrors int32

	escapeProcess bool

	autoCommit bool

	maxRows int

	rowPrefetch int

	bufPrefetch int

	lobMode int

	stmtPoolMaxSize int

	ignoreCase bool

	alwayseAllowCommit bool

	batchType int

	batchNotOnCall bool

	isBdtaRS bool

	sslCertPath string

	sslKeyPath string

	sslFilesPath string

	kerberosLoginConfPath string

	uKeyName string

	uKeyPin string

	svcConfPath string

	columnNameCase int

	caseSensitive bool

	compatibleMode int

	localTimezone int16

	schema string

	logLevel int

	logDir string

	logFlushFreq int

	logFlushQueueSize int

	logBufferSize int

	statEnable bool

	statDir string

	statFlushFreq int

	statSlowSqlCount int

	statHighFreqSqlCount int

	statSqlMaxCount int

	statSqlRemoveMode int
}

func (c *DmConnector) init() *DmConnector {
	c.compress = compressDef
	c.compressID = compressIDDef
	c.charCode = charCodeDef
	c.enRsCache = enRsCacheDef
	c.rsCacheSize = rsCacheSizeDef
	c.rsRefreshFreq = rsRefreshFreqDef
	c.loginMode = loginModeDef
	c.loginStatus = loginStatusDef
	c.loginDscCtrl = false
	c.switchTimes = 1
	c.switchInterval = 1000
	c.epSelector = 0
	c.keyWords = nil
	c.loginEncrypt = loginEncryptDef
	c.loginCertificate = loginCertificateDef
	c.dec2Double = dec2DoubleDef
	c.rwHA = rwHADef
	c.rwStandby = rwStandbyDef
	c.rwSeparate = rwSeparateDef
	c.rwPercent = rwPercentDef
	c.rwAutoDistribute = rwAutoDistributeDef
	c.rwStandbyRecoverTime = rwStandbyRecoverTimeDef
	c.rwIgnoreSql = false
	c.doSwitch = DO_SWITCH_WHEN_CONN_ERROR
	c.driverReconnect = false
	c.cluster = CLUSTER_TYPE_NORMAL
	c.cipherPath = cipherPathDef
	c.url = urlDef
	c.user = userDef
	c.password = passwordDef
	c.host = hostDef
	c.port = portDef
	c.appName = appNameDef
	c.osName = runtime.GOOS
	c.mppLocal = mppLocalDef
	c.socketTimeout = socketTimeoutDef
	c.connectTimeout = connectTimeoutDef
	c.sessionTimeout = sessionTimeoutDef
	c.osAuthType = osAuthTypeDef
	c.continueBatchOnError = continueBatchOnErrorDef
	c.batchAllowMaxErrors = 0
	c.escapeProcess = escapeProcessDef
	c.autoCommit = autoCommitDef
	c.maxRows = maxRowsDef
	c.rowPrefetch = rowPrefetchDef
	c.bufPrefetch = bufPrefetchDef
	c.lobMode = lobModeDef
	c.stmtPoolMaxSize = stmtPoolMaxSizeDef
	c.ignoreCase = ignoreCaseDef
	c.alwayseAllowCommit = alwayseAllowCommitDef
	c.batchType = 1
	c.batchNotOnCall = false
	c.isBdtaRS = isBdtaRSDef
	c.kerberosLoginConfPath = kerberosLoginConfPathDef
	c.uKeyName = uKeyNameDef
	c.uKeyPin = uKeyPinDef
	c.columnNameCase = COLUMN_NAME_NATURAL_CASE
	c.caseSensitive = caseSensitiveDef
	c.compatibleMode = compatibleModeDef
	_, tzs := time.Now().Zone()
	c.localTimezone = int16(tzs / 60)
	c.idGenerator = dmConntorIDGenerator

	c.logDir = LogDirDef
	c.logFlushFreq = LogFlushFreqDef
	c.logFlushQueueSize = LogFlushQueueSizeDef
	c.logBufferSize = LogBufferSizeDef
	c.statEnable = StatEnableDef
	c.statDir = StatDirDef
	c.statFlushFreq = StatFlushFreqDef
	c.statSlowSqlCount = StatSlowSqlCountDef
	c.statHighFreqSqlCount = StatHighFreqSqlCountDef
	c.statSqlMaxCount = StatSqlMaxCountDef
	c.statSqlRemoveMode = StatSqlRemoveModeDef
	return c
}

func (c *DmConnector) setAttributes(props *Properties) error {
	if props == nil || props.Len() == 0 {
		return nil
	}

	c.url = props.GetTrimString(UrlKey, c.url)
	c.host = props.GetTrimString(HostKey, c.host)
	c.port = int32(props.GetInt(PortKey, int(c.port), 0, 65535))
	c.user = props.GetString(UserKey, c.user)
	c.password = props.GetString(PasswordKey, c.password)
	c.dialName = props.GetString(DialNameKey, "")
	c.rwStandby = props.GetBool(RwStandbyKey, c.rwStandby)

	if b := props.GetBool(IsCompressKey, false); b {
		c.compress = Dm_build_90
	}

	c.compress = props.GetInt(CompressKey, c.compress, 0, 2)
	c.compressID = int8(props.GetInt(CompressIdKey, int(c.compressID), 0, 1))
	c.enRsCache = props.GetBool(EnRsCacheKey, c.enRsCache)
	c.localTimezone = int16(props.GetInt(TimeZoneKey, int(c.localTimezone), -720, 720))
	c.rsCacheSize = props.GetInt(RsCacheSizeKey, c.rsCacheSize, 0, int(INT32_MAX))
	c.rsRefreshFreq = props.GetInt(RsRefreshFreqKey, c.rsRefreshFreq, 0, int(INT32_MAX))
	c.loginMode = int32(props.GetInt(LoginModeKey, int(c.loginMode), 0, 4))
	c.loginStatus = props.GetInt(LoginStatusKey, c.loginStatus, 0, int(INT32_MAX))
	c.loginDscCtrl = props.GetBool(LoginDscCtrlKey, c.loginDscCtrl)
	c.switchTimes = int32(props.GetInt(SwitchTimesKey, int(c.switchTimes), 0, int(INT32_MAX)))
	c.switchInterval = props.GetInt(SwitchIntervalKey, c.switchInterval, 0, int(INT32_MAX))
	c.epSelector = int32(props.GetInt(EpSelectorKey, int(c.epSelector), 0, 1))
	c.loginEncrypt = props.GetBool(LoginEncryptKey, c.loginEncrypt)
	c.loginCertificate = props.GetTrimString(LoginCertificateKey, c.loginCertificate)
	c.dec2Double = props.GetBool(Dec2DoubleKey, c.dec2Double)
	parseLanguage(props.GetString(LanguageKey, ""))

	c.rwSeparate = props.GetBool(RwSeparateKey, c.rwSeparate)
	c.rwAutoDistribute = props.GetBool(RwAutoDistributeKey, c.rwAutoDistribute)
	c.rwPercent = int32(props.GetInt(RwPercentKey, int(c.rwPercent), 0, 100))
	c.rwHA = props.GetBool(RwHAKey, c.rwHA)
	c.rwStandbyRecoverTime = props.GetInt(RwStandbyRecoverTimeKey, c.rwStandbyRecoverTime, 0, int(INT32_MAX))
	c.rwIgnoreSql = props.GetBool(RwIgnoreSqlKey, c.rwIgnoreSql)
	c.doSwitch = int32(props.GetInt(DoSwitchKey, int(c.doSwitch), 0, 2))
	c.driverReconnect = props.GetBool(DriverReconnectKey, c.driverReconnect)
	c.parseCluster(props)
	c.cipherPath = props.GetTrimString(CipherPathKey, c.cipherPath)

	if props.GetBool(CompatibleOraKey, false) {
		c.compatibleMode = int(COMPATIBLE_MODE_ORACLE)
	}
	c.parseCompatibleMode(props)
	c.keyWords = props.GetStringArray(KeywordsKey, c.keyWords)

	c.appName = props.GetTrimString(AppNameKey, c.appName)
	c.osName = props.GetTrimString(OsNameKey, c.osName)
	c.mppLocal = props.GetBool(MppLocalKey, c.mppLocal)
	c.socketTimeout = props.GetInt(SocketTimeoutKey, c.socketTimeout, 0, int(INT32_MAX))
	c.connectTimeout = props.GetInt(ConnectTimeoutKey, c.connectTimeout, 0, int(INT32_MAX))
	c.sessionTimeout = props.GetInt(SessionTimeoutKey, c.sessionTimeout, 0, int(INT32_MAX))

	err := c.parseOsAuthType(props)
	if err != nil {
		return err
	}
	c.continueBatchOnError = props.GetBool(ContinueBatchOnErrorKey, c.continueBatchOnError)
	c.batchAllowMaxErrors = int32(props.GetInt(BatchAllowMaxErrorsKey, int(c.batchAllowMaxErrors), 0, int(INT32_MAX)))
	c.escapeProcess = props.GetBool(EscapeProcessKey, c.escapeProcess)
	c.autoCommit = props.GetBool(AutoCommitKey, c.autoCommit)
	c.maxRows = props.GetInt(MaxRowsKey, c.maxRows, 0, int(INT32_MAX))
	c.rowPrefetch = props.GetInt(RowPrefetchKey, c.rowPrefetch, 0, int(INT32_MAX))
	c.bufPrefetch = props.GetInt(BufPrefetchKey, c.bufPrefetch, int(Dm_build_76), int(Dm_build_77))
	c.lobMode = props.GetInt(LobModeKey, c.lobMode, 1, 2)
	c.stmtPoolMaxSize = props.GetInt(StmtPoolSizeKey, c.stmtPoolMaxSize, 0, int(INT32_MAX))
	c.ignoreCase = props.GetBool(IgnoreCaseKey, c.ignoreCase)
	c.alwayseAllowCommit = props.GetBool(AlwayseAllowCommitKey, c.alwayseAllowCommit)
	c.batchType = props.GetInt(BatchTypeKey, c.batchType, 1, 2)
	c.batchNotOnCall = props.GetBool(BatchNotOnCallKey, c.batchNotOnCall)
	c.isBdtaRS = props.GetBool(IsBdtaRSKey, c.isBdtaRS)
	c.sslFilesPath = props.GetTrimString(SslFilesPathKey, c.sslFilesPath)
	c.sslCertPath = props.GetTrimString(SslCertPathKey, c.sslCertPath)
	if c.sslCertPath == "" && c.sslFilesPath != "" {
		c.sslCertPath = filepath.Join(c.sslFilesPath, "client-cert.pem")
	}
	c.sslKeyPath = props.GetTrimString(SslKeyPathKey, c.sslKeyPath)
	if c.sslKeyPath == "" && c.sslFilesPath != "" {
		c.sslKeyPath = filepath.Join(c.sslKeyPath, "client-key.pem")
	}

	c.kerberosLoginConfPath = props.GetTrimString(KerberosLoginConfPathKey, c.kerberosLoginConfPath)

	c.uKeyName = props.GetTrimString(UKeyNameKey, c.uKeyName)
	c.uKeyPin = props.GetTrimString(UKeyPinKey, c.uKeyPin)

	c.svcConfPath = props.GetString("confPath", "")

	if props.GetBool(ColumnNameUpperCaseKey, false) {
		c.columnNameCase = COLUMN_NAME_UPPER_CASE
	}

	v := props.GetTrimString(ColumnNameCaseKey, "")
	if util.StringUtil.EqualsIgnoreCase(v, "upper") {
		c.columnNameCase = COLUMN_NAME_UPPER_CASE
	} else if util.StringUtil.EqualsIgnoreCase(v, "lower") {
		c.columnNameCase = COLUMN_NAME_LOWER_CASE
	}

	c.schema = props.GetTrimString(SchemaKey, c.schema)

	c.logLevel = ParseLogLevel(props)
	LogLevel = c.logLevel
	c.logDir = util.StringUtil.FormatDir(props.GetTrimString(LogDirKey, LogDirDef))
	LogDir = c.logDir
	c.logBufferSize = props.GetInt(LogBufferSizeKey, LogBufferSizeDef, 1, int(INT32_MAX))
	LogBufferSize = c.logBufferSize
	c.logFlushFreq = props.GetInt(LogFlushFreqKey, LogFlushFreqDef, 1, int(INT32_MAX))
	LogFlushFreq = c.logFlushFreq
	c.logFlushQueueSize = props.GetInt(LogFlusherQueueSizeKey, LogFlushQueueSizeDef, 1, int(INT32_MAX))
	LogFlushQueueSize = c.logFlushQueueSize

	c.statEnable = props.GetBool(StatEnableKey, StatEnableDef)
	StatEnable = c.statEnable
	c.statDir = util.StringUtil.FormatDir(props.GetTrimString(StatDirKey, StatDirDef))
	StatDir = c.statDir
	c.statFlushFreq = props.GetInt(StatFlushFreqKey, StatFlushFreqDef, 1, int(INT32_MAX))
	StatFlushFreq = c.statFlushFreq
	c.statHighFreqSqlCount = props.GetInt(StatHighFreqSqlCountKey, StatHighFreqSqlCountDef, 0, 1000)
	StatHighFreqSqlCount = c.statHighFreqSqlCount
	c.statSlowSqlCount = props.GetInt(StatSlowSqlCountKey, StatSlowSqlCountDef, 0, 1000)
	StatSlowSqlCount = c.statSlowSqlCount
	c.statSqlMaxCount = props.GetInt(StatSqlMaxCountKey, StatSqlMaxCountDef, 0, 100000)
	StatSqlMaxCount = c.statSqlMaxCount
	c.parseStatSqlRemoveMode(props)
	return nil
}

func (c *DmConnector) parseOsAuthType(props *Properties) error {
	value := props.GetString(OsAuthTypeKey, "")
	if value != "" && !util.StringUtil.IsDigit(value) {
		if util.StringUtil.EqualsIgnoreCase(value, "ON") {
			c.osAuthType = Dm_build_74
		} else if util.StringUtil.EqualsIgnoreCase(value, "SYSDBA") {
			c.osAuthType = Dm_build_70
		} else if util.StringUtil.EqualsIgnoreCase(value, "SYSAUDITOR") {
			c.osAuthType = Dm_build_72
		} else if util.StringUtil.EqualsIgnoreCase(value, "SYSSSO") {
			c.osAuthType = Dm_build_71
		} else if util.StringUtil.EqualsIgnoreCase(value, "AUTO") {
			c.osAuthType = Dm_build_73
		} else if util.StringUtil.EqualsIgnoreCase(value, "OFF") {
			c.osAuthType = Dm_build_69
		}
	} else {
		c.osAuthType = byte(props.GetInt(OsAuthTypeKey, int(c.osAuthType), 0, 4))
	}
	if c.user == "" && c.osAuthType == Dm_build_69 {
		c.user = "SYSDBA"
	} else if c.osAuthType != Dm_build_69 && c.user != "" {
		return ECGO_OSAUTH_ERROR.throw()
	} else if c.osAuthType != Dm_build_69 {
		c.user = os.Getenv("user")
		c.password = ""
	}
	return nil
}

func (c *DmConnector) parseCompatibleMode(props *Properties) {
	value := props.GetString(CompatibleModeKey, "")
	if value != "" && !util.StringUtil.IsDigit(value) {
		if util.StringUtil.EqualsIgnoreCase(value, "oracle") {
			c.compatibleMode = COMPATIBLE_MODE_ORACLE
		} else if util.StringUtil.EqualsIgnoreCase(value, "mysql") {
			c.compatibleMode = COMPATIBLE_MODE_MYSQL
		}
	} else {
		c.compatibleMode = props.GetInt(CompatibleModeKey, c.compatibleMode, 0, 2)
	}
}

func (c *DmConnector) parseStatSqlRemoveMode(props *Properties) {
	value := props.GetString(StatSqlRemoveModeKey, "")
	if value != "" && !util.StringUtil.IsDigit(value) {
		if util.StringUtil.EqualsIgnoreCase("oldest", value) || util.StringUtil.EqualsIgnoreCase("eldest", value) {
			c.statSqlRemoveMode = STAT_SQL_REMOVE_OLDEST
		} else if util.StringUtil.EqualsIgnoreCase("latest", value) {
			c.statSqlRemoveMode = STAT_SQL_REMOVE_LATEST
		}
	} else {
		c.statSqlRemoveMode = props.GetInt(StatSqlRemoveModeKey, StatSqlRemoveModeDef, 1, 2)
	}
}

func (c *DmConnector) parseCluster(props *Properties) {
	value := props.GetTrimString(ClusterKey, "")
	if util.StringUtil.EqualsIgnoreCase(value, "DSC") {
		c.cluster = CLUSTER_TYPE_DSC
	} else if util.StringUtil.EqualsIgnoreCase(value, "RW") {
		c.cluster = CLUSTER_TYPE_RW
	} else if util.StringUtil.EqualsIgnoreCase(value, "DW") {
		c.cluster = CLUSTER_TYPE_DW
	} else if util.StringUtil.EqualsIgnoreCase(value, "MPP") {
		c.cluster = CLUSTER_TYPE_MPP
	} else {
		c.cluster = CLUSTER_TYPE_NORMAL
	}
}

func (c *DmConnector) parseDSN(dsn string) (*Properties, string, error) {
	var dsnProps = NewProperties()
	url, err := url.Parse(dsn)
	if err != nil {
		return nil, "", err
	}
	if url.Scheme != "dm" {
		return nil, "", DSN_INVALID_SCHEMA
	}

	if url.User != nil {
		c.user = url.User.Username()
		c.password, _ = url.User.Password()
	}

	q := url.Query()
	for k := range q {
		dsnProps.Set(k, q.Get(k))
	}

	return dsnProps, url.Host, nil
}

func (c *DmConnector) BuildDSN() string {
	var buf bytes.Buffer

	buf.WriteString("dm://")

	if len(c.user) > 0 {
		buf.WriteString(url.QueryEscape(c.user))
		if len(c.password) > 0 {
			buf.WriteByte(':')
			buf.WriteString(url.QueryEscape(c.password))
		}
		buf.WriteByte('@')
	}

	if len(c.host) > 0 {
		buf.WriteString(c.host)
		if c.port > 0 {
			buf.WriteByte(':')
			buf.WriteString(strconv.Itoa(int(c.port)))
		}
	}

	hasParam := false
	if c.connectTimeout > 0 {
		if hasParam {
			buf.WriteString("&timeout=")
		} else {
			buf.WriteString("?timeout=")
			hasParam = true
		}
		buf.WriteString(strconv.Itoa(c.connectTimeout))
	}
	return buf.String()
}

func (c *DmConnector) mergeConfigs(dsn string) error {
	props, host, err := c.parseDSN(dsn)
	if err != nil {
		return err
	}

	driverInit(props.GetString("svcConfPath", ""))

	addressRemapStr := props.GetTrimString(AddressRemapKey, "")
	userRemapStr := props.GetTrimString(UserRemapKey, "")
	if addressRemapStr == "" {
		addressRemapStr = GlobalProperties.GetTrimString(AddressRemapKey, "")
	}
	if userRemapStr == "" {
		userRemapStr = GlobalProperties.GetTrimString(UserRemapKey, "")
	}

	host = c.remap(host, addressRemapStr)

	c.user = c.remap(c.user, userRemapStr)

	if a := props.GetTrimString(host, ""); a != "" {

		if strings.HasPrefix(a, "(") && strings.HasSuffix(a, ")") {
			a = strings.TrimSpace(a[1 : len(a)-1])
		}
		c.group = parseServerName(host, a)
		if c.group != nil {
			c.group.props = NewProperties()
			c.group.props.SetProperties(GlobalProperties)
		}
	} else if group, ok := ServerGroupMap[strings.ToLower(host)]; ok {

		c.group = group
	} else {
		host, port, err := net.SplitHostPort(host)
		if err == nil {
			ip := net.ParseIP(host)
			var v4InV6Prefix = []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0xff, 0xff}
			if ip != nil && len(ip) == net.IPv6len && !bytes.Equal(ip[0:12], v4InV6Prefix) {

				host = "[" + host + "]"
			}
		}

		c.host = host

		tmpPort, err := strconv.Atoi(port)
		if err != nil {
			c.port = portDef
		} else {
			c.port = int32(tmpPort)
		}

		c.group = newEPGroup(c.host+":"+strconv.Itoa(int(c.port)), []*ep{newEP(c.host, c.port)})
	}

	props.SetDiffProperties(c.group.props)

	props.SetDiffProperties(GlobalProperties)

	if props.GetBool(RwSeparateKey, false) {
		props.SetIfNotExist(LoginModeKey, strconv.Itoa(int(LOGIN_MODE_PRIMARY_ONLY)))
		props.SetIfNotExist(LoginStatusKey, strconv.Itoa(int(SERVER_STATUS_OPEN)))

		props.SetIfNotExist(DoSwitchKey, "true")
	}

	if err = c.setAttributes(props); err != nil {
		return err
	}
	return nil
}

func (c *DmConnector) remap(origin string, cfgStr string) string {
	if cfgStr == "" || origin == "" {
		return origin
	}

	maps := regexp.MustCompile("\\(.*?,.*?\\)").FindAllString(cfgStr, -1)
	for _, kvStr := range maps {
		kv := strings.Split(strings.TrimSpace(kvStr[1:len(kvStr)-1]), ",")
		if util.StringUtil.Equals(strings.TrimSpace(kv[0]), origin) {
			return strings.TrimSpace(kv[1])
		}
	}
	return origin
}

func (c *DmConnector) Connect(ctx context.Context) (driver.Conn, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.filterChain.reset().DmConnectorConnect(c, ctx)
}

func (c *DmConnector) Driver() driver.Driver {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.filterChain.reset().DmConnectorDriver(c)
}

func (c *DmConnector) connect(ctx context.Context) (*DmConnection, error) {
	if c.group != nil && len(c.group.epList) > 0 {
		return c.group.connect(c)
	} else {
		return c.connectSingle(ctx)
	}
}

func (c *DmConnector) driver() *DmDriver {
	return c.dmDriver
}

func (c *DmConnector) connectSingle(ctx context.Context) (*DmConnection, error) {
	var err error
	var dc = &DmConnection{
		closech:     make(chan struct{}),
		dmConnector: c,
		autoCommit:  c.autoCommit,
	}

	dc.createFilterChain(c, nil)
	dc.objId = -1
	dc.init()

	dc.Access, err = dm_build_1357(ctx, dc)
	if err != nil {
		return nil, err
	}

	dc.startWatcher()
	if err = dc.watchCancel(ctx); err != nil {
		return nil, err
	}
	defer dc.finish()

	if err = dc.Access.dm_build_1402(); err != nil {

		if !dc.closed.IsSet() {
			close(dc.closech)
			if dc.Access != nil {
				dc.Access.Close()
			}
			dc.closed.Set(true)
		}
		return nil, err
	}

	if c.schema != "" {
		_, err = dc.exec("set schema "+c.schema, nil)
		if err != nil {
			return nil, err
		}
	}

	return dc, nil
}
