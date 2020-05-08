// DO NOT EDIT: automatically generated code

package ovn_nb

import (
	"fmt"

	"github.com/pkg/errors"

	"yunion.io/x/ovsdb/types"
)

type OVNNorthbound struct {
	ACL                      ACLTable
	AddressSet               AddressSetTable
	Connection               ConnectionTable
	DHCPOptions              DHCPOptionsTable
	DNS                      DNSTable
	GatewayChassis           GatewayChassisTable
	LoadBalancer             LoadBalancerTable
	LogicalRouter            LogicalRouterTable
	LogicalRouterPort        LogicalRouterPortTable
	LogicalRouterStaticRoute LogicalRouterStaticRouteTable
	LogicalSwitch            LogicalSwitchTable
	LogicalSwitchPort        LogicalSwitchPortTable
	NAT                      NATTable
	NBGlobal                 NBGlobalTable
	QoS                      QoSTable
	SSL                      SSLTable
}

func (db OVNNorthbound) FindOneMatchNonZeros(irow types.IRow) types.IRow {
	switch row := irow.(type) {
	case *ACL:
		if r := db.ACL.FindOneMatchNonZeros(row); r != nil {
			return r
		}
		return nil
	}
	switch row := irow.(type) {
	case *AddressSet:
		if r := db.AddressSet.FindOneMatchNonZeros(row); r != nil {
			return r
		}
		return nil
	}
	switch row := irow.(type) {
	case *Connection:
		if r := db.Connection.FindOneMatchNonZeros(row); r != nil {
			return r
		}
		return nil
	}
	switch row := irow.(type) {
	case *DHCPOptions:
		if r := db.DHCPOptions.FindOneMatchNonZeros(row); r != nil {
			return r
		}
		return nil
	}
	switch row := irow.(type) {
	case *DNS:
		if r := db.DNS.FindOneMatchNonZeros(row); r != nil {
			return r
		}
		return nil
	}
	switch row := irow.(type) {
	case *GatewayChassis:
		if r := db.GatewayChassis.FindOneMatchNonZeros(row); r != nil {
			return r
		}
		return nil
	}
	switch row := irow.(type) {
	case *LoadBalancer:
		if r := db.LoadBalancer.FindOneMatchNonZeros(row); r != nil {
			return r
		}
		return nil
	}
	switch row := irow.(type) {
	case *LogicalRouter:
		if r := db.LogicalRouter.FindOneMatchNonZeros(row); r != nil {
			return r
		}
		return nil
	}
	switch row := irow.(type) {
	case *LogicalRouterPort:
		if r := db.LogicalRouterPort.FindOneMatchNonZeros(row); r != nil {
			return r
		}
		return nil
	}
	switch row := irow.(type) {
	case *LogicalRouterStaticRoute:
		if r := db.LogicalRouterStaticRoute.FindOneMatchNonZeros(row); r != nil {
			return r
		}
		return nil
	}
	switch row := irow.(type) {
	case *LogicalSwitch:
		if r := db.LogicalSwitch.FindOneMatchNonZeros(row); r != nil {
			return r
		}
		return nil
	}
	switch row := irow.(type) {
	case *LogicalSwitchPort:
		if r := db.LogicalSwitchPort.FindOneMatchNonZeros(row); r != nil {
			return r
		}
		return nil
	}
	switch row := irow.(type) {
	case *NAT:
		if r := db.NAT.FindOneMatchNonZeros(row); r != nil {
			return r
		}
		return nil
	}
	switch row := irow.(type) {
	case *NBGlobal:
		if r := db.NBGlobal.FindOneMatchNonZeros(row); r != nil {
			return r
		}
		return nil
	}
	switch row := irow.(type) {
	case *QoS:
		if r := db.QoS.FindOneMatchNonZeros(row); r != nil {
			return r
		}
		return nil
	}
	switch row := irow.(type) {
	case *SSL:
		if r := db.SSL.FindOneMatchNonZeros(row); r != nil {
			return r
		}
		return nil
	}
	panic(types.ErrBadType)
}

type ACLTable []ACL

var _ types.ITable = &ACLTable{}

func (tbl ACLTable) OvsdbTableName() string {
	return "ACL"
}

func (tbl ACLTable) OvsdbIsRoot() bool {
	return false
}

func (tbl ACLTable) Rows() []types.IRow {
	r := make([]types.IRow, len(tbl))
	for i := range tbl {
		r[i] = &tbl[i]
	}
	return r
}

func (tbl ACLTable) NewRow() types.IRow {
	return &ACL{}
}

func (tbl *ACLTable) AppendRow(irow types.IRow) {
	row := irow.(*ACL)
	*tbl = append(*tbl, *row)
}

func (tbl ACLTable) FindOneMatchNonZeros(row1 *ACL) *ACL {
	for i := range tbl {
		row := &tbl[i]
		if row.MatchNonZeros(row1) {
			return row
		}
	}
	return nil
}

type ACL struct {
	Uuid        string            `json:"_uuid"`
	Version     string            `json:"_version"`
	Action      string            `json:"action"`
	Direction   string            `json:"direction"`
	ExternalIds map[string]string `json:"external_ids"`
	Log         bool              `json:"log"`
	Match       string            `json:"match"`
	Name        *string           `json:"name"`
	Priority    int64             `json:"priority"`
	Severity    *string           `json:"severity"`
}

var _ types.IRow = &ACL{}

func (row *ACL) OvsdbTableName() string {
	return "ACL"
}

func (row *ACL) OvsdbIsRoot() bool {
	return false
}

func (row *ACL) OvsdbUuid() string {
	return row.Uuid
}

func (row *ACL) OvsdbCmdArgs() []string {
	r := []string{}
	r = append(r, types.OvsdbCmdArgsString("action", row.Action)...)
	r = append(r, types.OvsdbCmdArgsString("direction", row.Direction)...)
	r = append(r, types.OvsdbCmdArgsMapStringString("external_ids", row.ExternalIds)...)
	r = append(r, types.OvsdbCmdArgsBoolean("log", row.Log)...)
	r = append(r, types.OvsdbCmdArgsString("match", row.Match)...)
	r = append(r, types.OvsdbCmdArgsStringOptional("name", row.Name)...)
	r = append(r, types.OvsdbCmdArgsInteger("priority", row.Priority)...)
	r = append(r, types.OvsdbCmdArgsStringOptional("severity", row.Severity)...)
	return r
}

func (row *ACL) SetColumn(name string, val interface{}) (err error) {
	defer func() {
		if panicErr := recover(); panicErr != nil {
			err = errors.Wrapf(panicErr.(error), "%s: %#v", name, fmt.Sprintf("%#v", val))
		}
	}()
	switch name {
	case "_uuid":
		row.Uuid = types.EnsureUuid(val)
	case "_version":
		row.Version = types.EnsureUuid(val)
	case "action":
		row.Action = types.EnsureString(val)
	case "direction":
		row.Direction = types.EnsureString(val)
	case "external_ids":
		row.ExternalIds = types.EnsureMapStringString(val)
	case "log":
		row.Log = types.EnsureBoolean(val)
	case "match":
		row.Match = types.EnsureString(val)
	case "name":
		row.Name = types.EnsureStringOptional(val)
	case "priority":
		row.Priority = types.EnsureInteger(val)
	case "severity":
		row.Severity = types.EnsureStringOptional(val)
	default:
		panic(types.ErrUnknownColumn)
	}
	return
}

func (row *ACL) MatchNonZeros(row1 *ACL) bool {
	if !types.MatchUuidIfNonZero(row.Uuid, row1.Uuid) {
		return false
	}
	if !types.MatchUuidIfNonZero(row.Version, row1.Version) {
		return false
	}
	if !types.MatchStringIfNonZero(row.Action, row1.Action) {
		return false
	}
	if !types.MatchStringIfNonZero(row.Direction, row1.Direction) {
		return false
	}
	if !types.MatchMapStringStringIfNonZero(row.ExternalIds, row1.ExternalIds) {
		return false
	}
	if !types.MatchBooleanIfNonZero(row.Log, row1.Log) {
		return false
	}
	if !types.MatchStringIfNonZero(row.Match, row1.Match) {
		return false
	}
	if !types.MatchStringOptionalIfNonZero(row.Name, row1.Name) {
		return false
	}
	if !types.MatchIntegerIfNonZero(row.Priority, row1.Priority) {
		return false
	}
	if !types.MatchStringOptionalIfNonZero(row.Severity, row1.Severity) {
		return false
	}
	return true
}

func (row *ACL) HasExternalIds() bool {
	return true
}

func (row *ACL) SetExternalId(k, v string) {
	if row.ExternalIds == nil {
		row.ExternalIds = map[string]string{}
	}
	row.ExternalIds[k] = v
}

func (row *ACL) GetExternalId(k string) (string, bool) {
	if row.ExternalIds == nil {
		return "", false
	}
	r, ok := row.ExternalIds[k]
	return r, ok
}

func (row *ACL) RemoveExternalId(k string) (string, bool) {
	if row.ExternalIds == nil {
		return "", false
	}
	r, ok := row.ExternalIds[k]
	if ok {
		delete(row.ExternalIds, k)
	}
	return r, ok
}

type AddressSetTable []AddressSet

var _ types.ITable = &AddressSetTable{}

func (tbl AddressSetTable) OvsdbTableName() string {
	return "Address_Set"
}

func (tbl AddressSetTable) OvsdbIsRoot() bool {
	return true
}

func (tbl AddressSetTable) Rows() []types.IRow {
	r := make([]types.IRow, len(tbl))
	for i := range tbl {
		r[i] = &tbl[i]
	}
	return r
}

func (tbl AddressSetTable) NewRow() types.IRow {
	return &AddressSet{}
}

func (tbl *AddressSetTable) AppendRow(irow types.IRow) {
	row := irow.(*AddressSet)
	*tbl = append(*tbl, *row)
}

func (tbl AddressSetTable) FindOneMatchNonZeros(row1 *AddressSet) *AddressSet {
	for i := range tbl {
		row := &tbl[i]
		if row.MatchNonZeros(row1) {
			return row
		}
	}
	return nil
}

type AddressSet struct {
	Uuid        string            `json:"_uuid"`
	Version     string            `json:"_version"`
	Addresses   []string          `json:"addresses"`
	ExternalIds map[string]string `json:"external_ids"`
	Name        string            `json:"name"`
}

var _ types.IRow = &AddressSet{}

func (row *AddressSet) OvsdbTableName() string {
	return "Address_Set"
}

func (row *AddressSet) OvsdbIsRoot() bool {
	return true
}

func (row *AddressSet) OvsdbUuid() string {
	return row.Uuid
}

