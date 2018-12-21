package stringutils

import (
	"reflect"
	"testing"
)

func TestEscapeString(t *testing.T) {
	type args struct {
		str   string
		pairs [][]string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "normalInput",
			args: args{
				str:   "abcd\n\"Te\\rst\"ddd\"$Test\"aaa\n$TTT",
				pairs: nil,
			},
			want: `abcd\n\"Te\\rst\"ddd\"\$Test\"aaa\n\$TTT`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := EscapeString(tt.args.str, tt.args.pairs); got != tt.want {
				t.Errorf("EscapeString() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSplitByQuotation(t *testing.T) {
	type args struct {
		line string
	}
	tests := []struct {
		name    string
		args    args
		want    []string
		wantErr bool
	}{
		{
			name:    "normalInput",
			args:    args{`"abc" addf "sada"`},
			want:    []string{"abc", " addf ", "sada"},
			wantErr: false,
		},
		{
			name:    "errorInput",
			args:    args{`"abc" "addf "sada"`},
			want:    nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := SplitByQuotation(tt.args.line)
			if (err != nil) != tt.wantErr {
				t.Errorf("SplitByQuotation() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("SplitByQuotation() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEscapeEchoString(t *testing.T) {
	type args struct {
		str string
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		{
			name:    "normalInput",
			args:    args{"abcd\n\"Te\\rst\"ddd\"$Test\"aaa\n$TTT"},
			want:    `abcd\n\"Te\\\\rst\"ddd\"\$Test\"aaa\n\$TTT`,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := EscapeEchoString(tt.args.str)
			if (err != nil) != tt.wantErr {
				t.Errorf("EscapeEchoString() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("EscapeEchoString() = %v, want %v", got, tt.want)
			}
		})
	}
}
