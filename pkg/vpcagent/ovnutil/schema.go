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

package ovnutil

import (
	"fmt"

	"yunion.io/x/pkg/errors"
)

const (
	ErrUnknownColumn = errors.Error("unknown column")
	ErrUnknownTable  = errors.Error("unknown table")
)

type IRow interface {
	SetColumn(name string, val interface{}) error
	OvnTableName() string
	OvnIsRoot() bool
	OvnUuid() string
	OvnArgs() []string
	OvnSetExternalIds(k, v string)
	OvnGetExternalIds(k string) (string, bool)
	OvnRemoveExternalIds(k string) (string, bool)
}

type ITable interface {
	NewRow() IRow
	Rows() []IRow
	OvnTableName() string
	OvnIsRoot() bool
}

type OVNNorthbound struct {
	SSL                      SSLTable
	NBGlobal                 NBGlobalTable
	LogicalRouterStaticRoute LogicalRouterStaticRouteTable
	LoadBalancer             LoadBalancerTable
	Connection               ConnectionTable
	DNS                      DNSTable
	QoS                      QoSTable
	LogicalRouter            LogicalRouterTable
	LogicalSwitchPort        LogicalSwitchPortTable
	GatewayChassis           GatewayChassisTable
	LogicalRouterPort        LogicalRouterPortTable
	LogicalSwitch            LogicalSwitchTable
	NAT                      NATTable
	DHCPOptions              DHCPOptionsTable
	ACL                      ACLTable
	AddressSet               AddressSetTable
}

func (db *OVNNorthbound) FindOneMatchNonZeros(irow IRow) (r IRow) {
	switch row := irow.(type) {
	case *GatewayChassis:
		if r := db.GatewayChassis.FindOneMatchNonZeros(row); r != nil {
			return r
		}
		return nil
	case *LogicalRouterPort:
		if r := db.LogicalRouterPort.FindOneMatchNonZeros(row); r != nil {
			return r
		}
		return nil
	case *LogicalSwitch:
		if r := db.LogicalSwitch.FindOneMatchNonZeros(row); r != nil {
			return r
		}
		return nil
	case *NAT:
		if r := db.NAT.FindOneMatchNonZeros(row); r != nil {
			return r
		}
		return nil
	case *DHCPOptions:
		if r := db.DHCPOptions.FindOneMatchNonZeros(row); r != nil {
			return r
		}
		return nil
	case *ACL:
		if r := db.ACL.FindOneMatchNonZeros(row); r != nil {
			return r
		}
		return nil
	case *AddressSet:
		if r := db.AddressSet.FindOneMatchNonZeros(row); r != nil {
			return r
		}
		return nil
	case *SSL:
		if r := db.SSL.FindOneMatchNonZeros(row); r != nil {
			return r
		}
		return nil
	case *NBGlobal:
		if r := db.NBGlobal.FindOneMatchNonZeros(row); r != nil {
			return r
		}
		return nil
	case *LogicalRouterStaticRoute:
		if r := db.LogicalRouterStaticRoute.FindOneMatchNonZeros(row); r != nil {
			return r
		}
		return nil
	case *LoadBalancer:
		if r := db.LoadBalancer.FindOneMatchNonZeros(row); r != nil {
			return r
		}
		return nil
	case *Connection:
		if r := db.Connection.FindOneMatchNonZeros(row); r != nil {
			return r
		}
		return nil
	case *DNS:
		if r := db.DNS.FindOneMatchNonZeros(row); r != nil {
			return r
		}
		return nil
	case *QoS:
		if r := db.QoS.FindOneMatchNonZeros(row); r != nil {
			return r
		}
		return nil
	case *LogicalRouter:
		if r := db.LogicalRouter.FindOneMatchNonZeros(row); r != nil {
			return r
		}
		return nil
	case *LogicalSwitchPort:
		if r := db.LogicalSwitchPort.FindOneMatchNonZeros(row); r != nil {
			return r
		}
		return nil
	}
	panic(ErrUnknownTable)
}

type LogicalRouter struct {
	Uuid         string            `json:"-"`
	Enabled      *bool             `json:"enabled"`
	Nat          []string          `json:"nat"`
	LoadBalancer []string          `json:"load_balancer"`
	Options      map[string]string `json:"options"`
	ExternalIds  map[string]string `json:"external_ids"`
	Name         string            `json:"name"`
	Ports        []string          `json:"ports"`
	StaticRoutes []string          `json:"static_routes"`
}

func (row *LogicalRouter) OvnTableName() string {
	return "Logical_Router"
}

func (row *LogicalRouter) OvnIsRoot() bool {
	return true
}

func (row *LogicalRouter) OvnUuid() string {
	return row.Uuid
}

func (row *LogicalRouter) OvnSetExternalIds(k, v string) {
	if row.ExternalIds == nil {
		row.ExternalIds = map[string]string{}
	}
	row.ExternalIds[k] = v
}

func (row *LogicalRouter) OvnGetExternalIds(k string) (string, bool) {
	if row.ExternalIds == nil {
		return "", false
	}
	r, ok := row.ExternalIds[k]
	return r, ok
}

func (row *LogicalRouter) OvnRemoveExternalIds(k string) (string, bool) {
	if row.ExternalIds == nil {
		return "", false
	}
	r, ok := row.ExternalIds[k]
	if ok {
		delete(row.ExternalIds, k)
	}
	return r, ok
}

func (row *LogicalRouter) SetColumn(name string, val interface{}) (err error) {
	defer func() {
		if panicErr := recover(); panicErr != nil {
			err = errors.Wrapf(panicErr.(error), "%s: %#v", name, fmt.Sprintf("%#v", val))
		}
	}()
	switch name {
	case "_uuid":
		row.Uuid = ensureUuid(val)
	case "nat":
		row.Nat = ensureUuidMultiples(val)
	case "load_balancer":
		row.LoadBalancer = ensureUuidMultiples(val)
	case "options":
		row.Options = ensureMapStringString(val)
	case "external_ids":
		row.ExternalIds = ensureMapStringString(val)
	case "name":
		row.Name = ensureString(val)
	case "ports":
		row.Ports = ensureUuidMultiples(val)
	case "static_routes":
		row.StaticRoutes = ensureUuidMultiples(val)
	case "enabled":
		row.Enabled = ensureBooleanOptional(val)
	default:
		panic(ErrUnknownColumn)
	}
	return
}

func (row *LogicalRouter) MatchNonZeros(row1 *LogicalRouter) bool {
	if !matchBooleanOptionalIfNonZero(row.Enabled, row1.Enabled) {
		return false
	}
	if !matchUuidMultiplesIfNonZero(row.Nat, row1.Nat) {
		return false
	}
	if !matchUuidMultiplesIfNonZero(row.LoadBalancer, row1.LoadBalancer) {
		return false
	}
	if !matchMapStringStringIfNonZero(row.Options, row1.Options) {
		return false
	}
	if !matchMapStringStringIfNonZero(row.ExternalIds, row1.ExternalIds) {
		return false
	}
	if !matchStringIfNonZero(row.Name, row1.Name) {
		return false
	}
	if !matchUuidMultiplesIfNonZero(row.Ports, row1.Ports) {
		return false
	}
	if !matchUuidMultiplesIfNonZero(row.StaticRoutes, row1.StaticRoutes) {
		return false
	}
	return true
}

func (row *LogicalRouter) OvnArgs() []string {
	r := []string{}
	r = append(r, OvnArgsUuidMultiples("nat", row.Nat)...)
	r = append(r, OvnArgsUuidMultiples("load_balancer", row.LoadBalancer)...)
	r = append(r, OvnArgsMapStringString("options", row.Options)...)
	r = append(r, OvnArgsMapStringString("external_ids", row.ExternalIds)...)
	r = append(r, OvnArgsString("name", row.Name)...)
	r = append(r, OvnArgsUuidMultiples("ports", row.Ports)...)
	r = append(r, OvnArgsUuidMultiples("static_routes", row.StaticRoutes)...)
	r = append(r, OvnArgsBooleanOptional("enabled", row.Enabled)...)
	return r
}

type LogicalRouterTable []LogicalRouter

func (tbl *LogicalRouterTable) NewRow() IRow {
	*tbl = append(*tbl, LogicalRouter{})
	return &(*tbl)[len(*tbl)-1]
}

func (tbl LogicalRouterTable) OvnTableName() string {
	return "Logical_Router"
}

func (tbl LogicalRouterTable) OvnIsRoot() bool {
	return true
}