func (row *AddressSet) OvsdbCmdArgs() []string {
	r := []string{}
	r = append(r, types.OvsdbCmdArgsStringMultiples("addresses", row.Addresses)...)
	r = append(r, types.OvsdbCmdArgsMapStringString("external_ids", row.ExternalIds)...)
	r = append(r, types.OvsdbCmdArgsString("name", row.Name)...)
	return r
}

func (row *AddressSet) SetColumn(name string, val interface{}) (err error) {
	defer func() {
		if panicErr := recover(); panicErr != nil {
			err = errors.Wrapf(panicErr.(error), "%s: %#v", name, fmt.Sprintf("%#v", val))
		}
	}()
	switch name {
	case "_uuid":
		row.Uuid = types.EnsureUuid(val)
	case "_version":
		row.Version = types.EnsureUuid(val)
	case "addresses":
		row.Addresses = types.EnsureStringMultiples(val)
	case "external_ids":
		row.ExternalIds = types.EnsureMapStringString(val)
	case "name":
		row.Name = types.EnsureString(val)
	default:
		panic(types.ErrUnknownColumn)
	}
	return
}

func (row *AddressSet) MatchNonZeros(row1 *AddressSet) bool {
	if !types.MatchUuidIfNonZero(row.Uuid, row1.Uuid) {
		return false
	}
	if !types.MatchUuidIfNonZero(row.Version, row1.Version) {
		return false
	}
	if !types.MatchStringMultiplesIfNonZero(row.Addresses, row1.Addresses) {
		return false
	}
	if !types.MatchMapStringStringIfNonZero(row.ExternalIds, row1.ExternalIds) {
		return false
	}
	if !types.MatchStringIfNonZero(row.Name, row1.Name) {
		return false
	}
	return true
}

func (row *AddressSet) HasExternalIds() bool {
	return true
}

func (row *AddressSet) SetExternalId(k, v string) {
	if row.ExternalIds == nil {
		row.ExternalIds = map[string]string{}
	}
	row.ExternalIds[k] = v
}

func (row *AddressSet) GetExternalId(k string) (string, bool) {
	if row.ExternalIds == nil {
		return "", false
	}
	r, ok := row.ExternalIds[k]
	return r, ok
}

func (row *AddressSet) RemoveExternalId(k string) (string, bool) {
	if row.ExternalIds == nil {
		return "", false
	}
	r, ok := row.ExternalIds[k]
	if ok {
		delete(row.ExternalIds, k)
	}
	return r, ok
}

type ConnectionTable []Connection

var _ types.ITable = &ConnectionTable{}

func (tbl ConnectionTable) OvsdbTableName() string {
	return "Connection"
}

func (tbl ConnectionTable) OvsdbIsRoot() bool {
	return false
}

func (tbl ConnectionTable) Rows() []types.IRow {
	r := make([]types.IRow, len(tbl))
	for i := range tbl {
		r[i] = &tbl[i]
	}
	return r
}

func (tbl ConnectionTable) NewRow() types.IRow {
	return &Connection{}
}

func (tbl *ConnectionTable) AppendRow(irow types.IRow) {
	row := irow.(*Connection)
	*tbl = append(*tbl, *row)
}

func (tbl ConnectionTable) FindOneMatchNonZeros(row1 *Connection) *Connection {
	for i := range tbl {
		row := &tbl[i]
		if row.MatchNonZeros(row1) {
			return row
		}
	}
	return nil
}

type Connection struct {
	Uuid            string            `json:"_uuid"`
	Version         string            `json:"_version"`
	ExternalIds     map[string]string `json:"external_ids"`
	InactivityProbe *int64            `json:"inactivity_probe"`
	IsConnected     bool              `json:"is_connected"`
	MaxBackoff      *int64            `json:"max_backoff"`
	OtherConfig     map[string]string `json:"other_config"`
	Status          map[string]string `json:"status"`
	Target          string            `json:"target"`
}

var _ types.IRow = &Connection{}

func (row *Connection) OvsdbTableName() string {
	return "Connection"
}

func (row *Connection) OvsdbIsRoot() bool {
	return false
}

func (row *Connection) OvsdbUuid() string {
	return row.Uuid
}

func (row *Connection) OvsdbCmdArgs() []string {
	r := []string{}
	r = append(r, types.OvsdbCmdArgsMapStringString("external_ids", row.ExternalIds)...)
	r = append(r, types.OvsdbCmdArgsIntegerOptional("inactivity_probe", row.InactivityProbe)...)
	r = append(r, types.OvsdbCmdArgsBoolean("is_connected", row.IsConnected)...)
	r = append(r, types.OvsdbCmdArgsIntegerOptional("max_backoff", row.MaxBackoff)...)
	r = append(r, types.OvsdbCmdArgsMapStringString("other_config", row.OtherConfig)...)
	r = append(r, types.OvsdbCmdArgsMapStringString("status", row.Status)...)
	r = append(r, types.OvsdbCmdArgsString("target", row.Target)...)
	return r
}

func (row *Connection) SetColumn(name string, val interface{}) (err error) {
	defer func() {
		if panicErr := recover(); panicErr != nil {
			err = errors.Wrapf(panicErr.(error), "%s: %#v", name, fmt.Sprintf("%#v", val))
		}
	}()
	switch name {
	case "_uuid":
		row.Uuid = types.EnsureUuid(val)
	case "_version":
		row.Version = types.EnsureUuid(val)
	case "external_ids":
		row.ExternalIds = types.EnsureMapStringString(val)
	case "inactivity_probe":
		row.InactivityProbe = types.EnsureIntegerOptional(val)
	case "is_connected":
		row.IsConnected = types.EnsureBoolean(val)
	case "max_backoff":
		row.MaxBackoff = types.EnsureIntegerOptional(val)
	case "other_config":
		row.OtherConfig = types.EnsureMapStringString(val)
	case "status":
		row.Status = types.EnsureMapStringString(val)
	case "target":
		row.Target = types.EnsureString(val)
	default:
		panic(types.ErrUnknownColumn)
	}
	return
}

func (row *Connection) MatchNonZeros(row1 *Connection) bool {
	if !types.MatchUuidIfNonZero(row.Uuid, row1.Uuid) {
		return false
	}
	if !types.MatchUuidIfNonZero(row.Version, row1.Version) {
		return false
	}
	if !types.MatchMapStringStringIfNonZero(row.ExternalIds, row1.ExternalIds) {
		return false
	}
	if !types.MatchIntegerOptionalIfNonZero(row.InactivityProbe, row1.InactivityProbe) {
		return false
	}
	if !types.MatchBooleanIfNonZero(row.IsConnected, row1.IsConnected) {
		return false
	}
	if !types.MatchIntegerOptionalIfNonZero(row.MaxBackoff, row1.MaxBackoff) {
		return false
	}
	if !types.MatchMapStringStringIfNonZero(row.OtherConfig, row1.OtherConfig) {
		return false
	}
	if !types.MatchMapStringStringIfNonZero(row.Status, row1.Status) {
		return false
	}
	if !types.MatchStringIfNonZero(row.Target, row1.Target) {
		return false
	}
	return true
}

func (row *Connection) HasExternalIds() bool {
	return true
}

func (row *Connection) SetExternalId(k, v string) {
	if row.ExternalIds == nil {
		row.ExternalIds = map[string]string{}
	}
	row.ExternalIds[k] = v
}

func (row *Connection) GetExternalId(k string) (string, bool) {
	if row.ExternalIds == nil {
		return "", false
	}
	r, ok := row.ExternalIds[k]
	return r, ok
}

func (row *Connection) RemoveExternalId(k string) (string, bool) {
	if row.ExternalIds == nil {
		return "", false
	}
	r, ok := row.ExternalIds[k]
	if ok {
		delete(row.ExternalIds, k)
	}
	return r, ok
}

type DHCPOptionsTable []DHCPOptions

var _ types.ITable = &DHCPOptionsTable{}

func (tbl DHCPOptionsTable) OvsdbTableName() string {
	return "DHCP_Options"
}

func (tbl DHCPOptionsTable) OvsdbIsRoot() bool {
	return true
}

func (tbl DHCPOptionsTable) Rows() []types.IRow {
	r := make([]types.IRow, len(tbl))
	for i := range tbl {
		r[i] = &tbl[i]
	}
	return r
}

func (tbl DHCPOptionsTable) NewRow() types.IRow {
	return &DHCPOptions{}
}

func (tbl *DHCPOptionsTable) AppendRow(irow types.IRow) {
	row := irow.(*DHCPOptions)
	*tbl = append(*tbl, *row)
}

func (tbl DHCPOptionsTable) FindOneMatchNonZeros(row1 *DHCPOptions) *DHCPOptions {
	for i := range tbl {
		row := &tbl[i]
		if row.MatchNonZeros(row1) {
			return row
		}
	}
	return nil
}

type DHCPOptions struct {
	Uuid        string            `json:"_uuid"`
	Version     string            `json:"_version"`
	Cidr        string            `json:"cidr"`
	ExternalIds map[string]string `json:"external_ids"`
	Options     map[string]string `json:"options"`
}

var _ types.IRow = &DHCPOptions{}

func (row *DHCPOptions) OvsdbTableName() string {
	return "DHCP_Options"
}

func (row *DHCPOptions) OvsdbIsRoot() bool {
	return true
}

func (row *DHCPOptions) OvsdbUuid() string {
	return row.Uuid
}

func (row *DHCPOptions) OvsdbCmdArgs() []string {
	r := []string{}
	r = append(r, types.OvsdbCmdArgsString("cidr", row.Cidr)...)
	r = append(r, types.OvsdbCmdArgsMapStringString("external_ids", row.ExternalIds)...)
	r = append(r, types.OvsdbCmdArgsMapStringString("options", row.Options)...)
	return r
}

func (row *DHCPOptions) SetColumn(name string, val interface{}) (err error) {
	defer func() {
		if panicErr := recover(); panicErr != nil {
			err = errors.Wrapf(panicErr.(error), "%s: %#v", name, fmt.Sprintf("%#v", val))
		}
	}()
	switch name {
	case "_uuid":
		row.Uuid = types.EnsureUuid(val)
	case "_version":
		row.Version = types.EnsureUuid(val)
	case "cidr":
		row.Cidr = types.EnsureString(val)
	case "external_ids":
		row.ExternalIds = types.EnsureMapStringString(val)
	case "options":
		row.Options = types.EnsureMapStringString(val)
	default:
		panic(types.ErrUnknownColumn)
	}
	return
}

func (row *DHCPOptions) MatchNonZeros(row1 *DHCPOptions) bool {
	if !types.MatchUuidIfNonZero(row.Uuid, row1.Uuid) {
		return false
	}
	if !types.MatchUuidIfNonZero(row.Version, row1.Version) {
		return false
	}
	if !types.MatchStringIfNonZero(row.Cidr, row1.Cidr) {
		return false
	}
	if !types.MatchMapStringStringIfNonZero(row.ExternalIds, row1.ExternalIds) {
		return false
	}
	if !types.MatchMapStringStringIfNonZero(row.Options, row1.Options) {
		return false
	}
	return true
}

