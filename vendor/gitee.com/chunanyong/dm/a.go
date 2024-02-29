/*
 * Copyright (c) 2000-2018, 达梦数据库有限公司.
 * All rights reserved.
 */
package dm

import (
	"bytes"
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"strconv"
	"time"
	"unicode/utf8"

	"gitee.com/chunanyong/dm/security"
)

const (
	Dm_build_1343 = 8192
	Dm_build_1344 = 2 * time.Second
)

type dm_build_1345 struct {
	dm_build_1346 net.Conn
	dm_build_1347 *tls.Conn
	dm_build_1348 *Dm_build_1009
	dm_build_1349 *DmConnection
	dm_build_1350 security.Cipher
	dm_build_1351 bool
	dm_build_1352 bool
	dm_build_1353 *security.DhKey

	dm_build_1354 bool
	dm_build_1355 string
	dm_build_1356 bool
}

func dm_build_1357(dm_build_1358 context.Context, dm_build_1359 *DmConnection) (*dm_build_1345, error) {
	var dm_build_1360 net.Conn
	var dm_build_1361 error

	dialsLock.RLock()
	dm_build_1362, dm_build_1363 := dials[dm_build_1359.dmConnector.dialName]
	dialsLock.RUnlock()
	if dm_build_1363 {
		dm_build_1360, dm_build_1361 = dm_build_1362(dm_build_1358, dm_build_1359.dmConnector.host+":"+strconv.Itoa(int(dm_build_1359.dmConnector.port)))
	} else {
		dm_build_1360, dm_build_1361 = dm_build_1365(dm_build_1359.dmConnector.host+":"+strconv.Itoa(int(dm_build_1359.dmConnector.port)), time.Duration(dm_build_1359.dmConnector.socketTimeout)*time.Second)
	}
	if dm_build_1361 != nil {
		return nil, dm_build_1361
	}

	dm_build_1364 := dm_build_1345{}
	dm_build_1364.dm_build_1346 = dm_build_1360
	dm_build_1364.dm_build_1348 = Dm_build_1012(Dm_build_14)
	dm_build_1364.dm_build_1349 = dm_build_1359
	dm_build_1364.dm_build_1351 = false
	dm_build_1364.dm_build_1352 = false
	dm_build_1364.dm_build_1354 = false
	dm_build_1364.dm_build_1355 = ""
	dm_build_1364.dm_build_1356 = false
	dm_build_1359.Access = &dm_build_1364

	return &dm_build_1364, nil
}

func dm_build_1365(dm_build_1366 string, dm_build_1367 time.Duration) (net.Conn, error) {
	dm_build_1368, dm_build_1369 := net.DialTimeout("tcp", dm_build_1366, dm_build_1367)
	if dm_build_1369 != nil {
		return &net.TCPConn{}, ECGO_COMMUNITION_ERROR.addDetail("\tdial address: " + dm_build_1366).throw()
	}

	if tcpConn, ok := dm_build_1368.(*net.TCPConn); ok {
		tcpConn.SetKeepAlive(true)
		tcpConn.SetKeepAlivePeriod(Dm_build_1344)
		tcpConn.SetNoDelay(true)

	}
	return dm_build_1368, nil
}

func (dm_build_1371 *dm_build_1345) dm_build_1370(dm_build_1372 dm_build_135) bool {
	var dm_build_1373 = dm_build_1371.dm_build_1349.dmConnector.compress
	if dm_build_1372.dm_build_150() == Dm_build_42 || dm_build_1373 == Dm_build_91 {
		return false
	}

	if dm_build_1373 == Dm_build_89 {
		return true
	} else if dm_build_1373 == Dm_build_90 {
		return !dm_build_1371.dm_build_1349.Local && dm_build_1372.dm_build_148() > Dm_build_88
	}

	return false
}

func (dm_build_1375 *dm_build_1345) dm_build_1374(dm_build_1376 dm_build_135) bool {
	var dm_build_1377 = dm_build_1375.dm_build_1349.dmConnector.compress
	if dm_build_1376.dm_build_150() == Dm_build_42 || dm_build_1377 == Dm_build_91 {
		return false
	}

	if dm_build_1377 == Dm_build_89 {
		return true
	} else if dm_build_1377 == Dm_build_90 {
		return dm_build_1375.dm_build_1348.Dm_build_1276(Dm_build_50) == 1
	}

	return false
}

