package userdata

import "testing"

func TestEncodeDecode(t *testing.T) {
	tests := []struct {
		name     string
		userdata string
	}{
		{
			name:     "equal",
			userdata: "1",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			enGot, err := Encode(tt.userdata)
			if err != nil {
				t.Errorf("Encode() error = %v", err)
				return
			}

			deGot, err := Decode(enGot)
			if err != nil {
				t.Errorf("Decode() error = %v", err)
				return
			}
			if tt.userdata != deGot {
				t.Errorf("decode after encode userdata = %v want %v", deGot, tt.userdata)
			}
		})
	}
}
