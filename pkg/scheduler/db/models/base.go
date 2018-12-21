package models

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/jinzhu/gorm"
)

type objectFunc func() interface{}

type BaseModel struct {
	CreatedAt time.Time `json:"created_at" gorm:"column:created_at;type:datetime" sql:"DEFAULT:NULL"`
	UpdatedAt time.Time `json:"updated_at" gorm:"column:updated_at;type:datetime" sql:"DEFAULT:NULL"`
	DeletedAt time.Time `json:"deleted_at" gorm:"column:deleted_at;type:datetime" sql:"DEFAULT:NULL"`
	Deleted   bool      `json:"deleted" gorm:"column:deleted;not null;index" sql:"DEFAULT:false"`
}

type StandaloneModel struct {
	BaseModel
	ID          string `json:"id" gorm:"primary_key;column:id;type:varchar(36) CHARACTER SET ascii"`
	Name        string `json:"name" gorm:"column:name;type:varchar(128) CHARACTER SET utf8"`
	Description string `json:"description,omitempty" gorm:"column:description"`
}

func (m *StandaloneModel) UUID() string {
	return m.ID
}

type JointBaseModel struct {
	BaseModel
	RowID string `json:"row_id" gorm:"primary_key;column:row_id"`
}

func (m *JointBaseModel) UUID() string {
	return m.RowID
}

type VirtualResourceModel struct {
	StandaloneModel
	Status           string    `json:"status" gorm:"column:status;not null"`
	TenantID         string    `json:"tenant_id" gorm:"column:tenant_id;not null"`
	UserID           string    `json:"user_id" gorm:"column:user_id;not null"`
	BillingType      string    `json:"billing_type" gorm:"column:billing_type"`
	IsSystem         bool      `json:"is_system" gorm:"column:is_system"`
	PendingDeletedAt time.Time `json:"pending_deleted_at" gorm:"column:pending_deleted_at;type:datetime" sql:"DEFAULT:NULL"`
	PendingDeleted   bool      `json:"pending_deleted" gorm:"column:pending_deleted;not null;index" sql:"DEFAULT:false"`
}

type SharableVirtualResourceModel struct {
	VirtualResourceModel
	IsPublic bool `json:"is_public" gorm:"column:is_public;not null"`
}

type resource struct {
	db        *gorm.DB
	tableName string
	getModel  objectFunc
	getModels objectFunc
}

func newResource(db *gorm.DB, tbl string, model, models objectFunc) (Resourcer, error) {
	if db == nil {
		return nil, fmt.Errorf("gorm db is nil")
	}
	r := new(resource)
	r.db = db
	r.tableName = tbl
	r.getModel = model
	r.getModels = models
	return r, nil
}

func (r *resource) DB() *gorm.DB {
	return r.db
}

func (r *resource) TableName() string {
	return r.tableName
}

func (r *resource) Model() interface{} {
	return r.getModel()
}

func (r *resource) Models() interface{} {
	return r.getModels()
}