func (dm_build_1379 *dm_build_1345) dm_build_1378(dm_build_1380 dm_build_135) (err error) {
	defer func() {
		if p := recover(); p != nil {
			if _, ok := p.(string); ok {
				err = ECGO_COMMUNITION_ERROR.addDetail("\t" + p.(string)).throw()
			} else {
				err = fmt.Errorf("internal error: %v", p)
			}
		}
	}()

	dm_build_1382 := dm_build_1380.dm_build_148()

	if dm_build_1382 > 0 {

		if dm_build_1379.dm_build_1370(dm_build_1380) {
			var retBytes, err = Compress(dm_build_1379.dm_build_1348, Dm_build_43, int(dm_build_1382), int(dm_build_1379.dm_build_1349.dmConnector.compressID))
			if err != nil {
				return err
			}

			dm_build_1379.dm_build_1348.Dm_build_1023(Dm_build_43)

			dm_build_1379.dm_build_1348.Dm_build_1064(dm_build_1382)

			dm_build_1379.dm_build_1348.Dm_build_1092(retBytes)

			dm_build_1380.dm_build_149(int32(len(retBytes)) + ULINT_SIZE)

			dm_build_1379.dm_build_1348.Dm_build_1196(Dm_build_50, 1)
		}

		if dm_build_1379.dm_build_1352 {
			dm_build_1382 = dm_build_1380.dm_build_148()
			var retBytes = dm_build_1379.dm_build_1350.Encrypt(dm_build_1379.dm_build_1348.Dm_build_1303(Dm_build_43, int(dm_build_1382)), true)

			dm_build_1379.dm_build_1348.Dm_build_1023(Dm_build_43)

			dm_build_1379.dm_build_1348.Dm_build_1092(retBytes)

			dm_build_1380.dm_build_149(int32(len(retBytes)))
		}
	}

	if dm_build_1379.dm_build_1348.Dm_build_1021() > Dm_build_15 {
		return ECGO_MSG_TOO_LONG.throw()
	}

	dm_build_1380.dm_build_144()
	if dm_build_1379.dm_build_1613(dm_build_1380) {
		if dm_build_1379.dm_build_1347 != nil {
			dm_build_1379.dm_build_1348.Dm_build_1026(0)
			if _, err := dm_build_1379.dm_build_1348.Dm_build_1045(dm_build_1379.dm_build_1347); err != nil {
				return err
			}
		}
	} else {
		dm_build_1379.dm_build_1348.Dm_build_1026(0)
		if _, err := dm_build_1379.dm_build_1348.Dm_build_1045(dm_build_1379.dm_build_1346); err != nil {
			return err
		}
	}
	return nil
}

func (dm_build_1384 *dm_build_1345) dm_build_1383(dm_build_1385 dm_build_135) (err error) {
	defer func() {
		if p := recover(); p != nil {
			if _, ok := p.(string); ok {
				err = ECGO_COMMUNITION_ERROR.addDetail("\t" + p.(string)).throw()
			} else {
				err = fmt.Errorf("internal error: %v", p)
			}
		}
	}()

	dm_build_1387 := int32(0)
	if dm_build_1384.dm_build_1613(dm_build_1385) {
		if dm_build_1384.dm_build_1347 != nil {
			dm_build_1384.dm_build_1348.Dm_build_1023(0)
			if _, err := dm_build_1384.dm_build_1348.Dm_build_1039(dm_build_1384.dm_build_1347, Dm_build_43); err != nil {
				return err
			}

			dm_build_1387 = dm_build_1385.dm_build_148()
			if dm_build_1387 > 0 {
				if _, err := dm_build_1384.dm_build_1348.Dm_build_1039(dm_build_1384.dm_build_1347, int(dm_build_1387)); err != nil {
					return err
				}
			}
		}
	} else {

		dm_build_1384.dm_build_1348.Dm_build_1023(0)
		if _, err := dm_build_1384.dm_build_1348.Dm_build_1039(dm_build_1384.dm_build_1346, Dm_build_43); err != nil {
			return err
		}
		dm_build_1387 = dm_build_1385.dm_build_148()

		if dm_build_1387 > 0 {
			if _, err := dm_build_1384.dm_build_1348.Dm_build_1039(dm_build_1384.dm_build_1346, int(dm_build_1387)); err != nil {
				return err
			}
		}
	}

	dm_build_1385.dm_build_145()

	dm_build_1387 = dm_build_1385.dm_build_148()
	if dm_build_1387 <= 0 {
		return nil
	}

	if dm_build_1384.dm_build_1352 {
		ebytes := dm_build_1384.dm_build_1348.Dm_build_1303(Dm_build_43, int(dm_build_1387))
		bytes, err := dm_build_1384.dm_build_1350.Decrypt(ebytes, true)
		if err != nil {
			return err
		}
		dm_build_1384.dm_build_1348.Dm_build_1023(Dm_build_43)
		dm_build_1384.dm_build_1348.Dm_build_1092(bytes)
		dm_build_1385.dm_build_149(int32(len(bytes)))
	}

	if dm_build_1384.dm_build_1374(dm_build_1385) {

		dm_build_1387 = dm_build_1385.dm_build_148()
		cbytes := dm_build_1384.dm_build_1348.Dm_build_1303(Dm_build_43+ULINT_SIZE, int(dm_build_1387-ULINT_SIZE))
		bytes, err := UnCompress(cbytes, int(dm_build_1384.dm_build_1349.dmConnector.compressID))
		if err != nil {
			return err
		}
		dm_build_1384.dm_build_1348.Dm_build_1023(Dm_build_43)
		dm_build_1384.dm_build_1348.Dm_build_1092(bytes)
		dm_build_1385.dm_build_149(int32(len(bytes)))
	}
	return nil
}

