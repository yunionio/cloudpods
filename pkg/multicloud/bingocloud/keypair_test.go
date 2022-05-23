package bingocloud

import (
	"testing"
)

func TestSRegion_lookUpKeypair(t *testing.T) {
	cfg := &BingoCloudConfig{
		endpoint:  "http://10.1.240.199",
		accessKey: "71F5215C74935CB779C4",
		secretKey: "WzM1QUIxRjY0QTg5REE5MjREMENBRjBDNkEwQUQw",
	}
	client := &SBingoCloudClient{BingoCloudConfig: cfg}

	region := SRegion{client: client}
	publicKey := "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQDWuAJxkWv6dnij/6Qkj4EQT3V8U10qMkgu9PKdoIHPNA/WLxa3i8a1xy69bPMRBYrf8kVvCXB4lTMVmmzaDNBo+DW2qvWOZlCAt/TjaJ2IyHndnUIbNlzYBw2q7qomsOxbulQo+EuyVrTdiI/jr8Aus+1g8TfEBG5sFGhDtvOcb0jTR6hX/muJ2eZqKoTD8vR+HSmUPuRQTEiX2WYOaL8GWqPs8j0sozTtfLHW0OnE7faEw/7gkrjp9BjcxWk1+hZf71FdIpEV+wj+UUj3cKHDQK7zHmuu/7rWNPFc0FaGt3rORFffTl5grG6eUiSvg7QXfu0kX0+hmSDhltF+WWIz david@LAPTOP-M4N1EO91"
	s, err := region.lookUpKeypair(publicKey)
	if err != nil {
		t.Fatal(err)
	}
	t.Log(s)
}