func JsonString(obj interface{}) (string, error) {
	bytes, err := json.Marshal(obj)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

func FetchByID(r Resourcer, id string) (interface{}, error) {
	cond := map[string]interface{}{
		"deleted": 0,
		"id":      id,
	}
	obj := r.Model()
	if err := r.DB().Where(condition2String(cond)).First(obj).Error; err != nil {
		return nil, err
	}
	return obj, nil
}

func rowsWithCondIn(r Resourcer, key string, set []string, cond map[string]interface{}) (*sql.Rows, error) {
	return r.DB().Table(r.TableName()).Where(condition2String(cond)).
		Where(fmt.Sprintf("%s in ('%s')", key, strings.Join(set, "','"))).Rows()
}

func rowsNotDeletedIn(r Resourcer, key string, set []string) (*sql.Rows, error) {
	cond := map[string]interface{}{
		"deleted": false,
	}
	return rowsWithCondIn(r, key, set, cond)
}

func rowsNotDeletedInWithCond(r Resourcer, key string, set []string, cond map[string]interface{}) (*sql.Rows, error) {
	cond["deleted"] = false
	return rowsWithCondIn(r, key, set, cond)
}

func virtualResourceRowsNotDeletedIn(r Resourcer, key string, set []string) (*sql.Rows, error) {
	cond := map[string]interface{}{
		"deleted":         false,
		"pending_deleted": false,
	}
	return rowsWithCondIn(r, key, set, cond)
}

func rowsToArray(r Resourcer, rows *sql.Rows) ([]interface{}, error) {
	defer rows.Close()

	columns, _ := rows.Columns()

	objs := make([]interface{}, 0, len(columns))
	for rows.Next() {
		obj := r.Model()
		err := r.DB().ScanRows(rows, obj)
		if err != nil {
			return nil, err
		}
		objs = append(objs, obj)
	}
	return objs, nil
}

func FetchByIDsWithKey(r Resourcer, key string, ids []string) ([]interface{}, error) {
	rows, err := rowsNotDeletedIn(r, key, ids)
	if err != nil {
		return nil, err
	}
	return rowsToArray(r, rows)
}

func FetchByIDsWithKeyAndCond(r Resourcer, key string, ids []string, cond map[string]interface{}) ([]interface{}, error) {
	rows, err := rowsNotDeletedInWithCond(r, key, ids, cond)
	if err != nil {
		return nil, err
	}
	return rowsToArray(r, rows)
}

func FetchByIDs(r Resourcer, ids []string) ([]interface{}, error) {
	return FetchByIDsWithKey(r, "id", ids)
}

func FetchByHostIDs(r Resourcer, ids []string) ([]interface{}, error) {
	return FetchByIDsWithKey(r, "host_id", ids)
}

func FetchGuestByIDs(ids []string) ([]interface{}, error) {
	return FetchByIDs(Guests, ids)
}

func FetchGuestByHostIDs(ids []string) ([]interface{}, error) {
	return FetchByIDsWithKey(Guests, "host_id", ids)
}

func FetchGuestByHostIDsWithCond(ids []string, cond map[string]interface{}) ([]interface{}, error) {
	return FetchByIDsWithKeyAndCond(Guests, "host_id", ids, cond)
}

func FetchHostByIDs(ids []string) ([]interface{}, error) {
	return FetchByIDs(Hosts, ids)
}

func FetchDiskByIDs(ids []string) ([]interface{}, error) {
	return FetchByIDs(Disks, ids)
}

func FetchGroupByIDs(ids []string) ([]interface{}, error) {
	return FetchByIDs(Groups, ids)
}

func FetchByWireIDs(r Resourcer, ids []string) ([]interface{}, error) {
	return FetchByIDsWithKey(r, "wire_id", ids)
}

func FetchByBaremetalIDs(r Resourcer, ids []string) ([]interface{}, error) {
	return FetchByIDsWithKey(r, "baremetal_id", ids)
}

func FetchByGuestIDs(r Resourcer, ids []string) ([]interface{}, error) {
	return FetchByIDsWithKey(r, "guest_id", ids)
}

func AllWithDeleted(r Resourcer) ([]interface{}, error) {
	return AllWithCond(r, map[string]interface{}{})
}

func All(r Resourcer) ([]interface{}, error) {
	cond := map[string]interface{}{
		"deleted": 0,
	}
	return AllWithCond(r, cond)
}

func AllWithCond(r Resourcer, cond map[string]interface{}) ([]interface{}, error) {
	rows, err := r.DB().Model(r.Model()).Where(condition2String(cond)).Rows()
	if err != nil {
		return nil, err
	}
	return rowsToArray(r, rows)
}

func AllIDs(r Resourcer) ([]string, error) {
	cond := map[string]interface{}{
		"deleted": 0,
	}
	return AllIDsWithCond(r, cond)
}

func AllHostIDs() ([]string, error) {
	cond := map[string]interface{}{
		"deleted":    0,
		"host_type!": "baremetal",
	}
	return AllIDsWithCond(Hosts, cond)
}

func AllBaremetalIDs() ([]string, error) {
	cond := map[string]interface{}{
		"deleted":   0,
		"host_type": "baremetal",
	}
	return AllIDsWithCond(Hosts, cond)
}

func condition2String(cond map[string]interface{}) string {
	result := make([]string, 0)
	for key, value := range cond {
		if _, ok := value.(string); ok {
			result = append(result, fmt.Sprintf("%s='%s'", key, value.(string)))
		} else if _, ok := value.(int); ok {
			result = append(result, fmt.Sprintf("%s=%d", key, value.(int)))
		} else if _, ok := value.(int64); ok {
			result = append(result, fmt.Sprintf("%s=%d", key, value.(int64)))
		} else if _, ok := value.(bool); ok {
			result = append(result, fmt.Sprintf("%s=%v", key, value.(bool)))
		}
	}
	return strings.Join(result, " and ")
}

func AllIDsWithCond(r Resourcer, cond map[string]interface{}) ([]string, error) {
	rows, err := r.DB().Table(r.TableName()).Where(condition2String(cond)).Select("id").Rows()
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	ids := []string{}
	for rows.Next() {
		var id string
		rows.Scan(&id)
		ids = append(ids, id)
	}
	return ids, nil
}

type StatusOfHost struct {
	ID        string    `json:"id" gorm:"primary_key;column:id;type:varchar(36) CHARACTER SET ascii"`
	UpdatedAt time.Time `json:"updated_at" gorm:"column:updated_at;type:datetime" sql:"DEFAULT:NULL"`
}

func AllHostStatus(isBaremetal bool) ([]StatusOfHost, error) {
	whereState := "deleted=0 and host_type%s"
	if isBaremetal {
		whereState = fmt.Sprintf(whereState, "='baremetal'")
	} else {
		whereState = fmt.Sprintf(whereState, "!='baremtal'")
	}

	status := make([]StatusOfHost, 0)
	err := Hosts.DB().Table(Hosts.TableName()).
		Select("id, updated_at").
		Where(whereState).
		Scan(&status).Error
	return status, err
}