func (dm_build_1389 *dm_build_1345) dm_build_1388(dm_build_1390 dm_build_135) (dm_build_1391 interface{}, dm_build_1392 error) {
	if dm_build_1389.dm_build_1356 {
		return nil, ECGO_CONNECTION_CLOSED.throw()
	}
	dm_build_1393 := dm_build_1389.dm_build_1349
	dm_build_1393.mu.Lock()
	defer dm_build_1393.mu.Unlock()
	dm_build_1392 = dm_build_1390.dm_build_139(dm_build_1390)
	if dm_build_1392 != nil {
		return nil, dm_build_1392
	}

	dm_build_1392 = dm_build_1389.dm_build_1378(dm_build_1390)
	if dm_build_1392 != nil {
		return nil, dm_build_1392
	}

	dm_build_1392 = dm_build_1389.dm_build_1383(dm_build_1390)
	if dm_build_1392 != nil {
		return nil, dm_build_1392
	}

	return dm_build_1390.dm_build_143(dm_build_1390)
}

func (dm_build_1395 *dm_build_1345) dm_build_1394() (*dm_build_592, error) {

	Dm_build_1396 := dm_build_598(dm_build_1395)
	_, dm_build_1397 := dm_build_1395.dm_build_1388(Dm_build_1396)
	if dm_build_1397 != nil {
		return nil, dm_build_1397
	}

	return Dm_build_1396, nil
}

func (dm_build_1399 *dm_build_1345) dm_build_1398() error {

	dm_build_1400 := dm_build_459(dm_build_1399)
	_, dm_build_1401 := dm_build_1399.dm_build_1388(dm_build_1400)
	if dm_build_1401 != nil {
		return dm_build_1401
	}

	return nil
}

func (dm_build_1403 *dm_build_1345) dm_build_1402() error {

	var dm_build_1404 *dm_build_592
	var err error
	if dm_build_1404, err = dm_build_1403.dm_build_1394(); err != nil {
		return err
	}

	if dm_build_1403.dm_build_1349.sslEncrypt == 2 {
		if err = dm_build_1403.dm_build_1609(false); err != nil {
			return ECGO_INIT_SSL_FAILED.addDetail("\n" + err.Error()).throw()
		}
	} else if dm_build_1403.dm_build_1349.sslEncrypt == 1 {
		if err = dm_build_1403.dm_build_1609(true); err != nil {
			return ECGO_INIT_SSL_FAILED.addDetail("\n" + err.Error()).throw()
		}
	}

	if dm_build_1403.dm_build_1352 || dm_build_1403.dm_build_1351 {
		k, err := dm_build_1403.dm_build_1599()
		if err != nil {
			return err
		}
		sessionKey := security.ComputeSessionKey(k, dm_build_1404.Dm_build_596)
		encryptType := dm_build_1404.dm_build_594
		hashType := int(dm_build_1404.Dm_build_595)
		if encryptType == -1 {
			encryptType = security.DES_CFB
		}
		if hashType == -1 {
			hashType = security.MD5
		}
		err = dm_build_1403.dm_build_1602(encryptType, sessionKey, dm_build_1403.dm_build_1349.dmConnector.cipherPath, hashType)
		if err != nil {
			return err
		}
	}

	if err := dm_build_1403.dm_build_1398(); err != nil {
		return err
	}
	return nil
}

func (dm_build_1407 *dm_build_1345) Dm_build_1406(dm_build_1408 *DmStatement) error {
	dm_build_1409 := dm_build_621(dm_build_1407, dm_build_1408)
	_, dm_build_1410 := dm_build_1407.dm_build_1388(dm_build_1409)
	if dm_build_1410 != nil {
		return dm_build_1410
	}

	return nil
}

