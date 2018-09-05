package command

import (
	"reflect"
	"strings"
	"testing"
)

func TestKubectlExec_Command(t *testing.T) {
	type fields struct {
		Kubectl *Kubectl
	}
	type args struct {
		cmd  string
		args []string
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   string
	}{
		{
			name: "bash command",
			fields: fields{
				Kubectl: NewKubectlCommand("/tmp/kubeconfig", "system"),
			},
			args: args{
				cmd:  "bash",
				args: []string{"-il"},
			},
			want: "kubectl --namespace system exec -i -t Pod1 -c Container1 -- bash -il",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := tt.fields.Kubectl.Exec().
				Stdin().TTY().
				Pod("Pod1").
				Container("Container1")
			cmd := c.Command(tt.args.cmd, tt.args.args...)
			name := cmd.name
			args := cmd.args
			got := []string{name}
			got = append(got, args...)
			if !reflect.DeepEqual(strings.Join(got, " "), tt.want) {
				t.Errorf("KubectlExec.Command() = %v, want %v", got, tt.want)
			}
		})
	}
}
