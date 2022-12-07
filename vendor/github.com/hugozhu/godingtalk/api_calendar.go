package godingtalk

import "time"

type Event struct {
	OAPIResponse
	Id          string
	Location    string
	Summary     string
	Description string
	Start       struct {
		DateTime string `json:"date_time"`
	}
	End struct {
		DateTime string `json:"date_time"`
	}
}

type ListEventsResponse struct {
	OAPIResponse
	Success bool `json:"success"`
	Result  struct {
		Events        []Event `json:"items"`
		Summary       string  `json:"summary"`
		NextPageToken string  `json:"next_page_token"`
	} `json:"result"`
}
type CalendarTime struct {
	TimeZone string `json:"time_zone"`
	Date     string `json:"date_time"`
}

type CalendarRequest struct {
	TimeMax CalendarTime `json:"time_max"`
	TimeMin CalendarTime `json:"time_min"`
	StaffId string       `json:"user_id"`
}

func (c *DingTalkClient) ListEvents(staffid string, from time.Time, to time.Time) (events []Event, err error) {
	location := time.Now().Location().String()
	timeMin := CalendarTime{
		TimeZone: location,
		Date:     from.Format("2006-01-02T15:04:05Z0700"),
	}
	timeMax := CalendarTime{
		TimeZone: location,
		Date:     to.Format("2006-01-02T15:04:05Z0700"),
	}

	data := CalendarRequest{
		TimeMax: timeMax,
		TimeMin: timeMin,
		StaffId: staffid,
	}
	var resp ListEventsResponse
	err = c.httpRPC("topapi/calendar/list", nil, data, &resp)
	events = resp.Result.Events
	return events, err
}