func (dm_build_1412 *dm_build_1345) Dm_build_1411(dm_build_1413 int32) error {
	dm_build_1414 := dm_build_631(dm_build_1412, dm_build_1413)
	_, dm_build_1415 := dm_build_1412.dm_build_1388(dm_build_1414)
	if dm_build_1415 != nil {
		return dm_build_1415
	}

	return nil
}

func (dm_build_1417 *dm_build_1345) Dm_build_1416(dm_build_1418 *DmStatement, dm_build_1419 bool, dm_build_1420 int16) (*execRetInfo, error) {
	dm_build_1421 := dm_build_498(dm_build_1417, dm_build_1418, dm_build_1419, dm_build_1420)
	dm_build_1422, dm_build_1423 := dm_build_1417.dm_build_1388(dm_build_1421)
	if dm_build_1423 != nil {
		return nil, dm_build_1423
	}
	return dm_build_1422.(*execRetInfo), nil
}

func (dm_build_1425 *dm_build_1345) Dm_build_1424(dm_build_1426 *DmStatement, dm_build_1427 int16) (*execRetInfo, error) {
	return dm_build_1425.Dm_build_1416(dm_build_1426, false, Dm_build_95)
}

func (dm_build_1429 *dm_build_1345) Dm_build_1428(dm_build_1430 *DmStatement, dm_build_1431 []OptParameter) (*execRetInfo, error) {
	dm_build_1432, dm_build_1433 := dm_build_1429.dm_build_1388(dm_build_238(dm_build_1429, dm_build_1430, dm_build_1431))
	if dm_build_1433 != nil {
		return nil, dm_build_1433
	}

	return dm_build_1432.(*execRetInfo), nil
}

func (dm_build_1435 *dm_build_1345) Dm_build_1434(dm_build_1436 *DmStatement, dm_build_1437 int16) (*execRetInfo, error) {
	return dm_build_1435.Dm_build_1416(dm_build_1436, true, dm_build_1437)
}

func (dm_build_1439 *dm_build_1345) Dm_build_1438(dm_build_1440 *DmStatement, dm_build_1441 [][]interface{}) (*execRetInfo, error) {
	dm_build_1442 := dm_build_270(dm_build_1439, dm_build_1440, dm_build_1441)
	dm_build_1443, dm_build_1444 := dm_build_1439.dm_build_1388(dm_build_1442)
	if dm_build_1444 != nil {
		return nil, dm_build_1444
	}
	return dm_build_1443.(*execRetInfo), nil
}

func (dm_build_1446 *dm_build_1345) Dm_build_1445(dm_build_1447 *DmStatement, dm_build_1448 [][]interface{}, dm_build_1449 bool) (*execRetInfo, error) {
	var dm_build_1450, dm_build_1451 = 0, 0
	var dm_build_1452 = len(dm_build_1448)
	var dm_build_1453 [][]interface{}
	var dm_build_1454 = NewExceInfo()
	dm_build_1454.updateCounts = make([]int64, dm_build_1452)
	var dm_build_1455 = false
	for dm_build_1450 < dm_build_1452 {
		for dm_build_1451 = dm_build_1450; dm_build_1451 < dm_build_1452; dm_build_1451++ {
			paramData := dm_build_1448[dm_build_1451]
			bindData := make([]interface{}, dm_build_1447.paramCount)
			dm_build_1455 = false
			for icol := 0; icol < int(dm_build_1447.paramCount); icol++ {
				if dm_build_1447.bindParams[icol].ioType == IO_TYPE_OUT {
					continue
				}
				if dm_build_1446.dm_build_1582(bindData, paramData, icol) {
					dm_build_1455 = true
					break
				}
			}

			if dm_build_1455 {
				break
			}
			dm_build_1453 = append(dm_build_1453, bindData)
		}

		if dm_build_1451 != dm_build_1450 {
			tmpExecInfo, err := dm_build_1446.Dm_build_1438(dm_build_1447, dm_build_1453)
			if err != nil {
				return nil, err
			}
			dm_build_1453 = dm_build_1453[0:0]
			dm_build_1454.union(tmpExecInfo, dm_build_1450, dm_build_1451-dm_build_1450)
		}

		if dm_build_1451 < dm_build_1452 {
			tmpExecInfo, err := dm_build_1446.Dm_build_1456(dm_build_1447, dm_build_1448[dm_build_1451], dm_build_1449)
			if err != nil {
				return nil, err
			}

			dm_build_1449 = true
			dm_build_1454.union(tmpExecInfo, dm_build_1451, 1)
		}

		dm_build_1450 = dm_build_1451 + 1
	}
	for _, i := range dm_build_1454.updateCounts {
		if i > 0 {
			dm_build_1454.updateCount += i
		}
	}
	return dm_build_1454, nil
}

