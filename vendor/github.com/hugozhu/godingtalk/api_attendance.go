/*
 * Author Kevin Zhu
 *
 * Direct questions, comments to <ipandtcp@gmail.com>
 */

package godingtalk

import (
	"errors"
	"time"
)

type Attendance struct {
	GmtModifed      int64   `json:"gmtModified"`    //: 1492594486000,
	IsLegal         string  `json:"isLegal"`        //: "N",
	BaseCheckTime   int64   `json:"baseCheckTime"`  //: 1492568460000,
	ID              int64   `json:"id"`             //: 933202551,
	UserAddress     string  `json:"userAddress"`    //: "北京市朝阳区崔各庄镇阿里中心.望京A座阿里巴巴绿地中心",
	UID             string  `json:"userId"`         //: "manager7078",
	CheckType       string  `json:"checkType"`      //: "OnDuty",
	TimeResult      string  `json:"timeResult"`     //: "Normal",
	DeviceID        string  `json:"deviceId"`       // :"cb7ace07d52fe9be14f4d8bec5e1ba79"
	CorpID          string  `json:"corpId"`         //: "ding7536bfee6fb1fa5a35c2f4657eb6378f",
	SourceType      string  `json:"sourceType"`     //: "USER",
	WorkDate        int64   `json:"workDate"`       //: 1492531200000,
	PlanCheckTime   int64   `json:"planCheckTime"`  //: 1492568497000,
	GmtCreate       int64   `json:"gmtCreate"`      //: 1492594486000,
	LocaltionMethod string  `json:"locationMethod"` //: "MAP",
	LocationResult  string  `json:"locationResult"` //: "Outside",
	UserLongitude   float64 `json:"userLongitude"`  //: 116.486888,
	PlanID          int     `json:"planId"`         //: 4550269081,
	GroupID         int     `json:"groupId"`        //: 121325603,
	UserAccuracy    int     `json:"userAccuracy"`   //: 65,
	UserCheckTime   int64   `json:"userCheckTime"`  //: 1492568497000,
	UserLatitude    float64 `json:"userLatitude"`   //: 39.999946,
	ProcInstID      string  `json:"procInstId"`     //: "cb992267-9b70"
	ApproveID       int     `json:"approveId"`      //         string, `json:""`//关联的审批id
	ClassId         int     `json:"classId"`        //考勤班次id，没有的话表示该次打卡不在排班内
	UserSsid        string  `json:"userSsid"`       //用户打卡wifi SSID
	UserMacAddr     string  `json:"userMacAddr"`    //用户打卡wifi Mac地址
	BaseAddress     string  `json:"baseAddress"`    //基准地址
	BaseLongitude   float32 `json:"baseLongitude"`  //    基准经度
	BaseLatitude    float32 `json:"baseLatitude"`   //   基准纬度
	BaseAccuracy    int     `json:"baseAccuracy"`   //  基准定位精度
	BaseSsid        string  `json:"baseSsid"`       //基准wifi ssid
	BaseMacAddr     string  `json:"baseMacAddr"`    //基准 Mac 地址
	OutsideRemark   string  `json:"outsideRemark"`  //打卡备注
}

type listAttendanceRecordResp struct {
	OAPIResponse
	Records []Attendance `json:"recordresult"`
}

// 获取所有的打卡记录，该员工当天如果打卡10条，那么10条都将返回
func (c *DingTalkClient) ListAttendanceRecord(ulist []string, dateFrom time.Time, dateTo time.Time) ([]Attendance, error) {
	var resp listAttendanceRecordResp
	if len(ulist) > 50 || len(ulist) < 1 {
		return nil, errors.New("Users can't more than 50 or less than 1")
	}

	if !dateFrom.Before(dateTo) {
		return nil, errors.New("FromDate must before ToDate")
	}
	if time.Duration(dateTo.UnixNano()-dateFrom.UnixNano()).Hours() > float64(7*24) {
		return nil, errors.New("Can't more than 6 days at once")
	}

	request := map[string]interface{}{
		"checkDateFrom": dateFrom.Format("2006-01-02 15:04:05"), // "yyyy-MM-dd hh:mm:ss",
		"checkDateTo":   dateTo.Format("2006-01-02 15:04:05"),   // "yyyy-MM-dd hh:mm:ss",
		"userIds":       ulist,                                  // 企业内的员工id列表，最多不能超过50个
	}
	return resp.Records, c.httpRPC("/attendance/listRecord", nil, request, &resp)
}

type listAttendanceResultResp struct {
	OAPIResponse
	HasMore bool         `json:"hasMore"`
	Records []Attendance `json:"recordresult"`
}

// 即使员工在这期间打了多次，该接口也只会返回两条记录，包括上午的打卡结果和下午的打卡结果
// 用户如果为空则获取所有用户
func (c *DingTalkClient) ListAttendanceResult(ulist []string, dateFrom, dateTo time.Time, offset, lmt int64) (listAttendanceResultResp, error) {
	var resp listAttendanceResultResp
	if time.Duration(dateTo.UnixNano()-dateFrom.UnixNano()).Hours() > float64(7*24) {
		return resp, errors.New("Can't more than 7 days at once")
	}

	if !dateFrom.Before(dateTo) {
		return resp, errors.New("FromDate must before ToDate")
	}

	request := map[string]interface{}{
		"workDateFrom": dateFrom.Format("2006-01-02 15:04:05"), // "yyyy-MM-dd hh:mm:ss",
		"workDateTo":   dateTo.Format("2006-01-02 15:04:05"),   // "yyyy-MM-dd hh:mm:ss",
		"userIdList":   ulist,                                  // ["员工UserId列表"], 必填，与offset和limit配合使用，不传表示分页获取全员的数据
		"offset":       offset,                                 // 必填，第一次传0，如果还有多余数据，下次传之前的offset加上limit的值
		"limit":        lmt,                                    // 最多50
	}
	return resp, c.httpRPC("/attendance/list", nil, request, &resp)
}
