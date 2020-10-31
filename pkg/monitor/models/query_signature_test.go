package models

import (
	"testing"

	"yunion.io/x/jsonutils"
)

func Test_digestQuerySignature(t *testing.T) {
	dataNoSig, _ := jsonutils.Parse([]byte(`{"name": "test"}`))
	dataWithSig, _ := jsonutils.Parse([]byte(`{"name": "test", "signature": "xxx"}`))

	tests := []struct {
		name string
		data *jsonutils.JSONDict
		want string
	}{
		{
			name: "data no signature",
			data: dataNoSig.(*jsonutils.JSONDict),
			want: "7d9fd2051fc32b32feab10946fab6bb91426ab7e39aa5439289ed892864aa91d",
		},
		{
			name: "data with signature",
			data: dataWithSig.(*jsonutils.JSONDict),
			want: "7d9fd2051fc32b32feab10946fab6bb91426ab7e39aa5439289ed892864aa91d",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := digestQuerySignature(tt.data); got != tt.want {
				t.Errorf("sumQuerySignature() = %v, want %v", got, tt.want)
			}
		})
	}
}
