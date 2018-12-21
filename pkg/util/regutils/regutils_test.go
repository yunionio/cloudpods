package regutils

import (
	"reflect"
	"testing"
)

func TestSubGroupMatch(t *testing.T) {
	type args struct {
		pattern string
		line    string
	}
	tests := []struct {
		name string
		args args
		want map[string]string
	}{
		{
			name: "normalInput",
			args: args{
				pattern: `(?P<idx>\d+)\s+(?P<start>\d+)s\s+(?P<end>\d+)s\s+(?P<count>\d+)s`,
				line:    `1      2048s       314984447s  314982400s  ntfs            Basic data partition  msftdata`,
			},
			want: map[string]string{
				"idx":   "1",
				"start": "2048",
				"end":   "314984447",
				"count": "314982400",
			},
		},
		{
			name: "emptyInput",
			args: args{
				pattern: `%s+`,
				line:    ``,
			},
			want: map[string]string{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := SubGroupMatch(tt.args.pattern, tt.args.line); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("SubGroupMatch() = %v, want %v", got, tt.want)
			}
		})
	}
}
