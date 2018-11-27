package vmdkutils

import "testing"

const (
	VMDKContent = `    version=1
    encoding="UTF-8"
    CID=ab798ecb
    parentCID=ffffffff
    isNativeSnapshot="no"
    createType="vmfs"

    # Extent description
    RW 62914560 VMFS "c8336ec2-3885-4205-a2f1-095e48228a56-flat.vmdk"

    # The Disk Data Base
    #DDB

    ddb.adapterType = "lsilogic"
    ddb.deletable = "false"
    ddb.geometry.cylinders = "62415"
    ddb.geometry.heads = "16"
    ddb.geometry.sectors = "63"
    ddb.longContentID = "4d1762d22407e6b7e3e63557ab798ecb"
    ddb.thinProvisioned = "1"
    ddb.uuid = "60 00 C2 97 32 7d ac 2b-ac 97 a4 95 5b a6 0b c0"
    ddb.virtualHWVersion = "13"
`
)

func TestParseStream(t *testing.T) {
	info, err := Parse(VMDKContent)
	if err != nil {
		t.Errorf("parse error %s", err)
	}
	t.Logf("%#v", info)

	_, err = Parse("")
	if err != nil {
		t.Logf("parse error %s", err)
	} else {
		t.Errorf("should parse error")
	}
}
