package models

import (
	"context"

	"golang.org/x/sync/errgroup"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/utils"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/onecloud/pkg/apis/monitor"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	"yunion.io/x/onecloud/pkg/monitor/dbinit"
	"yunion.io/x/onecloud/pkg/monitor/registry"
	"yunion.io/x/onecloud/pkg/util/rbacutils"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

type SMetricMeasurementManager struct {
	db.SEnabledResourceBaseManager
	db.SStatusStandaloneResourceBaseManager
	db.SScopedResourceBaseManager
}

type SMetricMeasurement struct {
	//db.SVirtualResourceBase
	db.SEnabledResourceBase
	db.SStatusStandaloneResourceBase
	db.SScopedResourceBase

	ResType     string `width:"32" list:"user" update:"user"`
	Database    string `width:"32" list:"user" update:"user"`
	DisplayName string `width:"256" list:"user" update:"user"`
}

var MetricMeasurementManager *SMetricMeasurementManager

func init() {
	MetricMeasurementManager = &SMetricMeasurementManager{
		SStatusStandaloneResourceBaseManager: db.NewStatusStandaloneResourceBaseManager(
			SMetricMeasurement{},
			"metricmeasurement_tbl",
			"metricmeasurement",
			"metricmeasurements",
		),
	}

	MetricMeasurementManager.SetVirtualObject(MetricMeasurementManager)
	registry.RegisterService(MetricMeasurementManager)
}

func (manager *SMetricMeasurementManager) NamespaceScope() rbacutils.TRbacScope {
	return rbacutils.ScopeSystem
}

func (manager *SMetricMeasurementManager) ListItemExportKeys(ctx context.Context, q *sqlchemy.SQuery, userCred mcclient.TokenCredential, keys stringutils2.SSortedStrings) (*sqlchemy.SQuery, error) {
	q, err := manager.SStatusStandaloneResourceBaseManager.ListItemExportKeys(ctx, q, userCred, keys)
	if err != nil {
		return nil, errors.Wrap(err, "SStatusStandaloneResourceBaseManager.ListItemExportKeys")
	}
	q, err = manager.SScopedResourceBaseManager.ListItemExportKeys(ctx, q, userCred, keys)
	if err != nil {
		return nil, errors.Wrap(err, "SScopedResourceBaseManager.ListItemExportKeys")
	}
	return q, nil
}

func (man *SMetricMeasurementManager) ValidateCreateData(
	ctx context.Context, userCred mcclient.TokenCredential,
	ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject,
	data monitor.MetricCreateInput) (monitor.MetricMeasurementCreateInput, error) {
	return data.Measurement, nil
}

func (measurement *SMetricMeasurement) CustomizeCreate(
	ctx context.Context, userCred mcclient.TokenCredential,
	ownerId mcclient.IIdentityProvider,
	query jsonutils.JSONObject,
	data jsonutils.JSONObject,
) error {
	err := measurement.SScopedResourceBase.CustomizeCreate(ctx, userCred, ownerId, query, data)
	if err != nil {
		return err
	}
	input := new(monitor.MetricCreateInput)
	if err := data.Unmarshal(input); err != nil {
		return err
	}

	for _, fieldInput := range input.MetricFields {
		field, err := measurement.SaveMetricField(ctx, userCred, ownerId, fieldInput)
		if err != nil {
			return err
		}
		err = measurement.attachMetricField(ctx, userCred, field)
		if err != nil {
			return err
		}
	}
	return nil
}

func (measurement *SMetricMeasurement) attachMetricField(ctx context.Context, userCred mcclient.TokenCredential,
	field *SMetricField) error {
	count, err := measurement.isAttachMetricField(field)
	if err != nil {
		return err
	}
	if count {
		return httperrors.ErrDuplicateName
	}
	metric := new(SMetric)
	if len(measurement.GetId()) == 0 {
		measurement.Id = db.DefaultUUIDGenerator()
	}
	metric.MeasurementId = measurement.GetId()
	metric.FieldId = field.GetId()
	return metric.DoSave(ctx)
}

func (measurement *SMetricMeasurement) isAttachMetricField(field *SMetricField) (bool, error) {
	q := MetricManager.Query().Equals(MetricManager.GetMasterFieldName(), measurement.GetId()).Equals(MetricManager.
		GetSlaveFieldName(), field.GetId())
	count, err := q.CountWithError()
	if err != nil {
		return false, err
	}
	return count > 0, nil

}

func (measurement *SMetricMeasurement) SaveMetricField(ctx context.Context, userCred mcclient.TokenCredential,
	ownerId mcclient.IIdentityProvider, fieldInput monitor.MetricFieldCreateInput) (*SMetricField, error) {
	return MetricFieldManager.SaveMetricField(ctx, userCred, ownerId, fieldInput)
}

func (manager *SMetricMeasurementManager) ListItemFilter(
	ctx context.Context, q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query monitor.MetricListInput,
) (*sqlchemy.SQuery, error) {
	var err error
	q, err = manager.SStandaloneResourceBaseManager.ListItemFilter(ctx, q, userCred, query.Measurement.StandaloneResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SStandaloneResourceBaseManager.ListItemFilter")
	}
	q, err = manager.SEnabledResourceBaseManager.ListItemFilter(ctx, q, userCred,
		query.Measurement.EnabledResourceBaseListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SEnabledResourceBaseManager.ListItemFilter")
	}
	q, err = manager.SScopedResourceBaseManager.ListItemFilter(ctx, q, userCred,
		query.Measurement.ScopedResourceBaseListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SScopedResourceBaseManager.ListItemFilter")
	}
	if len(query.Measurement.ResType) != 0 {
		q = q.Equals("res_type", query.Measurement.ResType)
	}
	if len(query.Measurement.DisplayName) != 0 {
		q = q.Equals("display_name", query.Measurement.DisplayName)
	}
	joinQuery, err := manager.listFilterMetricField(ctx, userCred, query.MetricFields)
	if err != nil {
		return q, err
	}
	joinSubQuery := joinQuery.SubQuery()
	q = q.Join(joinSubQuery, sqlchemy.Equals(q.Field("id"), joinSubQuery.Field(MetricManager.GetMasterFieldName())))
	return q, nil
}

func (man *SMetricMeasurementManager) OrderByExtraFields(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	input monitor.AlertListInput,
) (*sqlchemy.SQuery, error) {
	var err error

	q, err = man.SStatusStandaloneResourceBaseManager.OrderByExtraFields(ctx, q, userCred, input.StatusStandaloneResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SStandaloneResourceBaseManager.OrderByExtraFields")
	}
	q, err = man.SScopedResourceBaseManager.OrderByExtraFields(ctx, q, userCred, input.ScopedResourceBaseListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SScopedResourceBaseManager.OrderByExtraFields")
	}
	return q, nil
}

func (manager *SMetricMeasurementManager) listFilterMetricField(ctx context.Context, userCred mcclient.TokenCredential, query monitor.MetricFieldListInput) (*sqlchemy.SQuery, error) {
	joinQuery := MetricManager.Query(MetricManager.GetMasterFieldName()).Distinct()

	fieldQuery, err := MetricFieldManager.ListItemFilter(ctx, MetricFieldManager.Query(), userCred, query)
	if err != nil {
		return nil, err
	}
	fieldSubQuery := fieldQuery.SubQuery()
	joinQuery = joinQuery.Join(fieldSubQuery, sqlchemy.Equals(joinQuery.Field(MetricManager.
		GetSlaveFieldName()), fieldSubQuery.Field("id")))
	return joinQuery, nil
}

func (man *SMetricMeasurementManager) FetchCustomizeColumns(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	objs []interface{},
	fields stringutils2.SSortedStrings,
	isList bool,
) []monitor.MetricDetails {
	rows := make([]monitor.MetricDetails, len(objs))
	stdRows := man.SStatusStandaloneResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	scopedRows := man.SScopedResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	for i := range rows {
		rows[i] = monitor.MetricDetails{
			StatusStandaloneResourceDetails: stdRows[i],
			ScopedResourceBaseInfo:          scopedRows[i],
		}
		rows[i], _ = objs[i].(*SMetricMeasurement).GetMoreDetails(rows[i])
	}
	return rows
}
func (measurement *SMetricMeasurement) GetMoreDetails(out monitor.MetricDetails) (monitor.MetricDetails, error) {
	fields, err := measurement.getFields()
	if err != nil {
		log.Errorln(err)
		return out, err
	}
	fieldDetails := make([]monitor.MetricFieldDetail, 0)
	for _, field := range fields {
		fieldObj := jsonutils.Marshal(&field)
		fieldDetail := new(monitor.MetricFieldDetail)
		err := fieldObj.Unmarshal(fieldDetail)
		if err != nil {
			log.Errorln(err)
			return out, err
		}
		fieldDetails = append(fieldDetails, *fieldDetail)
	}
	out.MetricFields = fieldDetails
	return out, nil
}

func (measurement *SMetricMeasurement) ValidateUpdateData(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	data monitor.MetricUpdateInput,
) (monitor.MetricMeasurementUpdateInput, error) {
	if len(data.Measurement.ResType) == 0 {
		return data.Measurement, errors.Wrap(httperrors.ErrNotEmpty, "res_type")
	}
	if !utils.IsInStringArray(data.Measurement.ResType, monitor.MetricResType) {
		return data.Measurement, errors.Wrap(httperrors.ErrBadRequest, "res_type")
	}
	if len(data.Measurement.DisplayName) == 0 {
		return data.Measurement, errors.Wrap(httperrors.ErrNotEmpty, "display_name")
	}
	return data.Measurement, nil
}

func (measurement *SMetricMeasurement) PreUpdate(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject,
	data jsonutils.JSONObject) {
	input := new(monitor.MetricUpdateInput)
	if err := data.Unmarshal(input); err != nil {
		return
	}
	for _, fieldUpdateInput := range input.MetricFields {
		field, err := measurement.getMetricField(fieldUpdateInput.Name)
		if err != nil {
			log.Errorln(err, "metric measurement getMetricFields error")
			continue
		}
		if field == nil {
			log.Errorf("field:%s do not attach with measurement:%s", fieldUpdateInput.Name, measurement.Name)
			continue
		}
		err = measurement.updateMetricField(ctx, userCred, field, fieldUpdateInput)
		if err != nil {
			log.Errorln(err, "measurement updateMetricField")
		}
	}
}

func (measurement *SMetricMeasurement) getMetricField(name string) (*SMetricField, error) {
	fields := make([]SMetricField, 0)
	q := measurement.getFieldsQuery()
	q = q.Equals("name", name)
	err := db.FetchModelObjects(MetricFieldManager, q, &fields)
	if err != nil {
		return nil, err
	}
	if len(fields) == 0 {
		return nil, nil
	}
	if len(fields) != 1 {
		return nil, errors.Wrapf(sqlchemy.ErrDuplicateEntry, "found %d, metric field name: %s", len(fields), name)
	}
	return &fields[0], nil
}

func (measurement *SMetricMeasurement) getFields() ([]SMetricField, error) {
	fields := make([]SMetricField, 0)
	q := measurement.getFieldsQuery()
	err := db.FetchModelObjects(MetricFieldManager, q, &fields)
	if err != nil {
		return nil, err
	}
	return fields, nil
}

func (manager *SMetricMeasurementManager) getMeasurement(query *sqlchemy.SQuery) ([]SMetricMeasurement, error) {
	measurements := make([]SMetricMeasurement, 0)
	err := db.FetchModelObjects(MetricMeasurementManager, query, &measurements)
	if err != nil {
		return nil, err
	}
	return measurements, nil
}

func (manager *SMetricMeasurementManager) getInfluxdbMeasurements() (influxdbMeasurements []monitor.
	InfluxMeasurement, err error) {
	metric, err := manager.getMeasurement(manager.Query())
	if err != nil {
		return
	}
	for i, _ := range metric {
		influxdbMeasurements = append(influxdbMeasurements, monitor.InfluxMeasurement{
			Database:    metric[i].Database,
			Measurement: metric[i].Name,
		})
	}
	return

}

func (measurement *SMetricMeasurement) getFieldsQuery() *sqlchemy.SQuery {
	metricJoinQuery := MetricManager.Query().Equals(MetricManager.GetMasterFieldName(), measurement.GetId()).SubQuery()
	q := MetricFieldManager.Query()
	q = q.Join(metricJoinQuery, sqlchemy.Equals(q.Field("id"), metricJoinQuery.Field(MetricManager.GetSlaveFieldName())))
	return q
}

func (measurement *SMetricMeasurement) updateMetricField(ctx context.Context, userCred mcclient.TokenCredential,
	field *SMetricField, input monitor.MetricFieldUpdateInput) error {
	_, err := field.ValidateUpdateData(ctx, userCred, nil, input)
	if err != nil {
		return err
	}
	_, err = db.Update(field, func() error {
		field.Unit = input.Unit
		field.DisplayName = input.DisplayName
		return nil
	})
	return err
}

func (manager *SMetricMeasurementManager) Init() error {
	return nil
}

func (man *SMetricMeasurementManager) Run(ctx context.Context) error {

	err := man.initJsonMetricInfo(ctx)
	if err != nil {
		return errors.Wrap(err, "init metric json error")
	}
	log.Infoln("========metric_measurement_field init finish==========")
	return nil
}

func (manager *SMetricMeasurementManager) initMeasurementDatabase(ctx context.Context) (err error) {
	databases, err := DataSourceManager.GetDatabases()
	if err != nil {
		return err
	}
	databaseArr, err := databases.GetArray("databases")
	if err != nil {
		return err
	}
	databaseGroup, _ := errgroup.WithContext(ctx)
	for dIndex, _ := range databaseArr {
		databaseTmp := databaseArr[dIndex]
		databaseStr, _ := databaseTmp.GetString()
		databaseGroup.Go(manager.getMeasurementAsyn(ctx, databaseStr))
	}
	err = databaseGroup.Wait()
	return
}

func (manager *SMetricMeasurementManager) getMeasurementAsyn(ctx context.Context, database string) func() error {
	return func() error {
		query := jsonutils.NewDict()
		query.Add(jsonutils.NewString(database), "database")

		measurements, err := DataSourceManager.GetMeasurementsWithOutTimeFilter(query, "", "")
		if err != nil {
			return err
		}
		measurementArr, err := measurements.GetArray("measurements")
		if err != nil {
			return err
		}
		metrics := make([]monitor.MetricCreateInput, 0)
		for mIndex, _ := range measurementArr {
			measurementStr, _ := measurementArr[mIndex].GetString("measurement")
			metric := monitor.MetricCreateInput{}
			metricMea := monitor.MetricMeasurementCreateInput{}
			metricMea.Name = measurementStr
			metricMea.Database = database
			metric.Measurement = metricMea
			metric.MetricFields = make([]monitor.MetricFieldCreateInput, 0)
			metrics = append(metrics, metric)
		}
		return manager.initMetrics(ctx, metrics)
	}
}

func (manager *SMetricMeasurementManager) initJsonMetricInfo(ctx context.Context) error {
	metricDescriptions, err := jsonutils.ParseString(dbinit.MetricDescriptions)
	if err != nil {
		return errors.Wrap(err, "SMetricMeasurementManager Parse MetricDescriptionstr error")
	}
	metrics := make([]monitor.MetricCreateInput, 0)
	err = metricDescriptions.Unmarshal(&metrics)
	if err != nil {
		return errors.Wrap(err, "SMetricMeasurementManager Unmarshal MetricDescriptionstr error")
	}
	return manager.initMetrics(ctx, metrics)
}

func (manager *SMetricMeasurementManager) initMetrics(ctx context.Context, metrics []monitor.MetricCreateInput) (err error) {
	measurementGroup, _ := errgroup.WithContext(ctx)
	count := 0
	for mIndex, _ := range metrics {

		measurementTmp := metrics[mIndex]
		if mIndex < len(metrics) && count < 10 {
			count++
			measurementGroup.Go(func() error {
				return manager.initMeasurementAndFieldInfo(measurementTmp)
			})
		}
		if count == 1 {
			err := measurementGroup.Wait()
			if err != nil {
				return err
			}
			count = 0
		}
	}
	err = measurementGroup.Wait()
	return
}

func (manager *SMetricMeasurementManager) initMeasurementAndFieldInfo(createInput monitor.MetricCreateInput) error {
	userCred := auth.AdminCredential()
	listInput := new(monitor.MetricListInput)
	listInput.Measurement.Names = []string{createInput.Measurement.Name}
	measurements, err := manager.getMeasurementByName(userCred, *listInput)
	if err != nil {
		return errors.Wrap(err, "join query get  measurement error")
	}
	unInsertFields := createInput.MetricFields
	updateFields := make([]monitor.MetricFieldCreateInput, 0)
	if len(measurements) != 0 {
		unInsertFields, updateFields = measurements[0].getInsertAndUpdateFields(userCred, createInput)
	}
	if len(measurements) == 0 {
		_, err := db.DoCreate(manager, context.Background(), userCred, jsonutils.NewDict(),
			jsonutils.Marshal(&createInput),
			userCred)
		if err != nil {
			err = errors.Wrap(err, "create metricdescription error")
		}
		return err
	}
	createInput.MetricFields = unInsertFields
	return measurements[0].insertOrUpdateMetric(userCred, createInput, updateFields)
}

func (manager *SMetricMeasurementManager) getMeasurementByName(userCred mcclient.TokenCredential,
	listInput monitor.MetricListInput) ([]SMetricMeasurement, error) {
	query, err := MetricMeasurementManager.ListItemFilter(context.Background(), MetricMeasurementManager.Query(), userCred,
		listInput)
	if err != nil {
		return nil, err
	}
	return manager.getMeasurement(query)
}

func (self *SMetricMeasurement) getInsertAndUpdateFields(userCred mcclient.TokenCredential, input monitor.MetricCreateInput) (unInsertFields,
	updateFields []monitor.MetricFieldCreateInput) {
	measurementsIns := []interface{}{self}
	details := MetricMeasurementManager.FetchCustomizeColumns(context.Background(), userCred, jsonutils.NewDict(), measurementsIns,
		stringutils2.NewSortedStrings([]string{}), true)
	unInsertFields, updateFields = getUnInsertFields(input.MetricFields, details[0])
	return
}

func (self *SMetricMeasurement) insertOrUpdateMetric(userCred mcclient.TokenCredential,
	createInput monitor.MetricCreateInput, updateFields []monitor.MetricFieldCreateInput) error {
	_, err := db.Update(self, func() error {
		if len(createInput.Measurement.DisplayName) != 0 {
			self.DisplayName = createInput.Measurement.DisplayName
		}
		if len(createInput.Measurement.ResType) != 0 {
			self.ResType = createInput.Measurement.ResType
		}
		if len(createInput.Measurement.Database) != 0 {
			self.Database = createInput.Measurement.Database
		}
		return nil
	})
	if err != nil {
		return errors.Wrap(err, "update metric measurement error")
	}

	err = self.CustomizeCreate(context.Background(), userCred, userCred, jsonutils.NewDict(),
		jsonutils.Marshal(&createInput))
	if err != nil {
		return errors.Wrap(err, "create metric field error")
	}
	dbFields, _ := self.getFields()
	for i, _ := range dbFields {
		for upIndex, _ := range updateFields {
			if dbFields[i].Name == updateFields[upIndex].Name {
				_, err := db.Update(&dbFields[i], func() error {
					if len(updateFields[upIndex].DisplayName) != 0 {
						dbFields[i].DisplayName = updateFields[upIndex].DisplayName
					}
					if len(updateFields[upIndex].Unit) != 0 {
						dbFields[i].Unit = updateFields[upIndex].Unit
					}
					return nil
				})
				if err != nil {
					return errors.Wrap(err, "update metric field error")
				}
			}
		}
	}
	return nil
}

func getUnInsertFields(searchFields []monitor.MetricFieldCreateInput,
	dbFields monitor.MetricDetails) (unInsertFields, updateFields []monitor.
	MetricFieldCreateInput) {
	fieldCountMap := make(map[string]int)
	fieldMap := make(map[string]monitor.MetricFieldCreateInput, 0)
	for _, field := range searchFields {
		fieldCountMap[field.Name]++
		fieldMap[field.Name] = field
	}

	for _, dbField := range dbFields.MetricFields {
		count, _ := fieldCountMap[dbField.Name]
		if count == 1 {
			if field, ok := fieldMap[dbField.Name]; ok {
				updateFields = append(updateFields, field)
			}
			delete(fieldCountMap, dbField.Name)
		}
	}
	for fieldName, _ := range fieldCountMap {
		if field, ok := fieldMap[fieldName]; ok {
			unInsertFields = append(unInsertFields, field)
		}
	}
	return unInsertFields, updateFields
}

func (self *SMetricMeasurement) getMetricJoint() ([]SMetric, error) {
	metricJoint := make([]SMetric, 0)
	q := MetricManager.Query().Equals(MetricManager.GetMasterFieldName(), self.Id)
	if err := db.FetchModelObjects(MetricManager, q, &metricJoint); err != nil {
		return nil, err
	}
	return metricJoint, nil
}

func (self *SMetricMeasurement) CustomizeDelete(
	ctx context.Context, userCred mcclient.TokenCredential,
	query jsonutils.JSONObject, data jsonutils.JSONObject) error {
	metricJoint, err := self.getMetricJoint()
	if err != nil {
		return err
	}
	for _, joint := range metricJoint {
		field, err := joint.GetMetricField()
		if err != nil {
			return err
		}
		if err := field.CustomizeDelete(ctx, userCred, query, data); err != nil {
			return err
		}
		if err := field.Delete(ctx, userCred); err != nil {
			return err
		}
		if err := joint.Detach(ctx, userCred); err != nil {
			return err
		}
	}
	return nil
}