func (dm_build_1457 *dm_build_1345) Dm_build_1456(dm_build_1458 *DmStatement, dm_build_1459 []interface{}, dm_build_1460 bool) (*execRetInfo, error) {

	var dm_build_1461 = make([]interface{}, dm_build_1458.paramCount)
	for icol := 0; icol < int(dm_build_1458.paramCount); icol++ {
		if dm_build_1458.bindParams[icol].ioType == IO_TYPE_OUT {
			continue
		}
		if dm_build_1457.dm_build_1582(dm_build_1461, dm_build_1459, icol) {

			if !dm_build_1460 {
				preExecute := dm_build_487(dm_build_1457, dm_build_1458, dm_build_1458.bindParams)
				dm_build_1457.dm_build_1388(preExecute)
				dm_build_1460 = true
			}

			dm_build_1457.dm_build_1588(dm_build_1458, dm_build_1458.bindParams[icol], icol, dm_build_1459[icol].(iOffRowBinder))
			dm_build_1461[icol] = ParamDataEnum_OFF_ROW
		}
	}

	var dm_build_1462 = make([][]interface{}, 1, 1)
	dm_build_1462[0] = dm_build_1461

	dm_build_1463 := dm_build_270(dm_build_1457, dm_build_1458, dm_build_1462)
	dm_build_1464, dm_build_1465 := dm_build_1457.dm_build_1388(dm_build_1463)
	if dm_build_1465 != nil {
		return nil, dm_build_1465
	}
	return dm_build_1464.(*execRetInfo), nil
}

func (dm_build_1467 *dm_build_1345) Dm_build_1466(dm_build_1468 *DmStatement, dm_build_1469 int16) (*execRetInfo, error) {
	dm_build_1470 := dm_build_474(dm_build_1467, dm_build_1468, dm_build_1469)

	dm_build_1471, dm_build_1472 := dm_build_1467.dm_build_1388(dm_build_1470)
	if dm_build_1472 != nil {
		return nil, dm_build_1472
	}
	return dm_build_1471.(*execRetInfo), nil
}

func (dm_build_1474 *dm_build_1345) Dm_build_1473(dm_build_1475 *innerRows, dm_build_1476 int64) (*execRetInfo, error) {
	dm_build_1477 := dm_build_377(dm_build_1474, dm_build_1475, dm_build_1476, INT64_MAX)
	dm_build_1478, dm_build_1479 := dm_build_1474.dm_build_1388(dm_build_1477)
	if dm_build_1479 != nil {
		return nil, dm_build_1479
	}
	return dm_build_1478.(*execRetInfo), nil
}

func (dm_build_1481 *dm_build_1345) Commit() error {
	dm_build_1482 := dm_build_223(dm_build_1481)
	_, dm_build_1483 := dm_build_1481.dm_build_1388(dm_build_1482)
	if dm_build_1483 != nil {
		return dm_build_1483
	}

	return nil
}

func (dm_build_1485 *dm_build_1345) Rollback() error {
	dm_build_1486 := dm_build_536(dm_build_1485)
	_, dm_build_1487 := dm_build_1485.dm_build_1388(dm_build_1486)
	if dm_build_1487 != nil {
		return dm_build_1487
	}

	return nil
}

func (dm_build_1489 *dm_build_1345) Dm_build_1488(dm_build_1490 *DmConnection) error {
	dm_build_1491 := dm_build_541(dm_build_1489, dm_build_1490.IsoLevel)
	_, dm_build_1492 := dm_build_1489.dm_build_1388(dm_build_1491)
	if dm_build_1492 != nil {
		return dm_build_1492
	}

	return nil
}

func (dm_build_1494 *dm_build_1345) Dm_build_1493(dm_build_1495 *DmStatement, dm_build_1496 string) error {
	dm_build_1497 := dm_build_228(dm_build_1494, dm_build_1495, dm_build_1496)
	_, dm_build_1498 := dm_build_1494.dm_build_1388(dm_build_1497)
	if dm_build_1498 != nil {
		return dm_build_1498
	}

	return nil
}

