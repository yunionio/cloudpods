package models

import (
	"testing"

	api "yunion.io/x/onecloud/pkg/apis/llm"
)

func TestSkuHasDataDisk(t *testing.T) {
	if SkuHasDataDisk(nil) {
		t.Fatal("nil sku must not have a data disk")
	}
	if SkuHasDataDisk(&SLLMSku{}) {
		t.Fatal("sku without volumes must not have a data disk")
	}
	empty := api.Volumes{}
	sku := &SLLMSku{}
	sku.Volumes = &empty
	if SkuHasDataDisk(sku) {
		t.Fatal("sku with empty volumes must not have a data disk")
	}
	vols := api.Volumes{{SizeMB: 10240}}
	withDisk := &SLLMSku{}
	withDisk.Volumes = &vols
	if !SkuHasDataDisk(withDisk) {
		t.Fatal("sku with volumes must have a data disk")
	}
}