func (row *DHCPOptions) HasExternalIds() bool {
	return true
}

func (row *DHCPOptions) SetExternalId(k, v string) {
	if row.ExternalIds == nil {
		row.ExternalIds = map[string]string{}
	}
	row.ExternalIds[k] = v
}

func (row *DHCPOptions) GetExternalId(k string) (string, bool) {
	if row.ExternalIds == nil {
		return "", false
	}
	r, ok := row.ExternalIds[k]
	return r, ok
}

func (row *DHCPOptions) RemoveExternalId(k string) (string, bool) {
	if row.ExternalIds == nil {
		return "", false
	}
	r, ok := row.ExternalIds[k]
	if ok {
		delete(row.ExternalIds, k)
	}
	return r, ok
}

type DNSTable []DNS

var _ types.ITable = &DNSTable{}

func (tbl DNSTable) OvsdbTableName() string {
	return "DNS"
}

func (tbl DNSTable) OvsdbIsRoot() bool {
	return true
}

func (tbl DNSTable) Rows() []types.IRow {
	r := make([]types.IRow, len(tbl))
	for i := range tbl {
		r[i] = &tbl[i]
	}
	return r
}

func (tbl DNSTable) NewRow() types.IRow {
	return &DNS{}
}

func (tbl *DNSTable) AppendRow(irow types.IRow) {
	row := irow.(*DNS)
	*tbl = append(*tbl, *row)
}

func (tbl DNSTable) FindOneMatchNonZeros(row1 *DNS) *DNS {
	for i := range tbl {
		row := &tbl[i]
		if row.MatchNonZeros(row1) {
			return row
		}
	}
	return nil
}

type DNS struct {
	Uuid        string            `json:"_uuid"`
	Version     string            `json:"_version"`
	ExternalIds map[string]string `json:"external_ids"`
	Records     map[string]string `json:"records"`
}

var _ types.IRow = &DNS{}

func (row *DNS) OvsdbTableName() string {
	return "DNS"
}

func (row *DNS) OvsdbIsRoot() bool {
	return true
}

func (row *DNS) OvsdbUuid() string {
	return row.Uuid
}

func (row *DNS) OvsdbCmdArgs() []string {
	r := []string{}
	r = append(r, types.OvsdbCmdArgsMapStringString("external_ids", row.ExternalIds)...)
	r = append(r, types.OvsdbCmdArgsMapStringString("records", row.Records)...)
	return r
}

func (row *DNS) SetColumn(name string, val interface{}) (err error) {
	defer func() {
		if panicErr := recover(); panicErr != nil {
			err = errors.Wrapf(panicErr.(error), "%s: %#v", name, fmt.Sprintf("%#v", val))
		}
	}()
	switch name {
	case "_uuid":
		row.Uuid = types.EnsureUuid(val)
	case "_version":
		row.Version = types.EnsureUuid(val)
	case "external_ids":
		row.ExternalIds = types.EnsureMapStringString(val)
	case "records":
		row.Records = types.EnsureMapStringString(val)
	default:
		panic(types.ErrUnknownColumn)
	}
	return
}

func (row *DNS) MatchNonZeros(row1 *DNS) bool {
	if !types.MatchUuidIfNonZero(row.Uuid, row1.Uuid) {
		return false
	}
	if !types.MatchUuidIfNonZero(row.Version, row1.Version) {
		return false
	}
	if !types.MatchMapStringStringIfNonZero(row.ExternalIds, row1.ExternalIds) {
		return false
	}
	if !types.MatchMapStringStringIfNonZero(row.Records, row1.Records) {
		return false
	}
	return true
}

func (row *DNS) HasExternalIds() bool {
	return true
}

func (row *DNS) SetExternalId(k, v string) {
	if row.ExternalIds == nil {
		row.ExternalIds = map[string]string{}
	}
	row.ExternalIds[k] = v
}

func (row *DNS) GetExternalId(k string) (string, bool) {
	if row.ExternalIds == nil {
		return "", false
	}
	r, ok := row.ExternalIds[k]
	return r, ok
}

func (row *DNS) RemoveExternalId(k string) (string, bool) {
	if row.ExternalIds == nil {
		return "", false
	}
	r, ok := row.ExternalIds[k]
	if ok {
		delete(row.ExternalIds, k)
	}
	return r, ok
}

type GatewayChassisTable []GatewayChassis

var _ types.ITable = &GatewayChassisTable{}

func (tbl GatewayChassisTable) OvsdbTableName() string {
	return "Gateway_Chassis"
}

func (tbl GatewayChassisTable) OvsdbIsRoot() bool {
	return false
}

func (tbl GatewayChassisTable) Rows() []types.IRow {
	r := make([]types.IRow, len(tbl))
	for i := range tbl {
		r[i] = &tbl[i]
	}
	return r
}

func (tbl GatewayChassisTable) NewRow() types.IRow {
	return &GatewayChassis{}
}

func (tbl *GatewayChassisTable) AppendRow(irow types.IRow) {
	row := irow.(*GatewayChassis)
	*tbl = append(*tbl, *row)
}

func (tbl GatewayChassisTable) FindOneMatchNonZeros(row1 *GatewayChassis) *GatewayChassis {
	for i := range tbl {
		row := &tbl[i]
		if row.MatchNonZeros(row1) {
			return row
		}
	}
	return nil
}

type GatewayChassis struct {
	Uuid        string            `json:"_uuid"`
	Version     string            `json:"_version"`
	ChassisName string            `json:"chassis_name"`
	ExternalIds map[string]string `json:"external_ids"`
	Name        string            `json:"name"`
	Options     map[string]string `json:"options"`
	Priority    int64             `json:"priority"`
}

var _ types.IRow = &GatewayChassis{}

func (row *GatewayChassis) OvsdbTableName() string {
	return "Gateway_Chassis"
}

func (row *GatewayChassis) OvsdbIsRoot() bool {
	return false
}

func (row *GatewayChassis) OvsdbUuid() string {
	return row.Uuid
}

func (row *GatewayChassis) OvsdbCmdArgs() []string {
	r := []string{}
	r = append(r, types.OvsdbCmdArgsString("chassis_name", row.ChassisName)...)
	r = append(r, types.OvsdbCmdArgsMapStringString("external_ids", row.ExternalIds)...)
	r = append(r, types.OvsdbCmdArgsString("name", row.Name)...)
	r = append(r, types.OvsdbCmdArgsMapStringString("options", row.Options)...)
	r = append(r, types.OvsdbCmdArgsInteger("priority", row.Priority)...)
	return r
}

func (row *GatewayChassis) SetColumn(name string, val interface{}) (err error) {
	defer func() {
		if panicErr := recover(); panicErr != nil {
			err = errors.Wrapf(panicErr.(error), "%s: %#v", name, fmt.Sprintf("%#v", val))
		}
	}()
	switch name {
	case "_uuid":
		row.Uuid = types.EnsureUuid(val)
	case "_version":
		row.Version = types.EnsureUuid(val)
	case "chassis_name":
		row.ChassisName = types.EnsureString(val)
	case "external_ids":
		row.ExternalIds = types.EnsureMapStringString(val)
	case "name":
		row.Name = types.EnsureString(val)
	case "options":
		row.Options = types.EnsureMapStringString(val)
	case "priority":
		row.Priority = types.EnsureInteger(val)
	default:
		panic(types.ErrUnknownColumn)
	}
	return
}

func (row *GatewayChassis) MatchNonZeros(row1 *GatewayChassis) bool {
	if !types.MatchUuidIfNonZero(row.Uuid, row1.Uuid) {
		return false
	}
	if !types.MatchUuidIfNonZero(row.Version, row1.Version) {
		return false
	}
	if !types.MatchStringIfNonZero(row.ChassisName, row1.ChassisName) {
		return false
	}
	if !types.MatchMapStringStringIfNonZero(row.ExternalIds, row1.ExternalIds) {
		return false
	}
	if !types.MatchStringIfNonZero(row.Name, row1.Name) {
		return false
	}
	if !types.MatchMapStringStringIfNonZero(row.Options, row1.Options) {
		return false
	}
	if !types.MatchIntegerIfNonZero(row.Priority, row1.Priority) {
		return false
	}
	return true
}

func (row *GatewayChassis) HasExternalIds() bool {
	return true
}

func (row *GatewayChassis) SetExternalId(k, v string) {
	if row.ExternalIds == nil {
		row.ExternalIds = map[string]string{}
	}
	row.ExternalIds[k] = v
}

func (row *GatewayChassis) GetExternalId(k string) (string, bool) {
	if row.ExternalIds == nil {
		return "", false
	}
	r, ok := row.ExternalIds[k]
	return r, ok
}

func (row *GatewayChassis) RemoveExternalId(k string) (string, bool) {
	if row.ExternalIds == nil {
		return "", false
	}
	r, ok := row.ExternalIds[k]
	if ok {
		delete(row.ExternalIds, k)
	}
	return r, ok
}

type LoadBalancerTable []LoadBalancer

var _ types.ITable = &LoadBalancerTable{}

func (tbl LoadBalancerTable) OvsdbTableName() string {
	return "Load_Balancer"
}

func (tbl LoadBalancerTable) OvsdbIsRoot() bool {
	return true
}

func (tbl LoadBalancerTable) Rows() []types.IRow {
	r := make([]types.IRow, len(tbl))
	for i := range tbl {
		r[i] = &tbl[i]
	}
	return r
}

func (tbl LoadBalancerTable) NewRow() types.IRow {
	return &LoadBalancer{}
}

func (tbl *LoadBalancerTable) AppendRow(irow types.IRow) {
	row := irow.(*LoadBalancer)
	*tbl = append(*tbl, *row)
}

func (tbl LoadBalancerTable) FindOneMatchNonZeros(row1 *LoadBalancer) *LoadBalancer {
	for i := range tbl {
		row := &tbl[i]
		if row.MatchNonZeros(row1) {
			return row
		}
	}
	return nil
}

type LoadBalancer struct {
	Uuid        string            `json:"_uuid"`
	Version     string            `json:"_version"`
	ExternalIds map[string]string `json:"external_ids"`
	Name        string            `json:"name"`
	Protocol    *string           `json:"protocol"`
	Vips        map[string]string `json:"vips"`
}

var _ types.IRow = &LoadBalancer{}

func (row *LoadBalancer) OvsdbTableName() string {
	return "Load_Balancer"
}

func (row *LoadBalancer) OvsdbIsRoot() bool {
	return true
}

func (row *LoadBalancer) OvsdbUuid() string {
	return row.Uuid
}

