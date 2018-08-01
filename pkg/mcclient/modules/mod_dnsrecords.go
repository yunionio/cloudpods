package modules

var (
	DNSRecords ResourceManager
)

func init() {
	DNSRecords = NewComputeManager("dnsrecord", "dnsrecords",
		[]string{"ID", "Name", "Records", "TTL", "is_public"},
		[]string{})

	registerCompute(&DNSRecords)
}