func (dm_build_1500 *dm_build_1345) Dm_build_1499(dm_build_1501 []uint32) ([]int64, error) {
	dm_build_1502 := dm_build_639(dm_build_1500, dm_build_1501)
	dm_build_1503, dm_build_1504 := dm_build_1500.dm_build_1388(dm_build_1502)
	if dm_build_1504 != nil {
		return nil, dm_build_1504
	}
	return dm_build_1503.([]int64), nil
}

func (dm_build_1506 *dm_build_1345) Close() error {
	if dm_build_1506.dm_build_1356 {
		return nil
	}

	dm_build_1507 := dm_build_1506.dm_build_1346.Close()
	if dm_build_1507 != nil {
		return dm_build_1507
	}

	dm_build_1506.dm_build_1349 = nil
	dm_build_1506.dm_build_1356 = true
	return nil
}

func (dm_build_1509 *dm_build_1345) dm_build_1508(dm_build_1510 *lob) (int64, error) {
	dm_build_1511 := dm_build_410(dm_build_1509, dm_build_1510)
	dm_build_1512, dm_build_1513 := dm_build_1509.dm_build_1388(dm_build_1511)
	if dm_build_1513 != nil {
		return 0, dm_build_1513
	}
	return dm_build_1512.(int64), nil
}

func (dm_build_1515 *dm_build_1345) dm_build_1514(dm_build_1516 *lob, dm_build_1517 int32, dm_build_1518 int32) (*lobRetInfo, error) {
	dm_build_1519 := dm_build_395(dm_build_1515, dm_build_1516, int(dm_build_1517), int(dm_build_1518))
	dm_build_1520, dm_build_1521 := dm_build_1515.dm_build_1388(dm_build_1519)
	if dm_build_1521 != nil {
		return nil, dm_build_1521
	}
	return dm_build_1520.(*lobRetInfo), nil
}

func (dm_build_1523 *dm_build_1345) dm_build_1522(dm_build_1524 *DmBlob, dm_build_1525 int32, dm_build_1526 int32) ([]byte, error) {
	var dm_build_1527 = make([]byte, dm_build_1526)
	var dm_build_1528 int32 = 0
	var dm_build_1529 int32 = 0
	var dm_build_1530 *lobRetInfo
	var dm_build_1531 []byte
	var dm_build_1532 error
	for dm_build_1528 < dm_build_1526 {
		dm_build_1529 = dm_build_1526 - dm_build_1528
		if dm_build_1529 > Dm_build_128 {
			dm_build_1529 = Dm_build_128
		}
		dm_build_1530, dm_build_1532 = dm_build_1523.dm_build_1514(&dm_build_1524.lob, dm_build_1525+dm_build_1528, dm_build_1529)
		if dm_build_1532 != nil {
			return nil, dm_build_1532
		}
		dm_build_1531 = dm_build_1530.data
		if dm_build_1531 == nil || len(dm_build_1531) == 0 {
			break
		}
		Dm_build_650.Dm_build_706(dm_build_1527, int(dm_build_1528), dm_build_1531, 0, len(dm_build_1531))
		dm_build_1528 += int32(len(dm_build_1531))
		if dm_build_1524.readOver {
			break
		}
	}
	return dm_build_1527, nil
}

func (dm_build_1534 *dm_build_1345) dm_build_1533(dm_build_1535 *DmClob, dm_build_1536 int32, dm_build_1537 int32) (string, error) {
	var dm_build_1538 bytes.Buffer
	var dm_build_1539 int32 = 0
	var dm_build_1540 int32 = 0
	var dm_build_1541 *lobRetInfo
	var dm_build_1542 []byte
	var dm_build_1543 string
	var dm_build_1544 error
	for dm_build_1539 < dm_build_1537 {
		dm_build_1540 = dm_build_1537 - dm_build_1539
		if dm_build_1540 > Dm_build_128/2 {
			dm_build_1540 = Dm_build_128 / 2
		}
		dm_build_1541, dm_build_1544 = dm_build_1534.dm_build_1514(&dm_build_1535.lob, dm_build_1536+dm_build_1539, dm_build_1540)
		if dm_build_1544 != nil {
			return "", dm_build_1544
		}
		dm_build_1542 = dm_build_1541.data
		if dm_build_1542 == nil || len(dm_build_1542) == 0 {
			break
		}
		dm_build_1543 = Dm_build_650.Dm_build_807(dm_build_1542, 0, len(dm_build_1542), dm_build_1535.serverEncoding, dm_build_1534.dm_build_1349)

		dm_build_1538.WriteString(dm_build_1543)
		var strLen = dm_build_1541.charLen
		if strLen == -1 {
			strLen = int64(utf8.RuneCountInString(dm_build_1543))
		}
		dm_build_1539 += int32(strLen)
		if dm_build_1535.readOver {
			break
		}
	}
	return dm_build_1538.String(), nil
}

