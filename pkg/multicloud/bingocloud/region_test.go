package bingocloud

import (
	"testing"
)

func TestSRegion_GetSkus(t *testing.T) {
	cfg := &BingoCloudConfig{
		endpoint:  "http://10.1.240.199",
		accessKey: "71F5215C74935CB779C4",
		secretKey: "WzM1QUIxRjY0QTg5REE5MjREMENBRjBDNkEwQUQw",
	}
	client := &SBingoCloudClient{BingoCloudConfig: cfg}

	region := SRegion{client: client}
	got, err := region.GetSkus("")
	if err != nil {
		t.Fatal(err)
	}
	for _, sku := range got {
		t.Log(sku.GetCpuCoreCount())
	}
}