func (tbl LogicalRouterTable) Rows() []IRow {
	r := make([]IRow, len(tbl))
	for i := range tbl {
		r[i] = &tbl[i]
	}
	return r
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

type LogicalSwitchPort struct {
	Uuid             string            `json:"-"`
	PortSecurity     []string          `json:"port_security"`
	Type             string            `json:"type"`
	Dhcpv4Options    *string           `json:"dhcpv4_options"`
	Options          map[string]string `json:"options"`
	Dhcpv6Options    *string           `json:"dhcpv6_options"`
	ParentName       *string           `json:"parent_name"`
	Enabled          *bool             `json:"enabled"`
	ExternalIds      map[string]string `json:"external_ids"`
	TagRequest       *int64            `json:"tag_request"`
	Addresses        []string          `json:"addresses"`
	DynamicAddresses *string           `json:"dynamic_addresses"`
	Name             string            `json:"name"`
	Tag              *int64            `json:"tag"`
	Up               *bool             `json:"up"`
}

func (row *LogicalSwitchPort) OvnTableName() string {
	return "Logical_Switch_Port"
}

func (row *LogicalSwitchPort) OvnIsRoot() bool {
	return false
}

func (row *LogicalSwitchPort) OvnUuid() string {
	return row.Uuid
}

func (row *LogicalSwitchPort) OvnSetExternalIds(k, v string) {
	if row.ExternalIds == nil {
		row.ExternalIds = map[string]string{}
	}
	row.ExternalIds[k] = v
}

func (row *LogicalSwitchPort) OvnGetExternalIds(k string) (string, bool) {
	if row.ExternalIds == nil {
		return "", false
	}
	r, ok := row.ExternalIds[k]
	return r, ok
}

func (row *LogicalSwitchPort) OvnRemoveExternalIds(k string) (string, bool) {
	if row.ExternalIds == nil {
		return "", false
	}
	r, ok := row.ExternalIds[k]
	if ok {
		delete(row.ExternalIds, k)
	}
	return r, ok
}

func (row *LogicalSwitchPort) SetColumn(name string, val interface{}) (err error) {
	defer func() {
		if panicErr := recover(); panicErr != nil {
			err = errors.Wrapf(panicErr.(error), "%s: %#v", name, fmt.Sprintf("%#v", val))
		}
	}()
	switch name {
	case "_uuid":
		row.Uuid = ensureUuid(val)
	case "port_security":
		row.PortSecurity = ensureStringMultiples(val)
	case "type":
		row.Type = ensureString(val)
	case "dhcpv4_options":
		row.Dhcpv4Options = ensureUuidOptional(val)
	case "options":
		row.Options = ensureMapStringString(val)
	case "dhcpv6_options":
		row.Dhcpv6Options = ensureUuidOptional(val)
	case "parent_name":
		row.ParentName = ensureStringOptional(val)
	case "enabled":
		row.Enabled = ensureBooleanOptional(val)
	case "name":
		row.Name = ensureString(val)
	case "tag":
		row.Tag = ensureIntegerOptional(val)
	case "up":
		row.Up = ensureBooleanOptional(val)
	case "external_ids":
		row.ExternalIds = ensureMapStringString(val)
	case "tag_request":
		row.TagRequest = ensureIntegerOptional(val)
	case "addresses":
		row.Addresses = ensureStringMultiples(val)
	case "dynamic_addresses":
		row.DynamicAddresses = ensureStringOptional(val)
	default:
		panic(ErrUnknownColumn)
	}
	return
}

func (row *LogicalSwitchPort) MatchNonZeros(row1 *LogicalSwitchPort) bool {
	if !matchStringMultiplesIfNonZero(row.Addresses, row1.Addresses) {
		return false
	}
	if !matchStringOptionalIfNonZero(row.DynamicAddresses, row1.DynamicAddresses) {
		return false
	}
	if !matchStringIfNonZero(row.Name, row1.Name) {
		return false
	}
	if !matchIntegerOptionalIfNonZero(row.Tag, row1.Tag) {
		return false
	}
	if !matchBooleanOptionalIfNonZero(row.Up, row1.Up) {
		return false
	}
	if !matchMapStringStringIfNonZero(row.ExternalIds, row1.ExternalIds) {
		return false
	}
	if !matchIntegerOptionalIfNonZero(row.TagRequest, row1.TagRequest) {
		return false
	}
	if !matchUuidOptionalIfNonZero(row.Dhcpv4Options, row1.Dhcpv4Options) {
		return false
	}
	if !matchStringMultiplesIfNonZero(row.PortSecurity, row1.PortSecurity) {
		return false
	}
	if !matchStringIfNonZero(row.Type, row1.Type) {
		return false
	}
	if !matchStringOptionalIfNonZero(row.ParentName, row1.ParentName) {
		return false
	}
	if !matchBooleanOptionalIfNonZero(row.Enabled, row1.Enabled) {
		return false
	}
	if !matchMapStringStringIfNonZero(row.Options, row1.Options) {
		return false
	}
	if !matchUuidOptionalIfNonZero(row.Dhcpv6Options, row1.Dhcpv6Options) {
		return false
	}
	return true
}

func (row *LogicalSwitchPort) OvnArgs() []string {
	r := []string{}
	r = append(r, OvnArgsStringMultiples("port_security", row.PortSecurity)...)
	r = append(r, OvnArgsString("type", row.Type)...)
	r = append(r, OvnArgsUuidOptional("dhcpv4_options", row.Dhcpv4Options)...)
	r = append(r, OvnArgsMapStringString("options", row.Options)...)
	r = append(r, OvnArgsUuidOptional("dhcpv6_options", row.Dhcpv6Options)...)
	r = append(r, OvnArgsStringOptional("parent_name", row.ParentName)...)
	r = append(r, OvnArgsBooleanOptional("enabled", row.Enabled)...)
	r = append(r, OvnArgsIntegerOptional("tag", row.Tag)...)
	r = append(r, OvnArgsBooleanOptional("up", row.Up)...)
	r = append(r, OvnArgsMapStringString("external_ids", row.ExternalIds)...)
	r = append(r, OvnArgsIntegerOptional("tag_request", row.TagRequest)...)
	r = append(r, OvnArgsStringMultiples("addresses", row.Addresses)...)
	r = append(r, OvnArgsStringOptional("dynamic_addresses", row.DynamicAddresses)...)
	r = append(r, OvnArgsString("name", row.Name)...)
	return r
}

type LogicalSwitchPortTable []LogicalSwitchPort

func (tbl *LogicalSwitchPortTable) NewRow() IRow {
	*tbl = append(*tbl, LogicalSwitchPort{})
	return &(*tbl)[len(*tbl)-1]
}

func (tbl LogicalSwitchPortTable) OvnTableName() string {
	return "Logical_Switch_Port"
}

func (tbl LogicalSwitchPortTable) OvnIsRoot() bool {
	return false
}

func (tbl LogicalSwitchPortTable) Rows() []IRow {
	r := make([]IRow, len(tbl))
	for i := range tbl {
		r[i] = &tbl[i]
	}
	return r
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

type Connection struct {
	Uuid            string            `json:"-"`
	Status          map[string]string `json:"status"`
	Target          string            `json:"target"`
	MaxBackoff      *int64            `json:"max_backoff"`
	InactivityProbe *int64            `json:"inactivity_probe"`
	OtherConfig     map[string]string `json:"other_config"`
	ExternalIds     map[string]string `json:"external_ids"`
	IsConnected     bool              `json:"is_connected"`
}

func (row *Connection) OvnTableName() string {
	return "Connection"
}

func (row *Connection) OvnIsRoot() bool {
	return false
}

func (row *Connection) OvnUuid() string {
	return row.Uuid
}

func (row *Connection) OvnSetExternalIds(k, v string) {
	if row.ExternalIds == nil {
		row.ExternalIds = map[string]string{}
	}
	row.ExternalIds[k] = v
}

func (row *Connection) OvnGetExternalIds(k string) (string, bool) {
	if row.ExternalIds == nil {
		return "", false
	}
	r, ok := row.ExternalIds[k]
	return r, ok
}

func (row *Connection) OvnRemoveExternalIds(k string) (string, bool) {
	if row.ExternalIds == nil {
		return "", false
	}
	r, ok := row.ExternalIds[k]
	if ok {
		delete(row.ExternalIds, k)
	}
	return r, ok
}

func (row *Connection) SetColumn(name string, val interface{}) (err error) {
	defer func() {
		if panicErr := recover(); panicErr != nil {
			err = errors.Wrapf(panicErr.(error), "%s: %#v", name, fmt.Sprintf("%#v", val))
		}
	}()
	switch name {
	case "_uuid":
		row.Uuid = ensureUuid(val)
	case "target":
		row.Target = ensureString(val)
	case "max_backoff":
		row.MaxBackoff = ensureIntegerOptional(val)
	case "inactivity_probe":
		row.InactivityProbe = ensureIntegerOptional(val)
	case "other_config":
		row.OtherConfig = ensureMapStringString(val)
	case "external_ids":
		row.ExternalIds = ensureMapStringString(val)
	case "is_connected":
		row.IsConnected = ensureBoolean(val)
	case "status":
		row.Status = ensureMapStringString(val)
	default:
		panic(ErrUnknownColumn)
	}
	return
}

func (row *Connection) MatchNonZeros(row1 *Connection) bool {
	if !matchMapStringStringIfNonZero(row.ExternalIds, row1.ExternalIds) {
		return false
	}
	if !matchBooleanIfNonZero(row.IsConnected, row1.IsConnected) {
		return false
	}
	if !matchMapStringStringIfNonZero(row.Status, row1.Status) {
		return false
	}
	if !matchStringIfNonZero(row.Target, row1.Target) {
		return false
	}
	if !matchIntegerOptionalIfNonZero(row.MaxBackoff, row1.MaxBackoff) {
		return false
	}
	if !matchIntegerOptionalIfNonZero(row.InactivityProbe, row1.InactivityProbe) {
		return false
	}
	if !matchMapStringStringIfNonZero(row.OtherConfig, row1.OtherConfig) {
		return false
	}
	return true
}

func (row *Connection) OvnArgs() []string {
	r := []string{}
	r = append(r, OvnArgsBoolean("is_connected", row.IsConnected)...)
	r = append(r, OvnArgsMapStringString("status", row.Status)...)
	r = append(r, OvnArgsString("target", row.Target)...)
	r = append(r, OvnArgsIntegerOptional("max_backoff", row.MaxBackoff)...)
	r = append(r, OvnArgsIntegerOptional("inactivity_probe", row.InactivityProbe)...)
	r = append(r, OvnArgsMapStringString("other_config", row.OtherConfig)...)
	r = append(r, OvnArgsMapStringString("external_ids", row.ExternalIds)...)
	return r
}

type ConnectionTable []Connection

func (tbl *ConnectionTable) NewRow() IRow {
	*tbl = append(*tbl, Connection{})
	return &(*tbl)[len(*tbl)-1]
}

func (tbl ConnectionTable) OvnTableName() string {
	return "Connection"
}

func (tbl ConnectionTable) OvnIsRoot() bool {
	return false
}

func (tbl ConnectionTable) Rows() []IRow {
	r := make([]IRow, len(tbl))
	for i := range tbl {
		r[i] = &tbl[i]
	}
	return r
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

type DNS struct {
	Uuid        string            `json:"-"`
	Records     map[string]string `json:"records"`
	ExternalIds map[string]string `json:"external_ids"`
}

func (row *DNS) OvnTableName() string {
	return "DNS"
}

func (row *DNS) OvnIsRoot() bool {
	return true
}

func (row *DNS) OvnUuid() string {
	return row.Uuid
}

func (row *DNS) OvnSetExternalIds(k, v string) {
	if row.ExternalIds == nil {
		row.ExternalIds = map[string]string{}
	}
	row.ExternalIds[k] = v
}

func (row *DNS) OvnGetExternalIds(k string) (string, bool) {
	if row.ExternalIds == nil {
		return "", false
	}
	r, ok := row.ExternalIds[k]
	return r, ok
}

func (row *DNS) OvnRemoveExternalIds(k string) (string, bool) {
	if row.ExternalIds == nil {
		return "", false
	}
	r, ok := row.ExternalIds[k]
	if ok {
		delete(row.ExternalIds, k)
	}
	return r, ok
}

func (row *DNS) SetColumn(name string, val interface{}) (err error) {
	defer func() {
		if panicErr := recover(); panicErr != nil {
			err = errors.Wrapf(panicErr.(error), "%s: %#v", name, fmt.Sprintf("%#v", val))
		}
	}()
	switch name {
	case "_uuid":
		row.Uuid = ensureUuid(val)
	case "records":
		row.Records = ensureMapStringString(val)
	case "external_ids":
		row.ExternalIds = ensureMapStringString(val)
	default:
		panic(ErrUnknownColumn)
	}
	return
}

func (row *DNS) MatchNonZeros(row1 *DNS) bool {
	if !matchMapStringStringIfNonZero(row.Records, row1.Records) {
		return false
	}
	if !matchMapStringStringIfNonZero(row.ExternalIds, row1.ExternalIds) {
		return false
	}
	return true
}

func (row *DNS) OvnArgs() []string {
	r := []string{}
	r = append(r, OvnArgsMapStringString("records", row.Records)...)
	r = append(r, OvnArgsMapStringString("external_ids", row.ExternalIds)...)
	return r
}

type DNSTable []DNS

func (tbl *DNSTable) NewRow() IRow {
	*tbl = append(*tbl, DNS{})
	return &(*tbl)[len(*tbl)-1]
}

func (tbl DNSTable) OvnTableName() string {
	return "DNS"
}

func (tbl DNSTable) OvnIsRoot() bool {
	return true
}

func (tbl DNSTable) Rows() []IRow {
	r := make([]IRow, len(tbl))
	for i := range tbl {
		r[i] = &tbl[i]
	}
	return r
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

type QoS struct {
	Uuid        string            `json:"-"`
	Match       string            `json:"match"`
	Action      map[string]int64  `json:"action"`
	Bandwidth   map[string]int64  `json:"bandwidth"`
	ExternalIds map[string]string `json:"external_ids"`
	Priority    int64             `json:"priority"`
	Direction   string            `json:"direction"`
}

func (row *QoS) OvnTableName() string {
	return "QoS"
}

func (row *QoS) OvnIsRoot() bool {
	return false
}

func (row *QoS) OvnUuid() string {
	return row.Uuid
}

func (row *QoS) OvnSetExternalIds(k, v string) {
	if row.ExternalIds == nil {
		row.ExternalIds = map[string]string{}
	}
	row.ExternalIds[k] = v
}

func (row *QoS) OvnGetExternalIds(k string) (string, bool) {
	if row.ExternalIds == nil {
		return "", false
	}
	r, ok := row.ExternalIds[k]
	return r, ok
}

func (row *QoS) OvnRemoveExternalIds(k string) (string, bool) {
	if row.ExternalIds == nil {
		return "", false
	}
	r, ok := row.ExternalIds[k]
	if ok {
		delete(row.ExternalIds, k)
	}
	return r, ok
}

func (row *QoS) SetColumn(name string, val interface{}) (err error) {
	defer func() {
		if panicErr := recover(); panicErr != nil {
			err = errors.Wrapf(panicErr.(error), "%s: %#v", name, fmt.Sprintf("%#v", val))
		}
	}()
	switch name {
	case "_uuid":
		row.Uuid = ensureUuid(val)
	case "priority":
		row.Priority = ensureInteger(val)
	case "direction":
		row.Direction = ensureString(val)
	case "match":
		row.Match = ensureString(val)
	case "action":
		row.Action = ensureMapStringInteger(val)
	case "bandwidth":
		row.Bandwidth = ensureMapStringInteger(val)
	case "external_ids":
		row.ExternalIds = ensureMapStringString(val)
	default:
		panic(ErrUnknownColumn)
	}
	return
}

func (row *QoS) MatchNonZeros(row1 *QoS) bool {
	if !matchStringIfNonZero(row.Match, row1.Match) {
		return false
	}
	if !matchMapStringIntegerIfNonZero(row.Action, row1.Action) {
		return false
	}
	if !matchMapStringIntegerIfNonZero(row.Bandwidth, row1.Bandwidth) {
		return false
	}
	if !matchMapStringStringIfNonZero(row.ExternalIds, row1.ExternalIds) {
		return false
	}
	if !matchIntegerIfNonZero(row.Priority, row1.Priority) {
		return false
	}
	if !matchStringIfNonZero(row.Direction, row1.Direction) {
		return false
	}
	return true
}

func (row *QoS) OvnArgs() []string {
	r := []string{}
	r = append(r, OvnArgsInteger("priority", row.Priority)...)
	r = append(r, OvnArgsString("direction", row.Direction)...)
	r = append(r, OvnArgsString("match", row.Match)...)
	r = append(r, OvnArgsMapStringInteger("action", row.Action)...)
	r = append(r, OvnArgsMapStringInteger("bandwidth", row.Bandwidth)...)
	r = append(r, OvnArgsMapStringString("external_ids", row.ExternalIds)...)
	return r
}

type QoSTable []QoS

func (tbl *QoSTable) NewRow() IRow {
	*tbl = append(*tbl, QoS{})
	return &(*tbl)[len(*tbl)-1]
}

func (tbl QoSTable) OvnTableName() string {
	return "QoS"
}

func (tbl QoSTable) OvnIsRoot() bool {
	return false
}

func (tbl QoSTable) Rows() []IRow {
	r := make([]IRow, len(tbl))
	for i := range tbl {
		r[i] = &tbl[i]
	}
	return r
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

type GatewayChassis struct {
	Uuid        string            `json:"-"`
	Name        string            `json:"name"`
	ChassisName string            `json:"chassis_name"`
	Priority    int64             `json:"priority"`
	ExternalIds map[string]string `json:"external_ids"`
	Options     map[string]string `json:"options"`
}

func (row *GatewayChassis) OvnTableName() string {
	return "Gateway_Chassis"
}

func (row *GatewayChassis) OvnIsRoot() bool {
	return false
}

func (row *GatewayChassis) OvnUuid() string {
	return row.Uuid
}

func (row *GatewayChassis) OvnSetExternalIds(k, v string) {
	if row.ExternalIds == nil {
		row.ExternalIds = map[string]string{}
	}
	row.ExternalIds[k] = v
}

func (row *GatewayChassis) OvnGetExternalIds(k string) (string, bool) {
	if row.ExternalIds == nil {
		return "", false
	}
	r, ok := row.ExternalIds[k]
	return r, ok
}

func (row *GatewayChassis) OvnRemoveExternalIds(k string) (string, bool) {
	if row.ExternalIds == nil {
		return "", false
	}
	r, ok := row.ExternalIds[k]
	if ok {
		delete(row.ExternalIds, k)
	}
	return r, ok
}

func (row *GatewayChassis) SetColumn(name string, val interface{}) (err error) {
	defer func() {
		if panicErr := recover(); panicErr != nil {
			err = errors.Wrapf(panicErr.(error), "%s: %#v", name, fmt.Sprintf("%#v", val))
		}
	}()
	switch name {
	case "_uuid":
		row.Uuid = ensureUuid(val)
	case "name":
		row.Name = ensureString(val)
	case "chassis_name":
		row.ChassisName = ensureString(val)
	case "priority":
		row.Priority = ensureInteger(val)
	case "external_ids":
		row.ExternalIds = ensureMapStringString(val)
	case "options":
		row.Options = ensureMapStringString(val)
	default:
		panic(ErrUnknownColumn)
	}
	return
}

func (row *GatewayChassis) MatchNonZeros(row1 *GatewayChassis) bool {
	if !matchStringIfNonZero(row.ChassisName, row1.ChassisName) {
		return false
	}
	if !matchIntegerIfNonZero(row.Priority, row1.Priority) {
		return false
	}
	if !matchMapStringStringIfNonZero(row.ExternalIds, row1.ExternalIds) {
		return false
	}
	if !matchMapStringStringIfNonZero(row.Options, row1.Options) {
		return false
	}
	if !matchStringIfNonZero(row.Name, row1.Name) {
		return false
	}
	return true
}

func (row *GatewayChassis) OvnArgs() []string {
	r := []string{}
	r = append(r, OvnArgsString("name", row.Name)...)
	r = append(r, OvnArgsString("chassis_name", row.ChassisName)...)
	r = append(r, OvnArgsInteger("priority", row.Priority)...)
	r = append(r, OvnArgsMapStringString("external_ids", row.ExternalIds)...)
	r = append(r, OvnArgsMapStringString("options", row.Options)...)
	return r
}

type GatewayChassisTable []GatewayChassis

func (tbl *GatewayChassisTable) NewRow() IRow {
	*tbl = append(*tbl, GatewayChassis{})
	return &(*tbl)[len(*tbl)-1]
}

func (tbl GatewayChassisTable) OvnTableName() string {
	return "Gateway_Chassis"
}

func (tbl GatewayChassisTable) OvnIsRoot() bool {
	return false
}

func (tbl GatewayChassisTable) Rows() []IRow {
	r := make([]IRow, len(tbl))
	for i := range tbl {
		r[i] = &tbl[i]
	}
	return r
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

type LogicalRouterPort struct {
	Uuid           string            `json:"-"`
	Networks       []string          `json:"networks"`
	ExternalIds    map[string]string `json:"external_ids"`
	Enabled        *bool             `json:"enabled"`
	Peer           *string           `json:"peer"`
	Ipv6RaConfigs  map[string]string `json:"ipv6_ra_configs"`
	GatewayChassis []string          `json:"gateway_chassis"`
	Mac            string            `json:"mac"`
	Name           string            `json:"name"`
	Options        map[string]string `json:"options"`
}

func (row *LogicalRouterPort) OvnTableName() string {
	return "Logical_Router_Port"
}

func (row *LogicalRouterPort) OvnIsRoot() bool {
	return false
}

func (row *LogicalRouterPort) OvnUuid() string {
	return row.Uuid
}

func (row *LogicalRouterPort) OvnSetExternalIds(k, v string) {
	if row.ExternalIds == nil {
		row.ExternalIds = map[string]string{}
	}
	row.ExternalIds[k] = v
}

func (row *LogicalRouterPort) OvnGetExternalIds(k string) (string, bool) {
	if row.ExternalIds == nil {
		return "", false
	}
	r, ok := row.ExternalIds[k]
	return r, ok
}

func (row *LogicalRouterPort) OvnRemoveExternalIds(k string) (string, bool) {
	if row.ExternalIds == nil {
		return "", false
	}
	r, ok := row.ExternalIds[k]
	if ok {
		delete(row.ExternalIds, k)
	}
	return r, ok
}

func (row *LogicalRouterPort) SetColumn(name string, val interface{}) (err error) {
	defer func() {
		if panicErr := recover(); panicErr != nil {
			err = errors.Wrapf(panicErr.(error), "%s: %#v", name, fmt.Sprintf("%#v", val))
		}
	}()
	switch name {
	case "_uuid":
		row.Uuid = ensureUuid(val)
	case "peer":
		row.Peer = ensureStringOptional(val)
	case "ipv6_ra_configs":
		row.Ipv6RaConfigs = ensureMapStringString(val)
	case "gateway_chassis":
		row.GatewayChassis = ensureUuidMultiples(val)
	case "mac":
		row.Mac = ensureString(val)
	case "name":
		row.Name = ensureString(val)
	case "options":
		row.Options = ensureMapStringString(val)
	case "networks":
		row.Networks = ensureStringMultiples(val)
	case "external_ids":
		row.ExternalIds = ensureMapStringString(val)
	case "enabled":
		row.Enabled = ensureBooleanOptional(val)
	default:
		panic(ErrUnknownColumn)
	}
	return
}

func (row *LogicalRouterPort) MatchNonZeros(row1 *LogicalRouterPort) bool {
	if !matchMapStringStringIfNonZero(row.ExternalIds, row1.ExternalIds) {
		return false
	}
	if !matchBooleanOptionalIfNonZero(row.Enabled, row1.Enabled) {
		return false
	}
	if !matchStringMultiplesIfNonZero(row.Networks, row1.Networks) {
		return false
	}
	if !matchMapStringStringIfNonZero(row.Ipv6RaConfigs, row1.Ipv6RaConfigs) {
		return false
	}
	if !matchUuidMultiplesIfNonZero(row.GatewayChassis, row1.GatewayChassis) {
		return false
	}
	if !matchStringIfNonZero(row.Mac, row1.Mac) {
		return false
	}
	if !matchStringIfNonZero(row.Name, row1.Name) {
		return false
	}
	if !matchMapStringStringIfNonZero(row.Options, row1.Options) {
		return false
	}
	if !matchStringOptionalIfNonZero(row.Peer, row1.Peer) {
		return false
	}
	return true
}

func (row *LogicalRouterPort) OvnArgs() []string {
	r := []string{}
	r = append(r, OvnArgsString("mac", row.Mac)...)
	r = append(r, OvnArgsString("name", row.Name)...)
	r = append(r, OvnArgsMapStringString("options", row.Options)...)
	r = append(r, OvnArgsStringOptional("peer", row.Peer)...)
	r = append(r, OvnArgsMapStringString("ipv6_ra_configs", row.Ipv6RaConfigs)...)
	r = append(r, OvnArgsUuidMultiples("gateway_chassis", row.GatewayChassis)...)
	r = append(r, OvnArgsMapStringString("external_ids", row.ExternalIds)...)
	r = append(r, OvnArgsBooleanOptional("enabled", row.Enabled)...)
	r = append(r, OvnArgsStringMultiples("networks", row.Networks)...)
	return r
}

type LogicalRouterPortTable []LogicalRouterPort

func (tbl *LogicalRouterPortTable) NewRow() IRow {
	*tbl = append(*tbl, LogicalRouterPort{})
	return &(*tbl)[len(*tbl)-1]
}

func (tbl LogicalRouterPortTable) OvnTableName() string {
	return "Logical_Router_Port"
}

func (tbl LogicalRouterPortTable) OvnIsRoot() bool {
	return false
}

func (tbl LogicalRouterPortTable) Rows() []IRow {
	r := make([]IRow, len(tbl))
	for i := range tbl {
		r[i] = &tbl[i]
	}
	return r
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

type ACL struct {
	Uuid        string            `json:"-"`
	Direction   string            `json:"direction"`
	Match       string            `json:"match"`
	Action      string            `json:"action"`
	Log         bool              `json:"log"`
	Severity    *string           `json:"severity"`
	ExternalIds map[string]string `json:"external_ids"`
	Name        *string           `json:"name"`
	Priority    int64             `json:"priority"`
}

func (row *ACL) OvnTableName() string {
	return "ACL"
}

func (row *ACL) OvnIsRoot() bool {
	return false
}

func (row *ACL) OvnUuid() string {
	return row.Uuid
}

func (row *ACL) OvnSetExternalIds(k, v string) {
	if row.ExternalIds == nil {
		row.ExternalIds = map[string]string{}
	}
	row.ExternalIds[k] = v
}

func (row *ACL) OvnGetExternalIds(k string) (string, bool) {
	if row.ExternalIds == nil {
		return "", false
	}
	r, ok := row.ExternalIds[k]
	return r, ok
}

func (row *ACL) OvnRemoveExternalIds(k string) (string, bool) {
	if row.ExternalIds == nil {
		return "", false
	}
	r, ok := row.ExternalIds[k]
	if ok {
		delete(row.ExternalIds, k)
	}
	return r, ok
}

func (row *ACL) SetColumn(name string, val interface{}) (err error) {
	defer func() {
		if panicErr := recover(); panicErr != nil {
			err = errors.Wrapf(panicErr.(error), "%s: %#v", name, fmt.Sprintf("%#v", val))
		}
	}()
	switch name {
	case "_uuid":
		row.Uuid = ensureUuid(val)
	case "match":
		row.Match = ensureString(val)
	case "action":
		row.Action = ensureString(val)
	case "log":
		row.Log = ensureBoolean(val)
	case "severity":
		row.Severity = ensureStringOptional(val)
	case "external_ids":
		row.ExternalIds = ensureMapStringString(val)
	case "name":
		row.Name = ensureStringOptional(val)
	case "priority":
		row.Priority = ensureInteger(val)
	case "direction":
		row.Direction = ensureString(val)
	default:
		panic(ErrUnknownColumn)
	}
	return
}

func (row *ACL) MatchNonZeros(row1 *ACL) bool {
	if !matchStringIfNonZero(row.Action, row1.Action) {
		return false
	}
	if !matchBooleanIfNonZero(row.Log, row1.Log) {
		return false
	}
	if !matchStringOptionalIfNonZero(row.Severity, row1.Severity) {
		return false
	}
	if !matchMapStringStringIfNonZero(row.ExternalIds, row1.ExternalIds) {
		return false
	}
	if !matchStringOptionalIfNonZero(row.Name, row1.Name) {
		return false
	}
	if !matchIntegerIfNonZero(row.Priority, row1.Priority) {
		return false
	}
	if !matchStringIfNonZero(row.Direction, row1.Direction) {
		return false
	}
	if !matchStringIfNonZero(row.Match, row1.Match) {
		return false
	}
	return true
}

func (row *ACL) OvnArgs() []string {
	r := []string{}
	r = append(r, OvnArgsStringOptional("name", row.Name)...)
	r = append(r, OvnArgsInteger("priority", row.Priority)...)
	r = append(r, OvnArgsString("direction", row.Direction)...)
	r = append(r, OvnArgsString("match", row.Match)...)
	r = append(r, OvnArgsString("action", row.Action)...)
	r = append(r, OvnArgsBoolean("log", row.Log)...)
	r = append(r, OvnArgsStringOptional("severity", row.Severity)...)
	r = append(r, OvnArgsMapStringString("external_ids", row.ExternalIds)...)
	return r
}

type ACLTable []ACL

func (tbl *ACLTable) NewRow() IRow {
	*tbl = append(*tbl, ACL{})
	return &(*tbl)[len(*tbl)-1]
}

func (tbl ACLTable) OvnTableName() string {
	return "ACL"
}

func (tbl ACLTable) OvnIsRoot() bool {
	return false
}

func (tbl ACLTable) Rows() []IRow {
	r := make([]IRow, len(tbl))
	for i := range tbl {
		r[i] = &tbl[i]
	}
	return r
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

type AddressSet struct {
	Uuid        string            `json:"-"`
	Addresses   []string          `json:"addresses"`
	ExternalIds map[string]string `json:"external_ids"`
	Name        string            `json:"name"`
}

func (row *AddressSet) OvnTableName() string {
	return "Address_Set"
}

func (row *AddressSet) OvnIsRoot() bool {
	return true
}

func (row *AddressSet) OvnUuid() string {
	return row.Uuid
}

func (row *AddressSet) OvnSetExternalIds(k, v string) {
	if row.ExternalIds == nil {
		row.ExternalIds = map[string]string{}
	}
	row.ExternalIds[k] = v
}

func (row *AddressSet) OvnGetExternalIds(k string) (string, bool) {
	if row.ExternalIds == nil {
		return "", false
	}
	r, ok := row.ExternalIds[k]
	return r, ok
}

func (row *AddressSet) OvnRemoveExternalIds(k string) (string, bool) {
	if row.ExternalIds == nil {
		return "", false
	}
	r, ok := row.ExternalIds[k]
	if ok {
		delete(row.ExternalIds, k)
	}
	return r, ok
}

func (row *AddressSet) SetColumn(name string, val interface{}) (err error) {
	defer func() {
		if panicErr := recover(); panicErr != nil {
			err = errors.Wrapf(panicErr.(error), "%s: %#v", name, fmt.Sprintf("%#v", val))
		}
	}()
	switch name {
	case "_uuid":
		row.Uuid = ensureUuid(val)
	case "external_ids":
		row.ExternalIds = ensureMapStringString(val)
	case "name":
		row.Name = ensureString(val)
	case "addresses":
		row.Addresses = ensureStringMultiples(val)
	default:
		panic(ErrUnknownColumn)
	}
	return
}

func (row *AddressSet) MatchNonZeros(row1 *AddressSet) bool {
	if !matchMapStringStringIfNonZero(row.ExternalIds, row1.ExternalIds) {
		return false
	}
	if !matchStringIfNonZero(row.Name, row1.Name) {
		return false
	}
	if !matchStringMultiplesIfNonZero(row.Addresses, row1.Addresses) {
		return false
	}
	return true
}

func (row *AddressSet) OvnArgs() []string {
	r := []string{}
	r = append(r, OvnArgsString("name", row.Name)...)
	r = append(r, OvnArgsStringMultiples("addresses", row.Addresses)...)
	r = append(r, OvnArgsMapStringString("external_ids", row.ExternalIds)...)
	return r
}

type AddressSetTable []AddressSet

func (tbl *AddressSetTable) NewRow() IRow {
	*tbl = append(*tbl, AddressSet{})
	return &(*tbl)[len(*tbl)-1]
}

func (tbl AddressSetTable) OvnTableName() string {
	return "Address_Set"
}

func (tbl AddressSetTable) OvnIsRoot() bool {
	return true
}

func (tbl AddressSetTable) Rows() []IRow {
	r := make([]IRow, len(tbl))
	for i := range tbl {
		r[i] = &tbl[i]
	}
	return r
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

type LogicalSwitch struct {
	Uuid         string            `json:"-"`
	ExternalIds  map[string]string `json:"external_ids"`
	Name         string            `json:"name"`
	Ports        []string          `json:"ports"`
	Acls         []string          `json:"acls"`
	QosRules     []string          `json:"qos_rules"`
	LoadBalancer []string          `json:"load_balancer"`
	DnsRecords   []string          `json:"dns_records"`
	OtherConfig  map[string]string `json:"other_config"`
}

func (row *LogicalSwitch) OvnTableName() string {
	return "Logical_Switch"
}

func (row *LogicalSwitch) OvnIsRoot() bool {
	return true
}

func (row *LogicalSwitch) OvnUuid() string {
	return row.Uuid
}

func (row *LogicalSwitch) OvnSetExternalIds(k, v string) {
	if row.ExternalIds == nil {
		row.ExternalIds = map[string]string{}
	}
	row.ExternalIds[k] = v
}

func (row *LogicalSwitch) OvnGetExternalIds(k string) (string, bool) {
	if row.ExternalIds == nil {
		return "", false
	}
	r, ok := row.ExternalIds[k]
	return r, ok
}

func (row *LogicalSwitch) OvnRemoveExternalIds(k string) (string, bool) {
	if row.ExternalIds == nil {
		return "", false
	}
	r, ok := row.ExternalIds[k]
	if ok {
		delete(row.ExternalIds, k)
	}
	return r, ok
}

func (row *LogicalSwitch) SetColumn(name string, val interface{}) (err error) {
	defer func() {
		if panicErr := recover(); panicErr != nil {
			err = errors.Wrapf(panicErr.(error), "%s: %#v", name, fmt.Sprintf("%#v", val))
		}
	}()
	switch name {
	case "_uuid":
		row.Uuid = ensureUuid(val)
	case "qos_rules":
		row.QosRules = ensureUuidMultiples(val)
	case "load_balancer":
		row.LoadBalancer = ensureUuidMultiples(val)
	case "dns_records":
		row.DnsRecords = ensureUuidMultiples(val)
	case "other_config":
		row.OtherConfig = ensureMapStringString(val)
	case "external_ids":
		row.ExternalIds = ensureMapStringString(val)
	case "name":
		row.Name = ensureString(val)
	case "ports":
		row.Ports = ensureUuidMultiples(val)
	case "acls":
		row.Acls = ensureUuidMultiples(val)
	default:
		panic(ErrUnknownColumn)
	}
	return
}

func (row *LogicalSwitch) MatchNonZeros(row1 *LogicalSwitch) bool {
	if !matchUuidMultiplesIfNonZero(row.QosRules, row1.QosRules) {
		return false
	}
	if !matchUuidMultiplesIfNonZero(row.LoadBalancer, row1.LoadBalancer) {
		return false
	}
	if !matchUuidMultiplesIfNonZero(row.DnsRecords, row1.DnsRecords) {
		return false
	}
	if !matchMapStringStringIfNonZero(row.OtherConfig, row1.OtherConfig) {
		return false
	}
	if !matchMapStringStringIfNonZero(row.ExternalIds, row1.ExternalIds) {
		return false
	}
	if !matchStringIfNonZero(row.Name, row1.Name) {
		return false
	}
	if !matchUuidMultiplesIfNonZero(row.Ports, row1.Ports) {
		return false
	}
	if !matchUuidMultiplesIfNonZero(row.Acls, row1.Acls) {
		return false
	}
	return true
}

func (row *LogicalSwitch) OvnArgs() []string {
	r := []string{}
	r = append(r, OvnArgsUuidMultiples("qos_rules", row.QosRules)...)
	r = append(r, OvnArgsUuidMultiples("load_balancer", row.LoadBalancer)...)
	r = append(r, OvnArgsUuidMultiples("dns_records", row.DnsRecords)...)
	r = append(r, OvnArgsMapStringString("other_config", row.OtherConfig)...)
	r = append(r, OvnArgsMapStringString("external_ids", row.ExternalIds)...)
	r = append(r, OvnArgsString("name", row.Name)...)
	r = append(r, OvnArgsUuidMultiples("ports", row.Ports)...)
	r = append(r, OvnArgsUuidMultiples("acls", row.Acls)...)
	return r
}

type LogicalSwitchTable []LogicalSwitch

func (tbl *LogicalSwitchTable) NewRow() IRow {
	*tbl = append(*tbl, LogicalSwitch{})
	return &(*tbl)[len(*tbl)-1]
}

func (tbl LogicalSwitchTable) OvnTableName() string {
	return "Logical_Switch"
}

func (tbl LogicalSwitchTable) OvnIsRoot() bool {
	return true
}

func (tbl LogicalSwitchTable) Rows() []IRow {
	r := make([]IRow, len(tbl))
	for i := range tbl {
		r[i] = &tbl[i]
	}
	return r
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

type NAT struct {
	Uuid        string            `json:"-"`
	Type        string            `json:"type"`
	ExternalIds map[string]string `json:"external_ids"`
	ExternalIp  string            `json:"external_ip"`
	ExternalMac *string           `json:"external_mac"`
	LogicalIp   string            `json:"logical_ip"`
	LogicalPort *string           `json:"logical_port"`
}

func (row *NAT) OvnTableName() string {
	return "NAT"
}

func (row *NAT) OvnIsRoot() bool {
	return false
}

func (row *NAT) OvnUuid() string {
	return row.Uuid
}

func (row *NAT) OvnSetExternalIds(k, v string) {
	if row.ExternalIds == nil {
		row.ExternalIds = map[string]string{}
	}
	row.ExternalIds[k] = v
}

func (row *NAT) OvnGetExternalIds(k string) (string, bool) {
	if row.ExternalIds == nil {
		return "", false
	}
	r, ok := row.ExternalIds[k]
	return r, ok
}

func (row *NAT) OvnRemoveExternalIds(k string) (string, bool) {
	if row.ExternalIds == nil {
		return "", false
	}
	r, ok := row.ExternalIds[k]
	if ok {
		delete(row.ExternalIds, k)
	}
	return r, ok
}

func (row *NAT) SetColumn(name string, val interface{}) (err error) {
	defer func() {
		if panicErr := recover(); panicErr != nil {
			err = errors.Wrapf(panicErr.(error), "%s: %#v", name, fmt.Sprintf("%#v", val))
		}
	}()
	switch name {
	case "_uuid":
		row.Uuid = ensureUuid(val)
	case "type":
		row.Type = ensureString(val)
	case "external_ids":
		row.ExternalIds = ensureMapStringString(val)
	case "external_ip":
		row.ExternalIp = ensureString(val)
	case "external_mac":
		row.ExternalMac = ensureStringOptional(val)
	case "logical_ip":
		row.LogicalIp = ensureString(val)
	case "logical_port":
		row.LogicalPort = ensureStringOptional(val)
	default:
		panic(ErrUnknownColumn)
	}
	return
}

func (row *NAT) MatchNonZeros(row1 *NAT) bool {
	if !matchMapStringStringIfNonZero(row.ExternalIds, row1.ExternalIds) {
		return false
	}
	if !matchStringIfNonZero(row.ExternalIp, row1.ExternalIp) {
		return false
	}
	if !matchStringOptionalIfNonZero(row.ExternalMac, row1.ExternalMac) {
		return false
	}
	if !matchStringIfNonZero(row.LogicalIp, row1.LogicalIp) {
		return false
	}
	if !matchStringOptionalIfNonZero(row.LogicalPort, row1.LogicalPort) {
		return false
	}
	if !matchStringIfNonZero(row.Type, row1.Type) {
		return false
	}
	return true
}

func (row *NAT) OvnArgs() []string {
	r := []string{}
	r = append(r, OvnArgsMapStringString("external_ids", row.ExternalIds)...)
	r = append(r, OvnArgsString("external_ip", row.ExternalIp)...)
	r = append(r, OvnArgsStringOptional("external_mac", row.ExternalMac)...)
	r = append(r, OvnArgsString("logical_ip", row.LogicalIp)...)
	r = append(r, OvnArgsStringOptional("logical_port", row.LogicalPort)...)
	r = append(r, OvnArgsString("type", row.Type)...)
	return r
}

type NATTable []NAT

func (tbl *NATTable) NewRow() IRow {
	*tbl = append(*tbl, NAT{})
	return &(*tbl)[len(*tbl)-1]
}

func (tbl NATTable) OvnTableName() string {
	return "NAT"
}

func (tbl NATTable) OvnIsRoot() bool {
	return false
}

func (tbl NATTable) Rows() []IRow {
	r := make([]IRow, len(tbl))
	for i := range tbl {
		r[i] = &tbl[i]
	}
	return r
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

type DHCPOptions struct {
	Uuid        string            `json:"-"`
	ExternalIds map[string]string `json:"external_ids"`
	Cidr        string            `json:"cidr"`
	Options     map[string]string `json:"options"`
}

func (row *DHCPOptions) OvnTableName() string {
	return "DHCP_Options"
}

func (row *DHCPOptions) OvnIsRoot() bool {
	return true
}

func (row *DHCPOptions) OvnUuid() string {
	return row.Uuid
}

func (row *DHCPOptions) OvnSetExternalIds(k, v string) {
	if row.ExternalIds == nil {
		row.ExternalIds = map[string]string{}
	}
	row.ExternalIds[k] = v
}

func (row *DHCPOptions) OvnGetExternalIds(k string) (string, bool) {
	if row.ExternalIds == nil {
		return "", false
	}
	r, ok := row.ExternalIds[k]
	return r, ok
}

func (row *DHCPOptions) OvnRemoveExternalIds(k string) (string, bool) {
	if row.ExternalIds == nil {
		return "", false
	}
	r, ok := row.ExternalIds[k]
	if ok {
		delete(row.ExternalIds, k)
	}
	return r, ok
}

func (row *DHCPOptions) SetColumn(name string, val interface{}) (err error) {
	defer func() {
		if panicErr := recover(); panicErr != nil {
			err = errors.Wrapf(panicErr.(error), "%s: %#v", name, fmt.Sprintf("%#v", val))
		}
	}()
	switch name {
	case "_uuid":
		row.Uuid = ensureUuid(val)
	case "cidr":
		row.Cidr = ensureString(val)
	case "options":
		row.Options = ensureMapStringString(val)
	case "external_ids":
		row.ExternalIds = ensureMapStringString(val)
	default:
		panic(ErrUnknownColumn)
	}
	return
}

func (row *DHCPOptions) MatchNonZeros(row1 *DHCPOptions) bool {
	if !matchStringIfNonZero(row.Cidr, row1.Cidr) {
		return false
	}
	if !matchMapStringStringIfNonZero(row.Options, row1.Options) {
		return false
	}
	if !matchMapStringStringIfNonZero(row.ExternalIds, row1.ExternalIds) {
		return false
	}
	return true
}

func (row *DHCPOptions) OvnArgs() []string {
	r := []string{}
	r = append(r, OvnArgsString("cidr", row.Cidr)...)
	r = append(r, OvnArgsMapStringString("options", row.Options)...)
	r = append(r, OvnArgsMapStringString("external_ids", row.ExternalIds)...)
	return r
}

type DHCPOptionsTable []DHCPOptions

func (tbl *DHCPOptionsTable) NewRow() IRow {
	*tbl = append(*tbl, DHCPOptions{})
	return &(*tbl)[len(*tbl)-1]
}

func (tbl DHCPOptionsTable) OvnTableName() string {
	return "DHCP_Options"
}

func (tbl DHCPOptionsTable) OvnIsRoot() bool {
	return true
}

func (tbl DHCPOptionsTable) Rows() []IRow {
	r := make([]IRow, len(tbl))
	for i := range tbl {
		r[i] = &tbl[i]
	}
	return r
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

type LoadBalancer struct {
	Uuid        string            `json:"-"`
	ExternalIds map[string]string `json:"external_ids"`
	Name        string            `json:"name"`
	Vips        map[string]string `json:"vips"`
	Protocol    *string           `json:"protocol"`
}

func (row *LoadBalancer) OvnTableName() string {
	return "Load_Balancer"
}

func (row *LoadBalancer) OvnIsRoot() bool {
	return true
}

func (row *LoadBalancer) OvnUuid() string {
	return row.Uuid
}

func (row *LoadBalancer) OvnSetExternalIds(k, v string) {
	if row.ExternalIds == nil {
		row.ExternalIds = map[string]string{}
	}
	row.ExternalIds[k] = v
}

func (row *LoadBalancer) OvnGetExternalIds(k string) (string, bool) {
	if row.ExternalIds == nil {
		return "", false
	}
	r, ok := row.ExternalIds[k]
	return r, ok
}

func (row *LoadBalancer) OvnRemoveExternalIds(k string) (string, bool) {
	if row.ExternalIds == nil {
		return "", false
	}
	r, ok := row.ExternalIds[k]
	if ok {
		delete(row.ExternalIds, k)
	}
	return r, ok
}

func (row *LoadBalancer) SetColumn(name string, val interface{}) (err error) {
	defer func() {
		if panicErr := recover(); panicErr != nil {
			err = errors.Wrapf(panicErr.(error), "%s: %#v", name, fmt.Sprintf("%#v", val))
		}
	}()
	switch name {
	case "_uuid":
		row.Uuid = ensureUuid(val)
	case "name":
		row.Name = ensureString(val)
	case "vips":
		row.Vips = ensureMapStringString(val)
	case "protocol":
		row.Protocol = ensureStringOptional(val)
	case "external_ids":
		row.ExternalIds = ensureMapStringString(val)
	default:
		panic(ErrUnknownColumn)
	}
	return
}

func (row *LoadBalancer) MatchNonZeros(row1 *LoadBalancer) bool {
	if !matchMapStringStringIfNonZero(row.Vips, row1.Vips) {
		return false
	}
	if !matchStringOptionalIfNonZero(row.Protocol, row1.Protocol) {
		return false
	}
	if !matchMapStringStringIfNonZero(row.ExternalIds, row1.ExternalIds) {
		return false
	}
	if !matchStringIfNonZero(row.Name, row1.Name) {
		return false
	}
	return true
}

func (row *LoadBalancer) OvnArgs() []string {
	r := []string{}
	r = append(r, OvnArgsString("name", row.Name)...)
	r = append(r, OvnArgsMapStringString("vips", row.Vips)...)
	r = append(r, OvnArgsStringOptional("protocol", row.Protocol)...)
	r = append(r, OvnArgsMapStringString("external_ids", row.ExternalIds)...)
	return r
}

type LoadBalancerTable []LoadBalancer

func (tbl *LoadBalancerTable) NewRow() IRow {
	*tbl = append(*tbl, LoadBalancer{})
	return &(*tbl)[len(*tbl)-1]
}

func (tbl LoadBalancerTable) OvnTableName() string {
	return "Load_Balancer"
}

func (tbl LoadBalancerTable) OvnIsRoot() bool {
	return true
}

func (tbl LoadBalancerTable) Rows() []IRow {
	r := make([]IRow, len(tbl))
	for i := range tbl {
		r[i] = &tbl[i]
	}
	return r
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

type SSL struct {
	Uuid            string            `json:"-"`
	SslCiphers      string            `json:"ssl_ciphers"`
	ExternalIds     map[string]string `json:"external_ids"`
	PrivateKey      string            `json:"private_key"`
	Certificate     string            `json:"certificate"`
	CaCert          string            `json:"ca_cert"`
	BootstrapCaCert bool              `json:"bootstrap_ca_cert"`
	SslProtocols    string            `json:"ssl_protocols"`
}

func (row *SSL) OvnTableName() string {
	return "SSL"
}

func (row *SSL) OvnIsRoot() bool {
	return false
}

func (row *SSL) OvnUuid() string {
	return row.Uuid
}

func (row *SSL) OvnSetExternalIds(k, v string) {
	if row.ExternalIds == nil {
		row.ExternalIds = map[string]string{}
	}
	row.ExternalIds[k] = v
}

func (row *SSL) OvnGetExternalIds(k string) (string, bool) {
	if row.ExternalIds == nil {
		return "", false
	}
	r, ok := row.ExternalIds[k]
	return r, ok
}

func (row *SSL) OvnRemoveExternalIds(k string) (string, bool) {
	if row.ExternalIds == nil {
		return "", false
	}
	r, ok := row.ExternalIds[k]
	if ok {
		delete(row.ExternalIds, k)
	}
	return r, ok
}

func (row *SSL) SetColumn(name string, val interface{}) (err error) {
	defer func() {
		if panicErr := recover(); panicErr != nil {
			err = errors.Wrapf(panicErr.(error), "%s: %#v", name, fmt.Sprintf("%#v", val))
		}
	}()
	switch name {
	case "_uuid":
		row.Uuid = ensureUuid(val)
	case "ssl_protocols":
		row.SslProtocols = ensureString(val)
	case "ssl_ciphers":
		row.SslCiphers = ensureString(val)
	case "external_ids":
		row.ExternalIds = ensureMapStringString(val)
	case "private_key":
		row.PrivateKey = ensureString(val)
	case "certificate":
		row.Certificate = ensureString(val)
	case "ca_cert":
		row.CaCert = ensureString(val)
	case "bootstrap_ca_cert":
		row.BootstrapCaCert = ensureBoolean(val)
	default:
		panic(ErrUnknownColumn)
	}
	return
}

func (row *SSL) MatchNonZeros(row1 *SSL) bool {
	if !matchMapStringStringIfNonZero(row.ExternalIds, row1.ExternalIds) {
		return false
	}
	if !matchStringIfNonZero(row.PrivateKey, row1.PrivateKey) {
		return false
	}
	if !matchStringIfNonZero(row.Certificate, row1.Certificate) {
		return false
	}
	if !matchStringIfNonZero(row.CaCert, row1.CaCert) {
		return false
	}
	if !matchBooleanIfNonZero(row.BootstrapCaCert, row1.BootstrapCaCert) {
		return false
	}
	if !matchStringIfNonZero(row.SslProtocols, row1.SslProtocols) {
		return false
	}
	if !matchStringIfNonZero(row.SslCiphers, row1.SslCiphers) {
		return false
	}
	return true
}

func (row *SSL) OvnArgs() []string {
	r := []string{}
	r = append(r, OvnArgsBoolean("bootstrap_ca_cert", row.BootstrapCaCert)...)
	r = append(r, OvnArgsString("ssl_protocols", row.SslProtocols)...)
	r = append(r, OvnArgsString("ssl_ciphers", row.SslCiphers)...)
	r = append(r, OvnArgsMapStringString("external_ids", row.ExternalIds)...)
	r = append(r, OvnArgsString("private_key", row.PrivateKey)...)
	r = append(r, OvnArgsString("certificate", row.Certificate)...)
	r = append(r, OvnArgsString("ca_cert", row.CaCert)...)
	return r
}

type SSLTable []SSL

func (tbl *SSLTable) NewRow() IRow {
	*tbl = append(*tbl, SSL{})
	return &(*tbl)[len(*tbl)-1]
}

func (tbl SSLTable) OvnTableName() string {
	return "SSL"
}

func (tbl SSLTable) OvnIsRoot() bool {
	return false
}

func (tbl SSLTable) Rows() []IRow {
	r := make([]IRow, len(tbl))
	for i := range tbl {
		r[i] = &tbl[i]
	}
	return r
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

type NBGlobal struct {
	Uuid        string            `json:"-"`
	Connections []string          `json:"connections"`
	Ssl         *string           `json:"ssl"`
	NbCfg       int64             `json:"nb_cfg"`
	SbCfg       int64             `json:"sb_cfg"`
	HvCfg       int64             `json:"hv_cfg"`
	ExternalIds map[string]string `json:"external_ids"`
}

func (row *NBGlobal) OvnTableName() string {
	return "NB_Global"
}

func (row *NBGlobal) OvnIsRoot() bool {
	return true
}

func (row *NBGlobal) OvnUuid() string {
	return row.Uuid
}

func (row *NBGlobal) OvnSetExternalIds(k, v string) {
	if row.ExternalIds == nil {
		row.ExternalIds = map[string]string{}
	}
	row.ExternalIds[k] = v
}

func (row *NBGlobal) OvnGetExternalIds(k string) (string, bool) {
	if row.ExternalIds == nil {
		return "", false
	}
	r, ok := row.ExternalIds[k]
	return r, ok
}

func (row *NBGlobal) OvnRemoveExternalIds(k string) (string, bool) {
	if row.ExternalIds == nil {
		return "", false
	}
	r, ok := row.ExternalIds[k]
	if ok {
		delete(row.ExternalIds, k)
	}
	return r, ok
}

func (row *NBGlobal) SetColumn(name string, val interface{}) (err error) {
	defer func() {
		if panicErr := recover(); panicErr != nil {
			err = errors.Wrapf(panicErr.(error), "%s: %#v", name, fmt.Sprintf("%#v", val))
		}
	}()
	switch name {
	case "_uuid":
		row.Uuid = ensureUuid(val)
	case "sb_cfg":
		row.SbCfg = ensureInteger(val)
	case "hv_cfg":
		row.HvCfg = ensureInteger(val)
	case "external_ids":
		row.ExternalIds = ensureMapStringString(val)
	case "connections":
		row.Connections = ensureUuidMultiples(val)
	case "ssl":
		row.Ssl = ensureUuidOptional(val)
	case "nb_cfg":
		row.NbCfg = ensureInteger(val)
	default:
		panic(ErrUnknownColumn)
	}
	return
}

func (row *NBGlobal) MatchNonZeros(row1 *NBGlobal) bool {
	if !matchUuidOptionalIfNonZero(row.Ssl, row1.Ssl) {
		return false
	}
	if !matchIntegerIfNonZero(row.NbCfg, row1.NbCfg) {
		return false
	}
	if !matchIntegerIfNonZero(row.SbCfg, row1.SbCfg) {
		return false
	}
	if !matchIntegerIfNonZero(row.HvCfg, row1.HvCfg) {
		return false
	}
	if !matchMapStringStringIfNonZero(row.ExternalIds, row1.ExternalIds) {
		return false
	}
	if !matchUuidMultiplesIfNonZero(row.Connections, row1.Connections) {
		return false
	}
	return true
}

func (row *NBGlobal) OvnArgs() []string {
	r := []string{}
	r = append(r, OvnArgsInteger("sb_cfg", row.SbCfg)...)
	r = append(r, OvnArgsInteger("hv_cfg", row.HvCfg)...)
	r = append(r, OvnArgsMapStringString("external_ids", row.ExternalIds)...)
	r = append(r, OvnArgsUuidMultiples("connections", row.Connections)...)
	r = append(r, OvnArgsUuidOptional("ssl", row.Ssl)...)
	r = append(r, OvnArgsInteger("nb_cfg", row.NbCfg)...)
	return r
}

type NBGlobalTable []NBGlobal

func (tbl *NBGlobalTable) NewRow() IRow {
	*tbl = append(*tbl, NBGlobal{})
	return &(*tbl)[len(*tbl)-1]
}

func (tbl NBGlobalTable) OvnTableName() string {
	return "NB_Global"
}

func (tbl NBGlobalTable) OvnIsRoot() bool {
	return true
}

func (tbl NBGlobalTable) Rows() []IRow {
	r := make([]IRow, len(tbl))
	for i := range tbl {
		r[i] = &tbl[i]
	}
	return r
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

type LogicalRouterStaticRoute struct {
	Uuid        string            `json:"-"`
	IpPrefix    string            `json:"ip_prefix"`
	Policy      *string           `json:"policy"`
	Nexthop     string            `json:"nexthop"`
	OutputPort  *string           `json:"output_port"`
	ExternalIds map[string]string `json:"external_ids"`
}

func (row *LogicalRouterStaticRoute) OvnTableName() string {
	return "Logical_Router_Static_Route"
}

func (row *LogicalRouterStaticRoute) OvnIsRoot() bool {
	return false
}

func (row *LogicalRouterStaticRoute) OvnUuid() string {
	return row.Uuid
}

func (row *LogicalRouterStaticRoute) OvnSetExternalIds(k, v string) {
	if row.ExternalIds == nil {
		row.ExternalIds = map[string]string{}
	}
	row.ExternalIds[k] = v
}

func (row *LogicalRouterStaticRoute) OvnGetExternalIds(k string) (string, bool) {
	if row.ExternalIds == nil {
		return "", false
	}
	r, ok := row.ExternalIds[k]
	return r, ok
}

func (row *LogicalRouterStaticRoute) OvnRemoveExternalIds(k string) (string, bool) {
	if row.ExternalIds == nil {
		return "", false
	}
	r, ok := row.ExternalIds[k]
	if ok {
		delete(row.ExternalIds, k)
	}
	return r, ok
}

func (row *LogicalRouterStaticRoute) SetColumn(name string, val interface{}) (err error) {
	defer func() {
		if panicErr := recover(); panicErr != nil {
			err = errors.Wrapf(panicErr.(error), "%s: %#v", name, fmt.Sprintf("%#v", val))
		}
	}()
	switch name {
	case "_uuid":
		row.Uuid = ensureUuid(val)
	case "external_ids":
		row.ExternalIds = ensureMapStringString(val)
	case "ip_prefix":
		row.IpPrefix = ensureString(val)
	case "policy":
		row.Policy = ensureStringOptional(val)
	case "nexthop":
		row.Nexthop = ensureString(val)
	case "output_port":
		row.OutputPort = ensureStringOptional(val)
	default:
		panic(ErrUnknownColumn)
	}
	return
}

func (row *LogicalRouterStaticRoute) MatchNonZeros(row1 *LogicalRouterStaticRoute) bool {
	if !matchStringIfNonZero(row.IpPrefix, row1.IpPrefix) {
		return false
	}
	if !matchStringOptionalIfNonZero(row.Policy, row1.Policy) {
		return false
	}
	if !matchStringIfNonZero(row.Nexthop, row1.Nexthop) {
		return false
	}
	if !matchStringOptionalIfNonZero(row.OutputPort, row1.OutputPort) {
		return false
	}
	if !matchMapStringStringIfNonZero(row.ExternalIds, row1.ExternalIds) {
		return false
	}
	return true
}

func (row *LogicalRouterStaticRoute) OvnArgs() []string {
	r := []string{}
	r = append(r, OvnArgsString("ip_prefix", row.IpPrefix)...)
	r = append(r, OvnArgsStringOptional("policy", row.Policy)...)
	r = append(r, OvnArgsString("nexthop", row.Nexthop)...)
	r = append(r, OvnArgsStringOptional("output_port", row.OutputPort)...)
	r = append(r, OvnArgsMapStringString("external_ids", row.ExternalIds)...)
	return r
}

type LogicalRouterStaticRouteTable []LogicalRouterStaticRoute

func (tbl *LogicalRouterStaticRouteTable) NewRow() IRow {
	*tbl = append(*tbl, LogicalRouterStaticRoute{})
	return &(*tbl)[len(*tbl)-1]
}

func (tbl LogicalRouterStaticRouteTable) OvnTableName() string {
	return "Logical_Router_Static_Route"
}

func (tbl LogicalRouterStaticRouteTable) OvnIsRoot() bool {
	return false
}

func (tbl LogicalRouterStaticRouteTable) Rows() []IRow {
	r := make([]IRow, len(tbl))
	for i := range tbl {
		r[i] = &tbl[i]
	}
	return r
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
