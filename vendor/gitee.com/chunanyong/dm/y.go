/*
 * Copyright (c) 2000-2018, 达梦数据库有限公司.
 * All rights reserved.
 */

package dm

import (
	"bytes"
	"math/rand"
	"sync"
	"time"

	"gitee.com/chunanyong/dm/util"
)

/**
 * dm_svc.conf中配置的服务名对应的一组实例, 以及相关属性和状态信息
 *
 * 需求：
 * 1. 连接均匀分布在各个节点上
 * 2. loginMode，loginStatus匹配
 * 3. 连接异常节点比较耗时，在DB列表中包含异常节点时异常连接尽量靠后，减少对建连接速度的影响
 *
 *
 * DB 连接顺序：
 * 1. well distribution，每次连接都从列表的下一个节点开始
 * 2. 用DB sort值按从大到小排序，sort为一个四位数XXXX，个位--serverStatus，十位--serverMode，共 有三种模式，最优先的 *100, 次优先的*10
 */
type epGroup struct {
	name       string
	epList     []*ep
	props      *Properties
	epStartPos int32 // wellDistribute 起始位置
	lock       sync.Mutex
}

func newEPGroup(name string, serverList []*ep) *epGroup {
	g := new(epGroup)
	g.name = name
	g.epList = serverList
	if serverList == nil || len(serverList) == 0 {
		g.epStartPos = -1
	} else {
		// 保证进程间均衡，起始位置采用随机值
		g.epStartPos = rand.Int31n(int32(len(serverList))) - 1
	}
	return g
}

func (g *epGroup) connect(connector *DmConnector) (*DmConnection, error) {
	var dbSelector = g.getEPSelector(connector)
	var ex error = nil
	// 如果配置了loginMode的主、备等优先策略，而未找到最高优先级的节点时持续循环switchtimes次，如果最终还是没有找到最高优先级则选择次优先级的
	// 如果只有一个节点，一轮即可决定是否连接；多个节点时保证switchTimes轮尝试，最后一轮决定用哪个节点（由于节点已经按照模式优先级排序，最后一轮理论上就是连第一个节点）
	var cycleCount int32
	if len(g.epList) == 1 {
		cycleCount = 1
	} else {
		cycleCount = connector.switchTimes + 1
	}
	for i := int32(0); i < cycleCount; i++ {
		// 循环了一遍，如果没有符合要求的, 重新排序, 再尝试连接
		conn, err := g.traverseServerList(connector, dbSelector, i == 0, i == cycleCount-1)
		if err != nil {
			ex = err
			time.Sleep(time.Duration(connector.switchInterval) * time.Millisecond)
			continue
		}
		return conn, nil
	}
	return nil, ex
}

func (g *epGroup) getEPSelector(connector *DmConnector) *epSelector {
	if connector.epSelector == TYPE_HEAD_FIRST {
		return newEPSelector(g.epList)
	} else {
		serverCount := int32(len(g.epList))
		sortEPs := make([]*ep, serverCount)
		g.lock.Lock()
		defer g.lock.Unlock()
		g.epStartPos = (g.epStartPos + 1) % serverCount
		for i := int32(0); i < serverCount; i++ {
			sortEPs[i] = g.epList[(i+g.epStartPos)%serverCount]
		}
		return newEPSelector(sortEPs)
	}
}

/**
* 从指定编号开始，遍历一遍服务名中的ip列表，只连接指定类型（主机或备机）的ip
* @param servers
* @param checkTime
*
* @exception
* DBError.ECJDBC_INVALID_SERVER_MODE 有站点的模式不匹配
* DBError.ECJDBC_COMMUNITION_ERROR 所有站点都连不上
 */
func (g *epGroup) traverseServerList(connector *DmConnector, epSelector *epSelector, first bool, last bool) (*DmConnection, error) {
	epList := epSelector.sortDBList(first)
	errorMsg := bytes.NewBufferString("")
	var ex error = nil // 第一个错误
	for _, server := range epList {
		conn, err := server.connect(connector)
		if err != nil {
			if ex == nil {
				ex = err
			}
			errorMsg.WriteString("[")
			errorMsg.WriteString(server.String())
			errorMsg.WriteString("]")
			errorMsg.WriteString(err.Error())
			errorMsg.WriteString(util.StringUtil.LineSeparator())
			continue
		}
		valid, err := epSelector.checkServerMode(conn, last)
		if err != nil {
			if ex == nil {
				ex = err
			}
			errorMsg.WriteString("[")
			errorMsg.WriteString(server.String())
			errorMsg.WriteString("]")
			errorMsg.WriteString(err.Error())
			errorMsg.WriteString(util.StringUtil.LineSeparator())
			continue
		}
		if !valid {
			conn.close()
			err = ECGO_INVALID_SERVER_MODE.throw()
			if ex == nil {
				ex = err
			}
			errorMsg.WriteString("[")
			errorMsg.WriteString(server.String())
			errorMsg.WriteString("]")
			errorMsg.WriteString(err.Error())
			errorMsg.WriteString(util.StringUtil.LineSeparator())
			continue
		}
		return conn, nil
	}
	if ex != nil {
		return nil, ex
	}
	return nil, ECGO_COMMUNITION_ERROR.addDetail(errorMsg.String()).throw()
}