func (dm_build_1546 *dm_build_1345) dm_build_1545(dm_build_1547 *DmClob, dm_build_1548 int, dm_build_1549 string, dm_build_1550 string) (int, error) {
	var dm_build_1551 = Dm_build_650.Dm_build_866(dm_build_1549, dm_build_1550, dm_build_1546.dm_build_1349)
	var dm_build_1552 = 0
	var dm_build_1553 = len(dm_build_1551)
	var dm_build_1554 = 0
	var dm_build_1555 = 0
	var dm_build_1556 = 0
	var dm_build_1557 = dm_build_1553/Dm_build_127 + 1
	var dm_build_1558 byte = 0
	var dm_build_1559 byte = 0x01
	var dm_build_1560 byte = 0x02
	for i := 0; i < dm_build_1557; i++ {
		dm_build_1558 = 0
		if i == 0 {
			dm_build_1558 |= dm_build_1559
		}
		if i == dm_build_1557-1 {
			dm_build_1558 |= dm_build_1560
		}
		dm_build_1556 = dm_build_1553 - dm_build_1555
		if dm_build_1556 > Dm_build_127 {
			dm_build_1556 = Dm_build_127
		}

		setLobData := dm_build_555(dm_build_1546, &dm_build_1547.lob, dm_build_1558, dm_build_1548, dm_build_1551, dm_build_1552, dm_build_1556)
		ret, err := dm_build_1546.dm_build_1388(setLobData)
		if err != nil {
			return 0, err
		}
		tmp := ret.(int32)
		if err != nil {
			return -1, err
		}
		if tmp <= 0 {
			return dm_build_1554, nil
		} else {
			dm_build_1548 += int(tmp)
			dm_build_1554 += int(tmp)
			dm_build_1555 += dm_build_1556
			dm_build_1552 += dm_build_1556
		}
	}
	return dm_build_1554, nil
}

func (dm_build_1562 *dm_build_1345) dm_build_1561(dm_build_1563 *DmBlob, dm_build_1564 int, dm_build_1565 []byte) (int, error) {
	var dm_build_1566 = 0
	var dm_build_1567 = len(dm_build_1565)
	var dm_build_1568 = 0
	var dm_build_1569 = 0
	var dm_build_1570 = 0
	var dm_build_1571 = dm_build_1567/Dm_build_127 + 1
	var dm_build_1572 byte = 0
	var dm_build_1573 byte = 0x01
	var dm_build_1574 byte = 0x02
	for i := 0; i < dm_build_1571; i++ {
		dm_build_1572 = 0
		if i == 0 {
			dm_build_1572 |= dm_build_1573
		}
		if i == dm_build_1571-1 {
			dm_build_1572 |= dm_build_1574
		}
		dm_build_1570 = dm_build_1567 - dm_build_1569
		if dm_build_1570 > Dm_build_127 {
			dm_build_1570 = Dm_build_127
		}

		setLobData := dm_build_555(dm_build_1562, &dm_build_1563.lob, dm_build_1572, dm_build_1564, dm_build_1565, dm_build_1566, dm_build_1570)
		ret, err := dm_build_1562.dm_build_1388(setLobData)
		if err != nil {
			return 0, err
		}
		tmp := ret.(int32)
		if tmp <= 0 {
			return dm_build_1568, nil
		} else {
			dm_build_1564 += int(tmp)
			dm_build_1568 += int(tmp)
			dm_build_1569 += dm_build_1570
			dm_build_1566 += dm_build_1570
		}
	}
	return dm_build_1568, nil
}

func (dm_build_1576 *dm_build_1345) dm_build_1575(dm_build_1577 *lob, dm_build_1578 int) (int64, error) {
	dm_build_1579 := dm_build_421(dm_build_1576, dm_build_1577, dm_build_1578)
	dm_build_1580, dm_build_1581 := dm_build_1576.dm_build_1388(dm_build_1579)
	if dm_build_1581 != nil {
		return dm_build_1577.length, dm_build_1581
	}
	return dm_build_1580.(int64), nil
}

