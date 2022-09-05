package utils

import (
	"testing"

	api "yunion.io/x/onecloud/pkg/apis/compute"
)

func TestGenerateTelegrafConf(t *testing.T) {
	type args struct {
		serverDetails *api.ServerDetails
		influxdbUrl   string
		osType        string
		serverType    string
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		{
			name:    "test_empty",
			args:    args{new(api.ServerDetails), "", "Linux", "guest"},
			want:    "",
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GenerateTelegrafConf(tt.args.serverDetails, tt.args.influxdbUrl, tt.args.osType, tt.args.serverType)
			if (err != nil) != tt.wantErr {
				t.Errorf("GenerateTelegrafConf() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			t.Logf("got %s", got)
		})
	}
}