func (row *LoadBalancer) OvsdbCmdArgs() []string {
	r := []string{}
	r = append(r, types.OvsdbCmdArgsMapStringString("external_ids", row.ExternalIds)...)
	r = append(r, types.OvsdbCmdArgsString("name", row.Name)...)
	r = append(r, types.OvsdbCmdArgsStringOptional("protocol", row.Protocol)...)
	r = append(r, types.OvsdbCmdArgsMapStringString("vips", row.Vips)...)
	return r
}

func (row *LoadBalancer) SetColumn(name string, val interface{}) (err error) {
	defer func() {
		if panicErr := recover(); panicErr != nil {
			err = errors.Wrapf(panicErr.(error), "%s: %#v", name, fmt.Sprintf("%#v", val))
		}
	}()
	switch name {
	case "_uuid":
		row.Uuid = types.EnsureUuid(val)
	case "_version":
		row.Version = types.EnsureUuid(val)
	case "external_ids":
		row.ExternalIds = types.EnsureMapStringString(val)
	case "name":
		row.Name = types.EnsureString(val)
	case "protocol":
		row.Protocol = types.EnsureStringOptional(val)
	case "vips":
		row.Vips = types.EnsureMapStringString(val)
	default:
		panic(types.ErrUnknownColumn)
	}
	return
}

func (row *LoadBalancer) MatchNonZeros(row1 *LoadBalancer) bool {
	if !types.MatchUuidIfNonZero(row.Uuid, row1.Uuid) {
		return false
	}
	if !types.MatchUuidIfNonZero(row.Version, row1.Version) {
		return false
	}
	if !types.MatchMapStringStringIfNonZero(row.ExternalIds, row1.ExternalIds) {
		return false
	}
	if !types.MatchStringIfNonZero(row.Name, row1.Name) {
		return false
	}
	if !types.MatchStringOptionalIfNonZero(row.Protocol, row1.Protocol) {
		return false
	}
	if !types.MatchMapStringStringIfNonZero(row.Vips, row1.Vips) {
		return false
	}
	return true
}

func (row *LoadBalancer) HasExternalIds() bool {
	return true
}

func (row *LoadBalancer) SetExternalId(k, v string) {
	if row.ExternalIds == nil {
		row.ExternalIds = map[string]string{}
	}
	row.ExternalIds[k] = v
}

func (row *LoadBalancer) GetExternalId(k string) (string, bool) {
	if row.ExternalIds == nil {
		return "", false
	}
	r, ok := row.ExternalIds[k]
	return r, ok
}

func (row *LoadBalancer) RemoveExternalId(k string) (string, bool) {
	if row.ExternalIds == nil {
		return "", false
	}
	r, ok := row.ExternalIds[k]
	if ok {
		delete(row.ExternalIds, k)
	}
	return r, ok
}

type LogicalRouterTable []LogicalRouter

var _ types.ITable = &LogicalRouterTable{}

func (tbl LogicalRouterTable) OvsdbTableName() string {
	return "Logical_Router"
}

func (tbl LogicalRouterTable) OvsdbIsRoot() bool {
	return true
}

func (tbl LogicalRouterTable) Rows() []types.IRow {
	r := make([]types.IRow, len(tbl))
	for i := range tbl {
		r[i] = &tbl[i]
	}
	return r
}

func (tbl LogicalRouterTable) NewRow() types.IRow {
	return &LogicalRouter{}
}

func (tbl *LogicalRouterTable) AppendRow(irow types.IRow) {
	row := irow.(*LogicalRouter)
	*tbl = append(*tbl, *row)
}

func (tbl LogicalRouterTable) FindOneMatchNonZeros(row1 *LogicalRouter) *LogicalRouter {
	for i := range tbl {
		row := &tbl[i]
		if row.MatchNonZeros(row1) {
			return row
		}
	}
	return nil
}

type LogicalRouter struct {
	Uuid         string            `json:"_uuid"`
	Version      string            `json:"_version"`
	Enabled      *bool             `json:"enabled"`
	ExternalIds  map[string]string `json:"external_ids"`
	LoadBalancer []string          `json:"load_balancer"`
	Name         string            `json:"name"`
	Nat          []string          `json:"nat"`
	Options      map[string]string `json:"options"`
	Ports        []string          `json:"ports"`
	StaticRoutes []string          `json:"static_routes"`
}

var _ types.IRow = &LogicalRouter{}

func (row *LogicalRouter) OvsdbTableName() string {
	return "Logical_Router"
}

func (row *LogicalRouter) OvsdbIsRoot() bool {
	return true
}

func (row *LogicalRouter) OvsdbUuid() string {
	return row.Uuid
}

func (row *LogicalRouter) OvsdbCmdArgs() []string {
	r := []string{}
	r = append(r, types.OvsdbCmdArgsBooleanOptional("enabled", row.Enabled)...)
	r = append(r, types.OvsdbCmdArgsMapStringString("external_ids", row.ExternalIds)...)
	r = append(r, types.OvsdbCmdArgsUuidMultiples("load_balancer", row.LoadBalancer)...)
	r = append(r, types.OvsdbCmdArgsString("name", row.Name)...)
	r = append(r, types.OvsdbCmdArgsUuidMultiples("nat", row.Nat)...)
	r = append(r, types.OvsdbCmdArgsMapStringString("options", row.Options)...)
	r = append(r, types.OvsdbCmdArgsUuidMultiples("ports", row.Ports)...)
	r = append(r, types.OvsdbCmdArgsUuidMultiples("static_routes", row.StaticRoutes)...)
	return r
}

func (row *LogicalRouter) SetColumn(name string, val interface{}) (err error) {
	defer func() {
		if panicErr := recover(); panicErr != nil {
			err = errors.Wrapf(panicErr.(error), "%s: %#v", name, fmt.Sprintf("%#v", val))
		}
	}()
	switch name {
	case "_uuid":
		row.Uuid = types.EnsureUuid(val)
	case "_version":
		row.Version = types.EnsureUuid(val)
	case "enabled":
		row.Enabled = types.EnsureBooleanOptional(val)
	case "external_ids":
		row.ExternalIds = types.EnsureMapStringString(val)
	case "load_balancer":
		row.LoadBalancer = types.EnsureUuidMultiples(val)
	case "name":
		row.Name = types.EnsureString(val)
	case "nat":
		row.Nat = types.EnsureUuidMultiples(val)
	case "options":
		row.Options = types.EnsureMapStringString(val)
	case "ports":
		row.Ports = types.EnsureUuidMultiples(val)
	case "static_routes":
		row.StaticRoutes = types.EnsureUuidMultiples(val)
	default:
		panic(types.ErrUnknownColumn)
	}
	return
}

func (row *LogicalRouter) MatchNonZeros(row1 *LogicalRouter) bool {
	if !types.MatchUuidIfNonZero(row.Uuid, row1.Uuid) {
		return false
	}
	if !types.MatchUuidIfNonZero(row.Version, row1.Version) {
		return false
	}
	if !types.MatchBooleanOptionalIfNonZero(row.Enabled, row1.Enabled) {
		return false
	}
	if !types.MatchMapStringStringIfNonZero(row.ExternalIds, row1.ExternalIds) {
		return false
	}
	if !types.MatchUuidMultiplesIfNonZero(row.LoadBalancer, row1.LoadBalancer) {
		return false
	}
	if !types.MatchStringIfNonZero(row.Name, row1.Name) {
		return false
	}
	if !types.MatchUuidMultiplesIfNonZero(row.Nat, row1.Nat) {
		return false
	}
	if !types.MatchMapStringStringIfNonZero(row.Options, row1.Options) {
		return false
	}
	if !types.MatchUuidMultiplesIfNonZero(row.Ports, row1.Ports) {
		return false
	}
	if !types.MatchUuidMultiplesIfNonZero(row.StaticRoutes, row1.StaticRoutes) {
		return false
	}
	return true
}

func (row *LogicalRouter) HasExternalIds() bool {
	return true
}

func (row *LogicalRouter) SetExternalId(k, v string) {
	if row.ExternalIds == nil {
		row.ExternalIds = map[string]string{}
	}
	row.ExternalIds[k] = v
}

func (row *LogicalRouter) GetExternalId(k string) (string, bool) {
	if row.ExternalIds == nil {
		return "", false
	}
	r, ok := row.ExternalIds[k]
	return r, ok
}

func (row *LogicalRouter) RemoveExternalId(k string) (string, bool) {
	if row.ExternalIds == nil {
		return "", false
	}
	r, ok := row.ExternalIds[k]
	if ok {
		delete(row.ExternalIds, k)
	}
	return r, ok
}

type LogicalRouterPortTable []LogicalRouterPort

var _ types.ITable = &LogicalRouterPortTable{}

func (tbl LogicalRouterPortTable) OvsdbTableName() string {
	return "Logical_Router_Port"
}

func (tbl LogicalRouterPortTable) OvsdbIsRoot() bool {
	return false
}

func (tbl LogicalRouterPortTable) Rows() []types.IRow {
	r := make([]types.IRow, len(tbl))
	for i := range tbl {
		r[i] = &tbl[i]
	}
	return r
}

func (tbl LogicalRouterPortTable) NewRow() types.IRow {
	return &LogicalRouterPort{}
}

func (tbl *LogicalRouterPortTable) AppendRow(irow types.IRow) {
	row := irow.(*LogicalRouterPort)
	*tbl = append(*tbl, *row)
}

func (tbl LogicalRouterPortTable) FindOneMatchNonZeros(row1 *LogicalRouterPort) *LogicalRouterPort {
	for i := range tbl {
		row := &tbl[i]
		if row.MatchNonZeros(row1) {
			return row
		}
	}
	return nil
}

type LogicalRouterPort struct {
	Uuid           string            `json:"_uuid"`
	Version        string            `json:"_version"`
	Enabled        *bool             `json:"enabled"`
	ExternalIds    map[string]string `json:"external_ids"`
	GatewayChassis []string          `json:"gateway_chassis"`
	Ipv6RaConfigs  map[string]string `json:"ipv6_ra_configs"`
	Mac            string            `json:"mac"`
	Name           string            `json:"name"`
	Networks       []string          `json:"networks"`
	Options        map[string]string `json:"options"`
	Peer           *string           `json:"peer"`
}

var _ types.IRow = &LogicalRouterPort{}

func (row *LogicalRouterPort) OvsdbTableName() string {
	return "Logical_Router_Port"
}

func (row *LogicalRouterPort) OvsdbIsRoot() bool {
	return false
}

func (row *LogicalRouterPort) OvsdbUuid() string {
	return row.Uuid
}

