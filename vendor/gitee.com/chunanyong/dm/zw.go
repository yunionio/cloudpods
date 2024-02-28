/*
 * Copyright (c) 2000-2018, 达梦数据库有限公司.
 * All rights reserved.
 */

package dm

import (
	"math/rand"
	"strconv"
	"time"

	"gitee.com/chunanyong/dm/util"
)

var rwMap = make(map[string]*rwCounter)

type rwCounter struct {
	ntrx_primary int64

	ntrx_total int64

	primaryPercent float64

	standbyPercent float64

	standbyNTrxMap map[string]int64

	standbyIdMap map[string]int32

	standbyCount int32

	flag []int32

	increments []int32
}

func newRWCounter(primaryPercent int32, standbyCount int32) *rwCounter {
	rwc := new(rwCounter)
	rwc.standbyNTrxMap = make(map[string]int64)
	rwc.standbyIdMap = make(map[string]int32)
	rwc.reset(primaryPercent, standbyCount)
	return rwc
}

func (rwc *rwCounter) reset(primaryPercent int32, standbyCount int32) {
	rwc.ntrx_primary = 0
	rwc.ntrx_total = 0
	rwc.standbyCount = standbyCount
	rwc.increments = make([]int32, standbyCount+1)
	rwc.flag = make([]int32, standbyCount+1)
	var gcd = util.GCD(primaryPercent*standbyCount, 100-primaryPercent)
	rwc.increments[0] = primaryPercent * standbyCount / gcd
	for i, tmp := 1, (100-primaryPercent)/gcd; i < len(rwc.increments); i++ {
		rwc.increments[i] = tmp
	}
	copy(rwc.flag, rwc.increments)

	if standbyCount > 0 {
		rwc.primaryPercent = float64(primaryPercent) / 100.0
		rwc.standbyPercent = float64(100-primaryPercent) / 100.0 / float64(standbyCount)
	} else {
		rwc.primaryPercent = 1
		rwc.standbyPercent = 0
	}
}

// 连接创建成功后调用，需要服务器返回standbyCount
func getRwCounterInstance(conn *DmConnection, standbyCount int32) *rwCounter {
	key := conn.dmConnector.host + "_" + strconv.Itoa(int(conn.dmConnector.port)) + "_" + strconv.Itoa(int(conn.dmConnector.rwPercent))

	rwc, ok := rwMap[key]
	if !ok {
		rwc = newRWCounter(conn.dmConnector.rwPercent, standbyCount)
		rwMap[key] = rwc
	} else if rwc.standbyCount != standbyCount {
		rwc.reset(conn.dmConnector.rwPercent, standbyCount)
	}
	return rwc
}

/**
* @return 主机;
 */
func (rwc *rwCounter) countPrimary() RWSiteEnum {
	rwc.adjustNtrx()
	rwc.increasePrimaryNtrx()
	return PRIMARY
}

/**
* @param dest 主机; 备机; any;
* @return 主机; 备机
 */
func (rwc *rwCounter) count(dest RWSiteEnum, standby *DmConnection) RWSiteEnum {
	rwc.adjustNtrx()
	switch dest {
	case ANYSITE:
		{
			if rwc.primaryPercent == 1 || (rwc.flag[0] > rwc.getStandbyFlag(standby) && rwc.flag[0] > util.Sum(rwc.flag[1:])) {
				rwc.increasePrimaryNtrx()
				dest = PRIMARY
			} else {
				rwc.increaseStandbyNtrx(standby)
				dest = STANDBY
			}
		}
	case STANDBY:
		{
			rwc.increaseStandbyNtrx(standby)
		}
	case PRIMARY:
		{
			rwc.increasePrimaryNtrx()
		}
	}
	return dest
}

/**
* 防止ntrx超出有效范围，等比调整
 */
func (rwc *rwCounter) adjustNtrx() {
	if rwc.ntrx_total >= INT64_MAX {
		var min int64
		var i = 0
		for _, num := range rwc.standbyNTrxMap {
			if i == 0 || num < min {
				min = num
			}
			i++
		}
		if rwc.ntrx_primary < min {
			min = rwc.ntrx_primary
		}
		rwc.ntrx_primary /= min
		rwc.ntrx_total /= min
		for k, v := range rwc.standbyNTrxMap {
			rwc.standbyNTrxMap[k] = v / min
		}
	}

	if rwc.flag[0] <= 0 && util.Sum(rwc.flag[1:]) <= 0 {
		// 如果主库事务数以及所有备库事务数的总和 都 <= 0, 重置事务计数，给每个库的事务计数加上初始计数值
		for i := 0; i < len(rwc.flag); i++ {
			rwc.flag[i] += rwc.increments[i]
		}
	}
}

func (rwc *rwCounter) increasePrimaryNtrx() {
	rwc.ntrx_primary++
	rwc.flag[0]--
	rwc.ntrx_total++
}

//func (rwc *rwCounter) getStandbyNtrx(standby *DmConnection) int64 {
//	key := standby.dmConnector.host + ":" + strconv.Itoa(int(standby.dmConnector.port))
//	ret, ok := rwc.standbyNTrxMap[key]
//	if !ok {
//		ret = 0
//	}
//
//	return ret
//}

func (rwc *rwCounter) getStandbyId(standby *DmConnection) int32 {
	key := standby.dmConnector.host + ":" + strconv.Itoa(int(standby.dmConnector.port))
	sid, ok := rwc.standbyIdMap[key]
	if !ok {
		sid = int32(len(rwc.standbyIdMap) + 1) // 下标0是primary
		if sid > rwc.standbyCount {
			// 不在有效备库中
			return -1
		}
		rwc.standbyIdMap[key] = sid
	}
	return sid
}

func (rwc *rwCounter) getStandbyFlag(standby *DmConnection) int32 {
	sid := rwc.getStandbyId(standby)
	if sid > 0 && sid < int32(len(rwc.flag)) {
		// 保证备库有效
		return rwc.flag[sid]
	}
	return 0
}

func (rwc *rwCounter) increaseStandbyNtrx(standby *DmConnection) {
	key := standby.dmConnector.host + ":" + strconv.Itoa(int(standby.dmConnector.port))
	ret, ok := rwc.standbyNTrxMap[key]
	if ok {
		ret += 1
	} else {
		ret = 1
	}
	rwc.standbyNTrxMap[key] = ret
	sid, ok := rwc.standbyIdMap[key]
	if !ok {
		sid = int32(len(rwc.standbyIdMap) + 1) // 下标0是primary
		rwc.standbyIdMap[key] = sid
	}
	rwc.flag[sid]--
	rwc.ntrx_total++
}

func (rwc *rwCounter) random(rowCount int32) int32 {
	rand.Seed(time.Now().UnixNano())
	if rowCount > rwc.standbyCount {
		return rand.Int31n(rwc.standbyCount)
	} else {
		return rand.Int31n(rowCount)
	}
}

func (rwc *rwCounter) String() string {
	return "PERCENT(P/S) : " + strconv.FormatFloat(rwc.primaryPercent, 'f', -1, 64) + "/" + strconv.FormatFloat(rwc.standbyPercent, 'f', -1, 64) + "\nNTRX_PRIMARY : " +
		strconv.FormatInt(rwc.ntrx_primary, 10) + "\nNTRX_TOTAL : " + strconv.FormatInt(rwc.ntrx_total, 10) + "\nNTRX_STANDBY : "
}
