package redfish

import "time"

type SCdromInfo struct {
	Image         string `json:"Image"`
	SupportAction bool   `json:"SupportAction"`
}

type SSystemInfo struct {
	Manufacturer string   `json:"Manufacturer"`
	Model        string   `json:"Model"`
	SKU          string   `json:"SKU"`
	SerialNumber string   `json:"SerialNumber"`
	UUID         string   `json:"UUID"`
	EthernetNICs []string `json:"EthernetNICs"`
	MemoryGB     int      `json:"MemoryGB"`
	NodeCount    int      `json:"NodeCount"`
	CpuDesc      string   `json:"CpuDesc"`
	PowerState   string   `json:"PowerState"`
	NextBootDev  string   `json:"NextBootDev"`

	NextBootDevSupported []string `json:"NextBootDevSupported"`
	ResetTypeSupported   []string `json:"ResetTypeSupported"`
}

type SEvent struct {
	Created  time.Time `json:"Created"`
	Message  string    `json:"Message"`
	Severity string    `json:"Severity"`
}

type SBiosInfo struct {
}

type SPower struct {
	PowerCapacityWatts int `json:"PowerCapacityWatts"`
	PowerConsumedWatts int `json:"PowerConsumedWatts"`
	PowerMetrics       struct {
		AverageConsumedWatts int `json:"AverageConsumedWatts"`
		IntervalInMin        int `json:"IntervalInMin"`
		MaxConsumedWatts     int `json:"MaxConsumedWatts"`
		MinConsumedWatts     int `json:"MinConsumedWatts"`
	} `json:"PowerMetrics"`
}

type STemperature struct {
	Name            string `help:"Name"`
	PhysicalContext string `json:"PhysicalContext"`
	ReadingCelsius  int    `json:"ReadingCelsius"`
}

type SNTPConf struct {
	NTPServers      []string `json:"NTPServers,allowempty"`
	ProtocolEnabled bool     `json:"ProtocolEnabled,allowfalse"`
	TimeZone        string   `json:"TimeZone"`
}