func (row *LogicalRouterPort) OvsdbCmdArgs() []string {
	r := []string{}
	r = append(r, types.OvsdbCmdArgsBooleanOptional("enabled", row.Enabled)...)
	r = append(r, types.OvsdbCmdArgsMapStringString("external_ids", row.ExternalIds)...)
	r = append(r, types.OvsdbCmdArgsUuidMultiples("gateway_chassis", row.GatewayChassis)...)
	r = append(r, types.OvsdbCmdArgsMapStringString("ipv6_ra_configs", row.Ipv6RaConfigs)...)
	r = append(r, types.OvsdbCmdArgsString("mac", row.Mac)...)
	r = append(r, types.OvsdbCmdArgsString("name", row.Name)...)
	r = append(r, types.OvsdbCmdArgsStringMultiples("networks", row.Networks)...)
	r = append(r, types.OvsdbCmdArgsMapStringString("options", row.Options)...)
	r = append(r, types.OvsdbCmdArgsStringOptional("peer", row.Peer)...)
	return r
}

func (row *LogicalRouterPort) SetColumn(name string, val interface{}) (err error) {
	defer func() {
		if panicErr := recover(); panicErr != nil {
			err = errors.Wrapf(panicErr.(error), "%s: %#v", name, fmt.Sprintf("%#v", val))
		}
	}()
	switch name {
	case "_uuid":
		row.Uuid = types.EnsureUuid(val)
	case "_version":
		row.Version = types.EnsureUuid(val)
	case "enabled":
		row.Enabled = types.EnsureBooleanOptional(val)
	case "external_ids":
		row.ExternalIds = types.EnsureMapStringString(val)
	case "gateway_chassis":
		row.GatewayChassis = types.EnsureUuidMultiples(val)
	case "ipv6_ra_configs":
		row.Ipv6RaConfigs = types.EnsureMapStringString(val)
	case "mac":
		row.Mac = types.EnsureString(val)
	case "name":
		row.Name = types.EnsureString(val)
	case "networks":
		row.Networks = types.EnsureStringMultiples(val)
	case "options":
		row.Options = types.EnsureMapStringString(val)
	case "peer":
		row.Peer = types.EnsureStringOptional(val)
	default:
		panic(types.ErrUnknownColumn)
	}
	return
}

func (row *LogicalRouterPort) MatchNonZeros(row1 *LogicalRouterPort) bool {
	if !types.MatchUuidIfNonZero(row.Uuid, row1.Uuid) {
		return false
	}
	if !types.MatchUuidIfNonZero(row.Version, row1.Version) {
		return false
	}
	if !types.MatchBooleanOptionalIfNonZero(row.Enabled, row1.Enabled) {
		return false
	}
	if !types.MatchMapStringStringIfNonZero(row.ExternalIds, row1.ExternalIds) {
		return false
	}
	if !types.MatchUuidMultiplesIfNonZero(row.GatewayChassis, row1.GatewayChassis) {
		return false
	}
	if !types.MatchMapStringStringIfNonZero(row.Ipv6RaConfigs, row1.Ipv6RaConfigs) {
		return false
	}
	if !types.MatchStringIfNonZero(row.Mac, row1.Mac) {
		return false
	}
	if !types.MatchStringIfNonZero(row.Name, row1.Name) {
		return false
	}
	if !types.MatchStringMultiplesIfNonZero(row.Networks, row1.Networks) {
		return false
	}
	if !types.MatchMapStringStringIfNonZero(row.Options, row1.Options) {
		return false
	}
	if !types.MatchStringOptionalIfNonZero(row.Peer, row1.Peer) {
		return false
	}
	return true
}

func (row *LogicalRouterPort) HasExternalIds() bool {
	return true
}

func (row *LogicalRouterPort) SetExternalId(k, v string) {
	if row.ExternalIds == nil {
		row.ExternalIds = map[string]string{}
	}
	row.ExternalIds[k] = v
}

func (row *LogicalRouterPort) GetExternalId(k string) (string, bool) {
	if row.ExternalIds == nil {
		return "", false
	}
	r, ok := row.ExternalIds[k]
	return r, ok
}

func (row *LogicalRouterPort) RemoveExternalId(k string) (string, bool) {
	if row.ExternalIds == nil {
		return "", false
	}
	r, ok := row.ExternalIds[k]
	if ok {
		delete(row.ExternalIds, k)
	}
	return r, ok
}

type LogicalRouterStaticRouteTable []LogicalRouterStaticRoute

var _ types.ITable = &LogicalRouterStaticRouteTable{}

func (tbl LogicalRouterStaticRouteTable) OvsdbTableName() string {
	return "Logical_Router_Static_Route"
}

func (tbl LogicalRouterStaticRouteTable) OvsdbIsRoot() bool {
	return false
}

func (tbl LogicalRouterStaticRouteTable) Rows() []types.IRow {
	r := make([]types.IRow, len(tbl))
	for i := range tbl {
		r[i] = &tbl[i]
	}
	return r
}

func (tbl LogicalRouterStaticRouteTable) NewRow() types.IRow {
	return &LogicalRouterStaticRoute{}
}

func (tbl *LogicalRouterStaticRouteTable) AppendRow(irow types.IRow) {
	row := irow.(*LogicalRouterStaticRoute)
	*tbl = append(*tbl, *row)
}

func (tbl LogicalRouterStaticRouteTable) FindOneMatchNonZeros(row1 *LogicalRouterStaticRoute) *LogicalRouterStaticRoute {
	for i := range tbl {
		row := &tbl[i]
		if row.MatchNonZeros(row1) {
			return row
		}
	}
	return nil
}

type LogicalRouterStaticRoute struct {
	Uuid        string            `json:"_uuid"`
	Version     string            `json:"_version"`
	ExternalIds map[string]string `json:"external_ids"`
	IpPrefix    string            `json:"ip_prefix"`
	Nexthop     string            `json:"nexthop"`
	OutputPort  *string           `json:"output_port"`
	Policy      *string           `json:"policy"`
}

var _ types.IRow = &LogicalRouterStaticRoute{}

func (row *LogicalRouterStaticRoute) OvsdbTableName() string {
	return "Logical_Router_Static_Route"
}

func (row *LogicalRouterStaticRoute) OvsdbIsRoot() bool {
	return false
}

func (row *LogicalRouterStaticRoute) OvsdbUuid() string {
	return row.Uuid
}

func (row *LogicalRouterStaticRoute) OvsdbCmdArgs() []string {
	r := []string{}
	r = append(r, types.OvsdbCmdArgsMapStringString("external_ids", row.ExternalIds)...)
	r = append(r, types.OvsdbCmdArgsString("ip_prefix", row.IpPrefix)...)
	r = append(r, types.OvsdbCmdArgsString("nexthop", row.Nexthop)...)
	r = append(r, types.OvsdbCmdArgsStringOptional("output_port", row.OutputPort)...)
	r = append(r, types.OvsdbCmdArgsStringOptional("policy", row.Policy)...)
	return r
}

func (row *LogicalRouterStaticRoute) SetColumn(name string, val interface{}) (err error) {
	defer func() {
		if panicErr := recover(); panicErr != nil {
			err = errors.Wrapf(panicErr.(error), "%s: %#v", name, fmt.Sprintf("%#v", val))
		}
	}()
	switch name {
	case "_uuid":
		row.Uuid = types.EnsureUuid(val)
	case "_version":
		row.Version = types.EnsureUuid(val)
	case "external_ids":
		row.ExternalIds = types.EnsureMapStringString(val)
	case "ip_prefix":
		row.IpPrefix = types.EnsureString(val)
	case "nexthop":
		row.Nexthop = types.EnsureString(val)
	case "output_port":
		row.OutputPort = types.EnsureStringOptional(val)
	case "policy":
		row.Policy = types.EnsureStringOptional(val)
	default:
		panic(types.ErrUnknownColumn)
	}
	return
}

func (row *LogicalRouterStaticRoute) MatchNonZeros(row1 *LogicalRouterStaticRoute) bool {
	if !types.MatchUuidIfNonZero(row.Uuid, row1.Uuid) {
		return false
	}
	if !types.MatchUuidIfNonZero(row.Version, row1.Version) {
		return false
	}
	if !types.MatchMapStringStringIfNonZero(row.ExternalIds, row1.ExternalIds) {
		return false
	}
	if !types.MatchStringIfNonZero(row.IpPrefix, row1.IpPrefix) {
		return false
	}
	if !types.MatchStringIfNonZero(row.Nexthop, row1.Nexthop) {
		return false
	}
	if !types.MatchStringOptionalIfNonZero(row.OutputPort, row1.OutputPort) {
		return false
	}
	if !types.MatchStringOptionalIfNonZero(row.Policy, row1.Policy) {
		return false
	}
	return true
}

func (row *LogicalRouterStaticRoute) HasExternalIds() bool {
	return true
}

func (row *LogicalRouterStaticRoute) SetExternalId(k, v string) {
	if row.ExternalIds == nil {
		row.ExternalIds = map[string]string{}
	}
	row.ExternalIds[k] = v
}

func (row *LogicalRouterStaticRoute) GetExternalId(k string) (string, bool) {
	if row.ExternalIds == nil {
		return "", false
	}
	r, ok := row.ExternalIds[k]
	return r, ok
}

func (row *LogicalRouterStaticRoute) RemoveExternalId(k string) (string, bool) {
	if row.ExternalIds == nil {
		return "", false
	}
	r, ok := row.ExternalIds[k]
	if ok {
		delete(row.ExternalIds, k)
	}
	return r, ok
}

type LogicalSwitchTable []LogicalSwitch

var _ types.ITable = &LogicalSwitchTable{}

func (tbl LogicalSwitchTable) OvsdbTableName() string {
	return "Logical_Switch"
}

func (tbl LogicalSwitchTable) OvsdbIsRoot() bool {
	return true
}

func (tbl LogicalSwitchTable) Rows() []types.IRow {
	r := make([]types.IRow, len(tbl))
	for i := range tbl {
		r[i] = &tbl[i]
	}
	return r
}

func (tbl LogicalSwitchTable) NewRow() types.IRow {
	return &LogicalSwitch{}
}

func (tbl *LogicalSwitchTable) AppendRow(irow types.IRow) {
	row := irow.(*LogicalSwitch)
	*tbl = append(*tbl, *row)
}

func (tbl LogicalSwitchTable) FindOneMatchNonZeros(row1 *LogicalSwitch) *LogicalSwitch {
	for i := range tbl {
		row := &tbl[i]
		if row.MatchNonZeros(row1) {
			return row
		}
	}
	return nil
}

type LogicalSwitch struct {
	Uuid         string            `json:"_uuid"`
	Version      string            `json:"_version"`
	Acls         []string          `json:"acls"`
	DnsRecords   []string          `json:"dns_records"`
	ExternalIds  map[string]string `json:"external_ids"`
	LoadBalancer []string          `json:"load_balancer"`
	Name         string            `json:"name"`
	OtherConfig  map[string]string `json:"other_config"`
	Ports        []string          `json:"ports"`
	QosRules     []string          `json:"qos_rules"`
}

var _ types.IRow = &LogicalSwitch{}

func (row *LogicalSwitch) OvsdbTableName() string {
	return "Logical_Switch"
}