func (dm_build_1583 *dm_build_1345) dm_build_1582(dm_build_1584 []interface{}, dm_build_1585 []interface{}, dm_build_1586 int) bool {
	var dm_build_1587 = false
	dm_build_1584[dm_build_1586] = dm_build_1585[dm_build_1586]

	if binder, ok := dm_build_1585[dm_build_1586].(iOffRowBinder); ok {
		dm_build_1587 = true
		dm_build_1584[dm_build_1586] = make([]byte, 0)
		var lob lob
		if l, ok := binder.getObj().(DmBlob); ok {
			lob = l.lob
		} else if l, ok := binder.getObj().(DmClob); ok {
			lob = l.lob
		}
		if &lob != nil && lob.canOptimized(dm_build_1583.dm_build_1349) {
			dm_build_1584[dm_build_1586] = &lobCtl{lob.buildCtlData()}
			dm_build_1587 = false
		}
	} else {
		dm_build_1584[dm_build_1586] = dm_build_1585[dm_build_1586]
	}
	return dm_build_1587
}

func (dm_build_1589 *dm_build_1345) dm_build_1588(dm_build_1590 *DmStatement, dm_build_1591 parameter, dm_build_1592 int, dm_build_1593 iOffRowBinder) error {
	var dm_build_1594 = Dm_build_935()
	dm_build_1593.read(dm_build_1594)
	var dm_build_1595 = 0
	for !dm_build_1593.isReadOver() || dm_build_1594.Dm_build_936() > 0 {
		if !dm_build_1593.isReadOver() && dm_build_1594.Dm_build_936() < Dm_build_127 {
			dm_build_1593.read(dm_build_1594)
		}
		if dm_build_1594.Dm_build_936() > Dm_build_127 {
			dm_build_1595 = Dm_build_127
		} else {
			dm_build_1595 = dm_build_1594.Dm_build_936()
		}

		putData := dm_build_526(dm_build_1589, dm_build_1590, int16(dm_build_1592), dm_build_1594, int32(dm_build_1595))
		_, err := dm_build_1589.dm_build_1388(putData)
		if err != nil {
			return err
		}
	}
	return nil
}

func (dm_build_1597 *dm_build_1345) dm_build_1596() ([]byte, error) {
	var dm_build_1598 error
	if dm_build_1597.dm_build_1353 == nil {
		if dm_build_1597.dm_build_1353, dm_build_1598 = security.NewClientKeyPair(); dm_build_1598 != nil {
			return nil, dm_build_1598
		}
	}
	return security.Bn2Bytes(dm_build_1597.dm_build_1353.GetY(), security.DH_KEY_LENGTH), nil
}

func (dm_build_1600 *dm_build_1345) dm_build_1599() (*security.DhKey, error) {
	var dm_build_1601 error
	if dm_build_1600.dm_build_1353 == nil {
		if dm_build_1600.dm_build_1353, dm_build_1601 = security.NewClientKeyPair(); dm_build_1601 != nil {
			return nil, dm_build_1601
		}
	}
	return dm_build_1600.dm_build_1353, nil
}

func (dm_build_1603 *dm_build_1345) dm_build_1602(dm_build_1604 int, dm_build_1605 []byte, dm_build_1606 string, dm_build_1607 int) (dm_build_1608 error) {
	if dm_build_1604 > 0 && dm_build_1604 < security.MIN_EXTERNAL_CIPHER_ID && dm_build_1605 != nil {
		dm_build_1603.dm_build_1350, dm_build_1608 = security.NewSymmCipher(dm_build_1604, dm_build_1605)
	} else if dm_build_1604 >= security.MIN_EXTERNAL_CIPHER_ID {
		if dm_build_1603.dm_build_1350, dm_build_1608 = security.NewThirdPartCipher(dm_build_1604, dm_build_1605, dm_build_1606, dm_build_1607); dm_build_1608 != nil {
			dm_build_1608 = THIRD_PART_CIPHER_INIT_FAILED.addDetailln(dm_build_1608.Error()).throw()
		}
	}
	return
}

func (dm_build_1610 *dm_build_1345) dm_build_1609(dm_build_1611 bool) (dm_build_1612 error) {
	if dm_build_1610.dm_build_1347, dm_build_1612 = security.NewTLSFromTCP(dm_build_1610.dm_build_1346, dm_build_1610.dm_build_1349.dmConnector.sslCertPath, dm_build_1610.dm_build_1349.dmConnector.sslKeyPath, dm_build_1610.dm_build_1349.dmConnector.user); dm_build_1612 != nil {
		return
	}
	if !dm_build_1611 {
		dm_build_1610.dm_build_1347 = nil
	}
	return
}

func (dm_build_1614 *dm_build_1345) dm_build_1613(dm_build_1615 dm_build_135) bool {
	return dm_build_1615.dm_build_150() != Dm_build_42 && dm_build_1614.dm_build_1349.sslEncrypt == 1
}
