package skus

import (
	"reflect"
	"testing"
)

func Test_diff(t *testing.T) {
	type args struct {
		origins  []string
		compares []string
	}
	tests := []struct {
		name string
		args args
		want []string
	}{
		{
			name: "Test array diff",
			args: args{
				origins:  []string{"1", "2", "3"},
				compares: []string{"2", "3", "5"},
			},
			want: []string{"1"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := diff(tt.args.origins, tt.args.compares); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("diff() = %v, want %v", got, tt.want)
			}
		})
	}
}