func (row *LogicalSwitch) OvsdbIsRoot() bool {
	return true
}

func (row *LogicalSwitch) OvsdbUuid() string {
	return row.Uuid
}

func (row *LogicalSwitch) OvsdbCmdArgs() []string {
	r := []string{}
	r = append(r, types.OvsdbCmdArgsUuidMultiples("acls", row.Acls)...)
	r = append(r, types.OvsdbCmdArgsUuidMultiples("dns_records", row.DnsRecords)...)
	r = append(r, types.OvsdbCmdArgsMapStringString("external_ids", row.ExternalIds)...)
	r = append(r, types.OvsdbCmdArgsUuidMultiples("load_balancer", row.LoadBalancer)...)
	r = append(r, types.OvsdbCmdArgsString("name", row.Name)...)
	r = append(r, types.OvsdbCmdArgsMapStringString("other_config", row.OtherConfig)...)
	r = append(r, types.OvsdbCmdArgsUuidMultiples("ports", row.Ports)...)
	r = append(r, types.OvsdbCmdArgsUuidMultiples("qos_rules", row.QosRules)...)
	return r
}

func (row *LogicalSwitch) SetColumn(name string, val interface{}) (err error) {
	defer func() {
		if panicErr := recover(); panicErr != nil {
			err = errors.Wrapf(panicErr.(error), "%s: %#v", name, fmt.Sprintf("%#v", val))
		}
	}()
	switch name {
	case "_uuid":
		row.Uuid = types.EnsureUuid(val)
	case "_version":
		row.Version = types.EnsureUuid(val)
	case "acls":
		row.Acls = types.EnsureUuidMultiples(val)
	case "dns_records":
		row.DnsRecords = types.EnsureUuidMultiples(val)
	case "external_ids":
		row.ExternalIds = types.EnsureMapStringString(val)
	case "load_balancer":
		row.LoadBalancer = types.EnsureUuidMultiples(val)
	case "name":
		row.Name = types.EnsureString(val)
	case "other_config":
		row.OtherConfig = types.EnsureMapStringString(val)
	case "ports":
		row.Ports = types.EnsureUuidMultiples(val)
	case "qos_rules":
		row.QosRules = types.EnsureUuidMultiples(val)
	default:
		panic(types.ErrUnknownColumn)
	}
	return
}

func (row *LogicalSwitch) MatchNonZeros(row1 *LogicalSwitch) bool {
	if !types.MatchUuidIfNonZero(row.Uuid, row1.Uuid) {
		return false
	}
	if !types.MatchUuidIfNonZero(row.Version, row1.Version) {
		return false
	}
	if !types.MatchUuidMultiplesIfNonZero(row.Acls, row1.Acls) {
		return false
	}
	if !types.MatchUuidMultiplesIfNonZero(row.DnsRecords, row1.DnsRecords) {
		return false
	}
	if !types.MatchMapStringStringIfNonZero(row.ExternalIds, row1.ExternalIds) {
		return false
	}
	if !types.MatchUuidMultiplesIfNonZero(row.LoadBalancer, row1.LoadBalancer) {
		return false
	}
	if !types.MatchStringIfNonZero(row.Name, row1.Name) {
		return false
	}
	if !types.MatchMapStringStringIfNonZero(row.OtherConfig, row1.OtherConfig) {
		return false
	}
	if !types.MatchUuidMultiplesIfNonZero(row.Ports, row1.Ports) {
		return false
	}
	if !types.MatchUuidMultiplesIfNonZero(row.QosRules, row1.QosRules) {
		return false
	}
	return true
}

func (row *LogicalSwitch) HasExternalIds() bool {
	return true
}

func (row *LogicalSwitch) SetExternalId(k, v string) {
	if row.ExternalIds == nil {
		row.ExternalIds = map[string]string{}
	}
	row.ExternalIds[k] = v
}

func (row *LogicalSwitch) GetExternalId(k string) (string, bool) {
	if row.ExternalIds == nil {
		return "", false
	}
	r, ok := row.ExternalIds[k]
	return r, ok
}

func (row *LogicalSwitch) RemoveExternalId(k string) (string, bool) {
	if row.ExternalIds == nil {
		return "", false
	}
	r, ok := row.ExternalIds[k]
	if ok {
		delete(row.ExternalIds, k)
	}
	return r, ok
}

type LogicalSwitchPortTable []LogicalSwitchPort

var _ types.ITable = &LogicalSwitchPortTable{}

func (tbl LogicalSwitchPortTable) OvsdbTableName() string {
	return "Logical_Switch_Port"
}

func (tbl LogicalSwitchPortTable) OvsdbIsRoot() bool {
	return false
}

func (tbl LogicalSwitchPortTable) Rows() []types.IRow {
	r := make([]types.IRow, len(tbl))
	for i := range tbl {
		r[i] = &tbl[i]
	}
	return r
}

func (tbl LogicalSwitchPortTable) NewRow() types.IRow {
	return &LogicalSwitchPort{}
}

func (tbl *LogicalSwitchPortTable) AppendRow(irow types.IRow) {
	row := irow.(*LogicalSwitchPort)
	*tbl = append(*tbl, *row)
}

func (tbl LogicalSwitchPortTable) FindOneMatchNonZeros(row1 *LogicalSwitchPort) *LogicalSwitchPort {
	for i := range tbl {
		row := &tbl[i]
		if row.MatchNonZeros(row1) {
			return row
		}
	}
	return nil
}

type LogicalSwitchPort struct {
	Uuid             string            `json:"_uuid"`
	Version          string            `json:"_version"`
	Addresses        []string          `json:"addresses"`
	Dhcpv4Options    *string           `json:"dhcpv4_options"`
	Dhcpv6Options    *string           `json:"dhcpv6_options"`
	DynamicAddresses *string           `json:"dynamic_addresses"`
	Enabled          *bool             `json:"enabled"`
	ExternalIds      map[string]string `json:"external_ids"`
	Name             string            `json:"name"`
	Options          map[string]string `json:"options"`
	ParentName       *string           `json:"parent_name"`
	PortSecurity     []string          `json:"port_security"`
	Tag              *int64            `json:"tag"`
	TagRequest       *int64            `json:"tag_request"`
	Type             string            `json:"type"`
	Up               *bool             `json:"up"`
}

var _ types.IRow = &LogicalSwitchPort{}

func (row *LogicalSwitchPort) OvsdbTableName() string {
	return "Logical_Switch_Port"
}

func (row *LogicalSwitchPort) OvsdbIsRoot() bool {
	return false
}

func (row *LogicalSwitchPort) OvsdbUuid() string {
	return row.Uuid
}

func (row *LogicalSwitchPort) OvsdbCmdArgs() []string {
	r := []string{}
	r = append(r, types.OvsdbCmdArgsStringMultiples("addresses", row.Addresses)...)
	r = append(r, types.OvsdbCmdArgsUuidOptional("dhcpv4_options", row.Dhcpv4Options)...)
	r = append(r, types.OvsdbCmdArgsUuidOptional("dhcpv6_options", row.Dhcpv6Options)...)
	r = append(r, types.OvsdbCmdArgsStringOptional("dynamic_addresses", row.DynamicAddresses)...)
	r = append(r, types.OvsdbCmdArgsBooleanOptional("enabled", row.Enabled)...)
	r = append(r, types.OvsdbCmdArgsMapStringString("external_ids", row.ExternalIds)...)
	r = append(r, types.OvsdbCmdArgsString("name", row.Name)...)
	r = append(r, types.OvsdbCmdArgsMapStringString("options", row.Options)...)
	r = append(r, types.OvsdbCmdArgsStringOptional("parent_name", row.ParentName)...)
	r = append(r, types.OvsdbCmdArgsStringMultiples("port_security", row.PortSecurity)...)
	r = append(r, types.OvsdbCmdArgsIntegerOptional("tag", row.Tag)...)
	r = append(r, types.OvsdbCmdArgsIntegerOptional("tag_request", row.TagRequest)...)
	r = append(r, types.OvsdbCmdArgsString("type", row.Type)...)
	r = append(r, types.OvsdbCmdArgsBooleanOptional("up", row.Up)...)
	return r
}

func (row *LogicalSwitchPort) SetColumn(name string, val interface{}) (err error) {
	defer func() {
		if panicErr := recover(); panicErr != nil {
			err = errors.Wrapf(panicErr.(error), "%s: %#v", name, fmt.Sprintf("%#v", val))
		}
	}()
	switch name {
	case "_uuid":
		row.Uuid = types.EnsureUuid(val)
	case "_version":
		row.Version = types.EnsureUuid(val)
	case "addresses":
		row.Addresses = types.EnsureStringMultiples(val)
	case "dhcpv4_options":
		row.Dhcpv4Options = types.EnsureUuidOptional(val)
	case "dhcpv6_options":
		row.Dhcpv6Options = types.EnsureUuidOptional(val)
	case "dynamic_addresses":
		row.DynamicAddresses = types.EnsureStringOptional(val)
	case "enabled":
		row.Enabled = types.EnsureBooleanOptional(val)
	case "external_ids":
		row.ExternalIds = types.EnsureMapStringString(val)
	case "name":
		row.Name = types.EnsureString(val)
	case "options":
		row.Options = types.EnsureMapStringString(val)
	case "parent_name":
		row.ParentName = types.EnsureStringOptional(val)
	case "port_security":
		row.PortSecurity = types.EnsureStringMultiples(val)
	case "tag":
		row.Tag = types.EnsureIntegerOptional(val)
	case "tag_request":
		row.TagRequest = types.EnsureIntegerOptional(val)
	case "type":
		row.Type = types.EnsureString(val)
	case "up":
		row.Up = types.EnsureBooleanOptional(val)
	default:
		panic(types.ErrUnknownColumn)
	}
	return
}

func (row *LogicalSwitchPort) MatchNonZeros(row1 *LogicalSwitchPort) bool {
	if !types.MatchUuidIfNonZero(row.Uuid, row1.Uuid) {
		return false
	}
	if !types.MatchUuidIfNonZero(row.Version, row1.Version) {
		return false
	}
	if !types.MatchStringMultiplesIfNonZero(row.Addresses, row1.Addresses) {
		return false
	}
	if !types.MatchUuidOptionalIfNonZero(row.Dhcpv4Options, row1.Dhcpv4Options) {
		return false
	}
	if !types.MatchUuidOptionalIfNonZero(row.Dhcpv6Options, row1.Dhcpv6Options) {
		return false
	}
	if !types.MatchStringOptionalIfNonZero(row.DynamicAddresses, row1.DynamicAddresses) {
		return false
	}
	if !types.MatchBooleanOptionalIfNonZero(row.Enabled, row1.Enabled) {
		return false
	}
	if !types.MatchMapStringStringIfNonZero(row.ExternalIds, row1.ExternalIds) {
		return false
	}
	if !types.MatchStringIfNonZero(row.Name, row1.Name) {
		return false
	}
	if !types.MatchMapStringStringIfNonZero(row.Options, row1.Options) {
		return false
	}
	if !types.MatchStringOptionalIfNonZero(row.ParentName, row1.ParentName) {
		return false
	}
	if !types.MatchStringMultiplesIfNonZero(row.PortSecurity, row1.PortSecurity) {
		return false
	}
	if !types.MatchIntegerOptionalIfNonZero(row.Tag, row1.Tag) {
		return false
	}
	if !types.MatchIntegerOptionalIfNonZero(row.TagRequest, row1.TagRequest) {
		return false
	}
	if !types.MatchStringIfNonZero(row.Type, row1.Type) {
		return false
	}
	if !types.MatchBooleanOptionalIfNonZero(row.Up, row1.Up) {
		return false
	}
	return true
}

