package score

import (
	"testing"

	"yunion.io/x/pkg/tristate"
)

func TestLess(t *testing.T) {
	type args struct {
		b1       *ScoreBucket
		b2       *ScoreBucket
		lessFunc func(s1, s2 *ScoreBucket) bool
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "equal",
			args: args{
				b1:       NewScoreBuckets(),
				b2:       NewScoreBuckets(),
				lessFunc: NormalLess,
			},
			want: false,
		},
		{
			name: "less",
			args: args{
				b1:       NewScoreBuckets().SetScore(SScore{1, "p1"}, tristate.True),
				b2:       NewScoreBuckets().SetScore(SScore{1, "p1"}, tristate.True).SetScore(SScore{2, "p2"}, tristate.True),
				lessFunc: PreferLess,
			},
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.args.lessFunc(tt.args.b1, tt.args.b2); got != tt.want {
				t.Errorf("Less() = %v, want %v", got, tt.want)
			}
		})
	}
}
