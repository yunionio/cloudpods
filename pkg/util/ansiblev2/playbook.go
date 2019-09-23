package ansiblev2

import (
	"fmt"

	"github.com/go-yaml/yaml"
)

type ITask interface {
	MarshalYAML() (interface{}, error)
}

type Task struct {
	Name          string
	WithPlugin    string
	WithPluginVal interface{}
	When          string
	Register      string
	IgnoreErrors  bool
	Vars          map[string]interface{}

	ModuleName string
	ModuleArgs map[string]interface{}
}

func (t *Task) MarshalYAML() (interface{}, error) {
	r := map[string]interface{}{
		t.ModuleName: t.ModuleArgs,
	}
	if t.Name != "" {
		r["name"] = t.Name
	}
	if t.WithPlugin != "" && t.WithPluginVal != nil {
		r["with_"+t.WithPlugin] = t.WithPluginVal
	}
	if t.When != "" {
		r["when"] = t.When
	}
	if t.Register != "" {
		r["register"] = t.Register
	}
	if t.IgnoreErrors {
		r["ignore_errors"] = "yes"
	}
	if len(t.Vars) > 0 {
		r["vars"] = t.Vars
	}
	return r, nil
}

type ShellTask struct {
	Name          string
	WithPlugin    string
	WithPluginVal interface{}
	When          string
	Register      string
	IgnoreErrors  bool
	Vars          map[string]interface{}

	Script     string
	ModuleArgs map[string]interface{}
}

func (t *ShellTask) MarshalYAML() (interface{}, error) {
	r := map[string]interface{}{
		"shell": t.Script,
	}
	if len(t.ModuleArgs) > 0 {
		r["args"] = t.ModuleArgs
	}

	if t.Name != "" {
		r["name"] = t.Name
	}
	if t.WithPlugin != "" && t.WithPluginVal != nil {
		r["with_"+t.WithPlugin] = t.WithPluginVal
	}
	if t.When != "" {
		r["when"] = t.When
	}
	if t.Register != "" {
		r["register"] = t.Register
	}
	if t.IgnoreErrors {
		r["ignore_errors"] = "yes"
	}
	if len(t.Vars) > 0 {
		r["vars"] = t.Vars
	}
	return r, nil
}

type Block struct {
	Name          string
	WithPlugin    string
	WithPluginVal interface{}
	When          string
	Register      string
	IgnoreErrors  bool
	Vars          map[string]interface{}

	Tasks []ITask
}

func NewBlock(tasks ...ITask) *Block {
	b := &Block{
		Tasks: tasks,
	}
	return b
}

func (b *Block) MarshalYAML() (interface{}, error) {
	r := map[string]interface{}{}
	if len(b.Tasks) > 0 {
		tasks := make([]interface{}, len(b.Tasks))
		for i := range tasks {
			var err error
			tasks[i], err = b.Tasks[i].MarshalYAML()
			if err != nil {
				return nil, err
			}
		}
		r["block"] = tasks
	}

	if b.Name != "" {
		r["name"] = b.Name
	}
	if b.WithPlugin != "" && b.WithPluginVal != nil {
		r["with_"+b.WithPlugin] = b.WithPluginVal
	}
	if b.When != "" {
		r["when"] = b.When
	}
	if b.Register != "" {
		r["register"] = b.Register
	}
	if b.IgnoreErrors {
		r["ignore_errors"] = "yes"
	}
	if len(b.Vars) > 0 {
		r["vars"] = b.Vars
	}
	return r, nil
}

type Play struct {
	Name         string
	RemoteUser   string
	Vars         map[string]interface{}
	IgnoreErrors bool

	Hosts string
	Tasks []ITask
}

func NewPlay(tasks ...ITask) *Play {
	play := &Play{
		Tasks: tasks,
	}
	return play
}

func (play *Play) MarshalYAML() (interface{}, error) {
	if play.Hosts == "" {
		return nil, fmt.Errorf("hosts is required but not set")
	}

	r := map[string]interface{}{
		"hosts": play.Hosts,
	}
	if len(play.Tasks) > 0 {
		tasks := make([]interface{}, len(play.Tasks))
		for i := range tasks {
			var err error
			tasks[i], err = play.Tasks[i].MarshalYAML()
			if err != nil {
				return nil, err
			}
		}
		r["tasks"] = tasks
	}

	if play.Name != "" {
		r["name"] = play.Name
	}
	if play.RemoteUser != "" {
		r["remote_user"] = play.RemoteUser
	}
	if len(play.Vars) > 0 {
		r["vars"] = play.Vars
	}
	if play.IgnoreErrors {
		r["ignore_errors"] = "yes"
	}
	return r, nil
}

type Playbook struct {
	Plays []*Play
}

func NewPlaybook(plays ...*Play) *Playbook {
	pb := &Playbook{
		Plays: plays,
	}
	return pb

}
func (pb *Playbook) MarshalYAML() (interface{}, error) {
	r := make([]interface{}, len(pb.Plays))
	for i := range pb.Plays {
		var err error
		r[i], err = pb.Plays[i].MarshalYAML()
		if err != nil {
			return nil, err
		}
	}
	return r, nil
}

func (pb *Playbook) String() string {
	b, err := yaml.Marshal(pb)
	if err != nil {
		// panic early
		panic(err)
	}
	return string(b)
}