func (row *LogicalSwitchPort) HasExternalIds() bool {
	return true
}

func (row *LogicalSwitchPort) SetExternalId(k, v string) {
	if row.ExternalIds == nil {
		row.ExternalIds = map[string]string{}
	}
	row.ExternalIds[k] = v
}

func (row *LogicalSwitchPort) GetExternalId(k string) (string, bool) {
	if row.ExternalIds == nil {
		return "", false
	}
	r, ok := row.ExternalIds[k]
	return r, ok
}

func (row *LogicalSwitchPort) RemoveExternalId(k string) (string, bool) {
	if row.ExternalIds == nil {
		return "", false
	}
	r, ok := row.ExternalIds[k]
	if ok {
		delete(row.ExternalIds, k)
	}
	return r, ok
}

type NATTable []NAT

var _ types.ITable = &NATTable{}

func (tbl NATTable) OvsdbTableName() string {
	return "NAT"
}

func (tbl NATTable) OvsdbIsRoot() bool {
	return false
}

func (tbl NATTable) Rows() []types.IRow {
	r := make([]types.IRow, len(tbl))
	for i := range tbl {
		r[i] = &tbl[i]
	}
	return r
}

func (tbl NATTable) NewRow() types.IRow {
	return &NAT{}
}

func (tbl *NATTable) AppendRow(irow types.IRow) {
	row := irow.(*NAT)
	*tbl = append(*tbl, *row)
}

func (tbl NATTable) FindOneMatchNonZeros(row1 *NAT) *NAT {
	for i := range tbl {
		row := &tbl[i]
		if row.MatchNonZeros(row1) {
			return row
		}
	}
	return nil
}

type NAT struct {
	Uuid        string            `json:"_uuid"`
	Version     string            `json:"_version"`
	ExternalIds map[string]string `json:"external_ids"`
	ExternalIp  string            `json:"external_ip"`
	ExternalMac *string           `json:"external_mac"`
	LogicalIp   string            `json:"logical_ip"`
	LogicalPort *string           `json:"logical_port"`
	Type        string            `json:"type"`
}

var _ types.IRow = &NAT{}

func (row *NAT) OvsdbTableName() string {
	return "NAT"
}

func (row *NAT) OvsdbIsRoot() bool {
	return false
}

func (row *NAT) OvsdbUuid() string {
	return row.Uuid
}

func (row *NAT) OvsdbCmdArgs() []string {
	r := []string{}
	r = append(r, types.OvsdbCmdArgsMapStringString("external_ids", row.ExternalIds)...)
	r = append(r, types.OvsdbCmdArgsString("external_ip", row.ExternalIp)...)
	r = append(r, types.OvsdbCmdArgsStringOptional("external_mac", row.ExternalMac)...)
	r = append(r, types.OvsdbCmdArgsString("logical_ip", row.LogicalIp)...)
	r = append(r, types.OvsdbCmdArgsStringOptional("logical_port", row.LogicalPort)...)
	r = append(r, types.OvsdbCmdArgsString("type", row.Type)...)
	return r
}

func (row *NAT) SetColumn(name string, val interface{}) (err error) {
	defer func() {
		if panicErr := recover(); panicErr != nil {
			err = errors.Wrapf(panicErr.(error), "%s: %#v", name, fmt.Sprintf("%#v", val))
		}
	}()
	switch name {
	case "_uuid":
		row.Uuid = types.EnsureUuid(val)
	case "_version":
		row.Version = types.EnsureUuid(val)
	case "external_ids":
		row.ExternalIds = types.EnsureMapStringString(val)
	case "external_ip":
		row.ExternalIp = types.EnsureString(val)
	case "external_mac":
		row.ExternalMac = types.EnsureStringOptional(val)
	case "logical_ip":
		row.LogicalIp = types.EnsureString(val)
	case "logical_port":
		row.LogicalPort = types.EnsureStringOptional(val)
	case "type":
		row.Type = types.EnsureString(val)
	default:
		panic(types.ErrUnknownColumn)
	}
	return
}

func (row *NAT) MatchNonZeros(row1 *NAT) bool {
	if !types.MatchUuidIfNonZero(row.Uuid, row1.Uuid) {
		return false
	}
	if !types.MatchUuidIfNonZero(row.Version, row1.Version) {
		return false
	}
	if !types.MatchMapStringStringIfNonZero(row.ExternalIds, row1.ExternalIds) {
		return false
	}
	if !types.MatchStringIfNonZero(row.ExternalIp, row1.ExternalIp) {
		return false
	}
	if !types.MatchStringOptionalIfNonZero(row.ExternalMac, row1.ExternalMac) {
		return false
	}
	if !types.MatchStringIfNonZero(row.LogicalIp, row1.LogicalIp) {
		return false
	}
	if !types.MatchStringOptionalIfNonZero(row.LogicalPort, row1.LogicalPort) {
		return false
	}
	if !types.MatchStringIfNonZero(row.Type, row1.Type) {
		return false
	}
	return true
}

func (row *NAT) HasExternalIds() bool {
	return true
}

func (row *NAT) SetExternalId(k, v string) {
	if row.ExternalIds == nil {
		row.ExternalIds = map[string]string{}
	}
	row.ExternalIds[k] = v
}

func (row *NAT) GetExternalId(k string) (string, bool) {
	if row.ExternalIds == nil {
		return "", false
	}
	r, ok := row.ExternalIds[k]
	return r, ok
}

func (row *NAT) RemoveExternalId(k string) (string, bool) {
	if row.ExternalIds == nil {
		return "", false
	}
	r, ok := row.ExternalIds[k]
	if ok {
		delete(row.ExternalIds, k)
	}
	return r, ok
}

type NBGlobalTable []NBGlobal

var _ types.ITable = &NBGlobalTable{}

func (tbl NBGlobalTable) OvsdbTableName() string {
	return "NB_Global"
}

func (tbl NBGlobalTable) OvsdbIsRoot() bool {
	return true
}

func (tbl NBGlobalTable) Rows() []types.IRow {
	r := make([]types.IRow, len(tbl))
	for i := range tbl {
		r[i] = &tbl[i]
	}
	return r
}

func (tbl NBGlobalTable) NewRow() types.IRow {
	return &NBGlobal{}
}

func (tbl *NBGlobalTable) AppendRow(irow types.IRow) {
	row := irow.(*NBGlobal)
	*tbl = append(*tbl, *row)
}

func (tbl NBGlobalTable) FindOneMatchNonZeros(row1 *NBGlobal) *NBGlobal {
	for i := range tbl {
		row := &tbl[i]
		if row.MatchNonZeros(row1) {
			return row
		}
	}
	return nil
}

type NBGlobal struct {
	Uuid        string            `json:"_uuid"`
	Version     string            `json:"_version"`
	Connections []string          `json:"connections"`
	ExternalIds map[string]string `json:"external_ids"`
	HvCfg       int64             `json:"hv_cfg"`
	NbCfg       int64             `json:"nb_cfg"`
	SbCfg       int64             `json:"sb_cfg"`
	Ssl         *string           `json:"ssl"`
}

var _ types.IRow = &NBGlobal{}

func (row *NBGlobal) OvsdbTableName() string {
	return "NB_Global"
}

func (row *NBGlobal) OvsdbIsRoot() bool {
	return true
}

func (row *NBGlobal) OvsdbUuid() string {
	return row.Uuid
}

func (row *NBGlobal) OvsdbCmdArgs() []string {
	r := []string{}
	r = append(r, types.OvsdbCmdArgsUuidMultiples("connections", row.Connections)...)
	r = append(r, types.OvsdbCmdArgsMapStringString("external_ids", row.ExternalIds)...)
	r = append(r, types.OvsdbCmdArgsInteger("hv_cfg", row.HvCfg)...)
	r = append(r, types.OvsdbCmdArgsInteger("nb_cfg", row.NbCfg)...)
	r = append(r, types.OvsdbCmdArgsInteger("sb_cfg", row.SbCfg)...)
	r = append(r, types.OvsdbCmdArgsUuidOptional("ssl", row.Ssl)...)
	return r
}

func (row *NBGlobal) SetColumn(name string, val interface{}) (err error) {
	defer func() {
		if panicErr := recover(); panicErr != nil {
			err = errors.Wrapf(panicErr.(error), "%s: %#v", name, fmt.Sprintf("%#v", val))
		}
	}()
	switch name {
	case "_uuid":
		row.Uuid = types.EnsureUuid(val)
	case "_version":
		row.Version = types.EnsureUuid(val)
	case "connections":
		row.Connections = types.EnsureUuidMultiples(val)
	case "external_ids":
		row.ExternalIds = types.EnsureMapStringString(val)
	case "hv_cfg":
		row.HvCfg = types.EnsureInteger(val)
	case "nb_cfg":
		row.NbCfg = types.EnsureInteger(val)
	case "sb_cfg":
		row.SbCfg = types.EnsureInteger(val)
	case "ssl":
		row.Ssl = types.EnsureUuidOptional(val)
	default:
		panic(types.ErrUnknownColumn)
	}
	return
}

func (row *NBGlobal) MatchNonZeros(row1 *NBGlobal) bool {
	if !types.MatchUuidIfNonZero(row.Uuid, row1.Uuid) {
		return false
	}
	if !types.MatchUuidIfNonZero(row.Version, row1.Version) {
		return false
	}
	if !types.MatchUuidMultiplesIfNonZero(row.Connections, row1.Connections) {
		return false
	}
	if !types.MatchMapStringStringIfNonZero(row.ExternalIds, row1.ExternalIds) {
		return false
	}
	if !types.MatchIntegerIfNonZero(row.HvCfg, row1.HvCfg) {
		return false
	}
	if !types.MatchIntegerIfNonZero(row.NbCfg, row1.NbCfg) {
		return false
	}
	if !types.MatchIntegerIfNonZero(row.SbCfg, row1.SbCfg) {
		return false
	}
	if !types.MatchUuidOptionalIfNonZero(row.Ssl, row1.Ssl) {
		return false
	}
	return true
}

func (row *NBGlobal) HasExternalIds() bool {
	return true
}

func (row *NBGlobal) SetExternalId(k, v string) {
	if row.ExternalIds == nil {
		row.ExternalIds = map[string]string{}
	}
	row.ExternalIds[k] = v
}

func (row *NBGlobal) GetExternalId(k string) (string, bool) {
	if row.ExternalIds == nil {
		return "", false
	}
	r, ok := row.ExternalIds[k]
	return r, ok
}

func (row *NBGlobal) RemoveExternalId(k string) (string, bool) {
	if row.ExternalIds == nil {
		return "", false
	}
	r, ok := row.ExternalIds[k]
	if ok {
		delete(row.ExternalIds, k)
	}
	return r, ok
}

type QoSTable []QoS

var _ types.ITable = &QoSTable{}

func (tbl QoSTable) OvsdbTableName() string {
	return "QoS"
}

func (tbl QoSTable) OvsdbIsRoot() bool {
	return false
}

func (tbl QoSTable) Rows() []types.IRow {
	r := make([]types.IRow, len(tbl))
	for i := range tbl {
		r[i] = &tbl[i]
	}
	return r
}

func (tbl QoSTable) NewRow() types.IRow {
	return &QoS{}
}

func (tbl *QoSTable) AppendRow(irow types.IRow) {
	row := irow.(*QoS)
	*tbl = append(*tbl, *row)
}

func (tbl QoSTable) FindOneMatchNonZeros(row1 *QoS) *QoS {
	for i := range tbl {
		row := &tbl[i]
		if row.MatchNonZeros(row1) {
			return row
		}
	}
	return nil
}

type QoS struct {
	Uuid        string            `json:"_uuid"`
	Version     string            `json:"_version"`
	Action      map[string]int64  `json:"action"`
	Bandwidth   map[string]int64  `json:"bandwidth"`
	Direction   string            `json:"direction"`
	ExternalIds map[string]string `json:"external_ids"`
	Match       string            `json:"match"`
	Priority    int64             `json:"priority"`
}

var _ types.IRow = &QoS{}

func (row *QoS) OvsdbTableName() string {
	return "QoS"
}

func (row *QoS) OvsdbIsRoot() bool {
	return false
}

func (row *QoS) OvsdbUuid() string {
	return row.Uuid
}

func (row *QoS) OvsdbCmdArgs() []string {
	r := []string{}
	r = append(r, types.OvsdbCmdArgsMapStringInteger("action", row.Action)...)
	r = append(r, types.OvsdbCmdArgsMapStringInteger("bandwidth", row.Bandwidth)...)
	r = append(r, types.OvsdbCmdArgsString("direction", row.Direction)...)
	r = append(r, types.OvsdbCmdArgsMapStringString("external_ids", row.ExternalIds)...)
	r = append(r, types.OvsdbCmdArgsString("match", row.Match)...)
	r = append(r, types.OvsdbCmdArgsInteger("priority", row.Priority)...)
	return r
}

func (row *QoS) SetColumn(name string, val interface{}) (err error) {
	defer func() {
		if panicErr := recover(); panicErr != nil {
			err = errors.Wrapf(panicErr.(error), "%s: %#v", name, fmt.Sprintf("%#v", val))
		}
	}()
	switch name {
	case "_uuid":
		row.Uuid = types.EnsureUuid(val)
	case "_version":
		row.Version = types.EnsureUuid(val)
	case "action":
		row.Action = types.EnsureMapStringInteger(val)
	case "bandwidth":
		row.Bandwidth = types.EnsureMapStringInteger(val)
	case "direction":
		row.Direction = types.EnsureString(val)
	case "external_ids":
		row.ExternalIds = types.EnsureMapStringString(val)
	case "match":
		row.Match = types.EnsureString(val)
	case "priority":
		row.Priority = types.EnsureInteger(val)
	default:
		panic(types.ErrUnknownColumn)
	}
	return
}

func (row *QoS) MatchNonZeros(row1 *QoS) bool {
	if !types.MatchUuidIfNonZero(row.Uuid, row1.Uuid) {
		return false
	}
	if !types.MatchUuidIfNonZero(row.Version, row1.Version) {
		return false
	}
	if !types.MatchMapStringIntegerIfNonZero(row.Action, row1.Action) {
		return false
	}
	if !types.MatchMapStringIntegerIfNonZero(row.Bandwidth, row1.Bandwidth) {
		return false
	}
	if !types.MatchStringIfNonZero(row.Direction, row1.Direction) {
		return false
	}
	if !types.MatchMapStringStringIfNonZero(row.ExternalIds, row1.ExternalIds) {
		return false
	}
	if !types.MatchStringIfNonZero(row.Match, row1.Match) {
		return false
	}
	if !types.MatchIntegerIfNonZero(row.Priority, row1.Priority) {
		return false
	}
	return true
}

func (row *QoS) HasExternalIds() bool {
	return true
}

func (row *QoS) SetExternalId(k, v string) {
	if row.ExternalIds == nil {
		row.ExternalIds = map[string]string{}
	}
	row.ExternalIds[k] = v
}

func (row *QoS) GetExternalId(k string) (string, bool) {
	if row.ExternalIds == nil {
		return "", false
	}
	r, ok := row.ExternalIds[k]
	return r, ok
}

func (row *QoS) RemoveExternalId(k string) (string, bool) {
	if row.ExternalIds == nil {
		return "", false
	}
	r, ok := row.ExternalIds[k]
	if ok {
		delete(row.ExternalIds, k)
	}
	return r, ok
}

type SSLTable []SSL

var _ types.ITable = &SSLTable{}

func (tbl SSLTable) OvsdbTableName() string {
	return "SSL"
}

func (tbl SSLTable) OvsdbIsRoot() bool {
	return false
}

func (tbl SSLTable) Rows() []types.IRow {
	r := make([]types.IRow, len(tbl))
	for i := range tbl {
		r[i] = &tbl[i]
	}
	return r
}

func (tbl SSLTable) NewRow() types.IRow {
	return &SSL{}
}

func (tbl *SSLTable) AppendRow(irow types.IRow) {
	row := irow.(*SSL)
	*tbl = append(*tbl, *row)
}

func (tbl SSLTable) FindOneMatchNonZeros(row1 *SSL) *SSL {
	for i := range tbl {
		row := &tbl[i]
		if row.MatchNonZeros(row1) {
			return row
		}
	}
	return nil
}

type SSL struct {
	Uuid            string            `json:"_uuid"`
	Version         string            `json:"_version"`
	BootstrapCaCert bool              `json:"bootstrap_ca_cert"`
	CaCert          string            `json:"ca_cert"`
	Certificate     string            `json:"certificate"`
	ExternalIds     map[string]string `json:"external_ids"`
	PrivateKey      string            `json:"private_key"`
	SslCiphers      string            `json:"ssl_ciphers"`
	SslProtocols    string            `json:"ssl_protocols"`
}

var _ types.IRow = &SSL{}

func (row *SSL) OvsdbTableName() string {
	return "SSL"
}

func (row *SSL) OvsdbIsRoot() bool {
	return false
}

func (row *SSL) OvsdbUuid() string {
	return row.Uuid
}

func (row *SSL) OvsdbCmdArgs() []string {
	r := []string{}
	r = append(r, types.OvsdbCmdArgsBoolean("bootstrap_ca_cert", row.BootstrapCaCert)...)
	r = append(r, types.OvsdbCmdArgsString("ca_cert", row.CaCert)...)
	r = append(r, types.OvsdbCmdArgsString("certificate", row.Certificate)...)
	r = append(r, types.OvsdbCmdArgsMapStringString("external_ids", row.ExternalIds)...)
	r = append(r, types.OvsdbCmdArgsString("private_key", row.PrivateKey)...)
	r = append(r, types.OvsdbCmdArgsString("ssl_ciphers", row.SslCiphers)...)
	r = append(r, types.OvsdbCmdArgsString("ssl_protocols", row.SslProtocols)...)
	return r
}

func (row *SSL) SetColumn(name string, val interface{}) (err error) {
	defer func() {
		if panicErr := recover(); panicErr != nil {
			err = errors.Wrapf(panicErr.(error), "%s: %#v", name, fmt.Sprintf("%#v", val))
		}
	}()
	switch name {
	case "_uuid":
		row.Uuid = types.EnsureUuid(val)
	case "_version":
		row.Version = types.EnsureUuid(val)
	case "bootstrap_ca_cert":
		row.BootstrapCaCert = types.EnsureBoolean(val)
	case "ca_cert":
		row.CaCert = types.EnsureString(val)
	case "certificate":
		row.Certificate = types.EnsureString(val)
	case "external_ids":
		row.ExternalIds = types.EnsureMapStringString(val)
	case "private_key":
		row.PrivateKey = types.EnsureString(val)
	case "ssl_ciphers":
		row.SslCiphers = types.EnsureString(val)
	case "ssl_protocols":
		row.SslProtocols = types.EnsureString(val)
	default:
		panic(types.ErrUnknownColumn)
	}
	return
}

func (row *SSL) MatchNonZeros(row1 *SSL) bool {
	if !types.MatchUuidIfNonZero(row.Uuid, row1.Uuid) {
		return false
	}
	if !types.MatchUuidIfNonZero(row.Version, row1.Version) {
		return false
	}
	if !types.MatchBooleanIfNonZero(row.BootstrapCaCert, row1.BootstrapCaCert) {
		return false
	}
	if !types.MatchStringIfNonZero(row.CaCert, row1.CaCert) {
		return false
	}
	if !types.MatchStringIfNonZero(row.Certificate, row1.Certificate) {
		return false
	}
	if !types.MatchMapStringStringIfNonZero(row.ExternalIds, row1.ExternalIds) {
		return false
	}
	if !types.MatchStringIfNonZero(row.PrivateKey, row1.PrivateKey) {
		return false
	}
	if !types.MatchStringIfNonZero(row.SslCiphers, row1.SslCiphers) {
		return false
	}
	if !types.MatchStringIfNonZero(row.SslProtocols, row1.SslProtocols) {
		return false
	}
	return true
}

func (row *SSL) HasExternalIds() bool {
	return true
}

func (row *SSL) SetExternalId(k, v string) {
	if row.ExternalIds == nil {
		row.ExternalIds = map[string]string{}
	}
	row.ExternalIds[k] = v
}

func (row *SSL) GetExternalId(k string) (string, bool) {
	if row.ExternalIds == nil {
		return "", false
	}
	r, ok := row.ExternalIds[k]
	return r, ok
}

func (row *SSL) RemoveExternalId(k string) (string, bool) {
	if row.ExternalIds == nil {
		return "", false
	}
	r, ok := row.ExternalIds[k]
	if ok {
		delete(row.ExternalIds, k)
	}
	return r, ok
}
