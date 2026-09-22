// Copyright 2019 Yunion
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package models

import (
	"reflect"
	"testing"

	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/apis"
	api "yunion.io/x/onecloud/pkg/apis/compute"
	imageapi "yunion.io/x/onecloud/pkg/apis/image"
)

func newContainerImageObj(command, args []string, envs []*apis.ContainerKeyValue) jsonutils.JSONObject {
	obj := jsonutils.NewDict()
	if command != nil {
		obj.Set("command", jsonutils.Marshal(command))
	}
	if args != nil {
		obj.Set("args", jsonutils.Marshal(args))
	}
	if envs != nil {
		obj.Set("envs", jsonutils.Marshal(envs))
	}
	return obj
}

func newContainerSpec(command, args []string, envs []*apis.ContainerKeyValue) *api.ContainerSpec {
	spec := &api.ContainerSpec{}
	spec.Command = command
	spec.Args = args
	spec.Envs = envs
	return spec
}

func envPairs(envs []*apis.ContainerKeyValue) [][2]string {
	ret := make([][2]string, 0, len(envs))
	for _, env := range envs {
		if env == nil {
			ret = append(ret, [2]string{})
			continue
		}
		ret = append(ret, [2]string{env.Key, env.Value})
	}
	return ret
}

// the container_image configures everything, the container sets nothing
func TestApplyContainerImageDefaultsEmptySpec(t *testing.T) {
	image := newContainerImageObj(
		[]string{"/bin/sh", "-c"},
		[]string{"echo", "hello"},
		[]*apis.ContainerKeyValue{{Key: "IMAGE_ENV", Value: "1"}},
	)
	spec := newContainerSpec(nil, nil, nil)
	if err := applyContainerImageDefaults(spec, image); err != nil {
		t.Fatalf("applyContainerImageDefaults: %v", err)
	}
	if !reflect.DeepEqual(spec.Command, []string{"/bin/sh", "-c"}) {
		t.Fatalf("unexpected command %#v", spec.Command)
	}
	if !reflect.DeepEqual(spec.Args, []string{"echo", "hello"}) {
		t.Fatalf("unexpected args %#v", spec.Args)
	}
	want := [][2]string{{"IMAGE_ENV", "1"}}
	if got := envPairs(spec.Envs); !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected envs %#v", got)
	}
}

// command/args of the container_image win over whatever the container sets
func TestApplyContainerImageDefaultsImageWins(t *testing.T) {
	// both configured on the image: both override the container values
	image := newContainerImageObj([]string{"/bin/sh"}, []string{"-c", "date"}, nil)
	spec := newContainerSpec([]string{"bash"}, []string{"run"}, nil)
	if err := applyContainerImageDefaults(spec, image); err != nil {
		t.Fatalf("applyContainerImageDefaults: %v", err)
	}
	if !reflect.DeepEqual(spec.Command, []string{"/bin/sh"}) {
		t.Fatalf("command should come from the image, got %#v", spec.Command)
	}
	if !reflect.DeepEqual(spec.Args, []string{"-c", "date"}) {
		t.Fatalf("args should come from the image, got %#v", spec.Args)
	}

	// only command configured on the image: args of the container are kept
	image = newContainerImageObj([]string{"/bin/sh"}, nil, nil)
	spec = newContainerSpec([]string{"bash"}, []string{"run"}, nil)
	if err := applyContainerImageDefaults(spec, image); err != nil {
		t.Fatalf("applyContainerImageDefaults: %v", err)
	}
	if !reflect.DeepEqual(spec.Command, []string{"/bin/sh"}) {
		t.Fatalf("command should come from the image, got %#v", spec.Command)
	}
	if !reflect.DeepEqual(spec.Args, []string{"run"}) {
		t.Fatalf("args should be kept since the image has none, got %#v", spec.Args)
	}

	// only args configured on the image: command of the container is kept
	image = newContainerImageObj(nil, []string{"-c", "date"}, nil)
	spec = newContainerSpec([]string{"bash"}, []string{"run"}, nil)
	if err := applyContainerImageDefaults(spec, image); err != nil {
		t.Fatalf("applyContainerImageDefaults: %v", err)
	}
	if !reflect.DeepEqual(spec.Command, []string{"bash"}) {
		t.Fatalf("command should be kept since the image has none, got %#v", spec.Command)
	}
	if !reflect.DeepEqual(spec.Args, []string{"-c", "date"}) {
		t.Fatalf("args should come from the image, got %#v", spec.Args)
	}
}

// an image without command/args/envs must not clear what the container set
func TestApplyContainerImageDefaultsEmptyImage(t *testing.T) {
	image := newContainerImageObj(nil, nil, nil)
	spec := newContainerSpec([]string{"a"}, []string{"b"}, []*apis.ContainerKeyValue{{Key: "A", Value: "1"}})
	if err := applyContainerImageDefaults(spec, image); err != nil {
		t.Fatalf("applyContainerImageDefaults: %v", err)
	}
	if !reflect.DeepEqual(spec.Command, []string{"a"}) || !reflect.DeepEqual(spec.Args, []string{"b"}) {
		t.Fatalf("command/args should be kept, got %#v %#v", spec.Command, spec.Args)
	}
	want := [][2]string{{"A", "1"}}
	if got := envPairs(spec.Envs); !reflect.DeepEqual(got, want) {
		t.Fatalf("envs should be kept, got %#v", got)
	}

	// an empty command list on the image means "unset" as well
	image = newContainerImageObj([]string{}, []string{}, nil)
	spec = newContainerSpec([]string{"a"}, []string{"b"}, nil)
	if err := applyContainerImageDefaults(spec, image); err != nil {
		t.Fatalf("applyContainerImageDefaults: %v", err)
	}
	if !reflect.DeepEqual(spec.Command, []string{"a"}) || !reflect.DeepEqual(spec.Args, []string{"b"}) {
		t.Fatalf("command/args should be kept, got %#v %#v", spec.Command, spec.Args)
	}

	// nil image payload is a no-op
	spec = newContainerSpec([]string{"a"}, nil, nil)
	if err := applyContainerImageDefaults(spec, nil); err != nil {
		t.Fatalf("applyContainerImageDefaults(nil): %v", err)
	}
	if !reflect.DeepEqual(spec.Command, []string{"a"}) {
		t.Fatalf("command should be kept, got %#v", spec.Command)
	}
}

// envs are merged: the container keeps its order, the image wins on key conflict
func TestApplyContainerImageDefaultsEnvsMerge(t *testing.T) {
	image := newContainerImageObj(nil, nil, []*apis.ContainerKeyValue{
		{Key: "B", Value: "2-from-image"},
		{Key: "A", Value: "1-from-image"},
		{Key: "D", Value: "4"},
	})
	spec := newContainerSpec(nil, nil, []*apis.ContainerKeyValue{
		{Key: "A", Value: "1-from-container"},
		{Key: "B", Value: "2-from-container"},
		{Key: "C", Value: "3"},
	})
	if err := applyContainerImageDefaults(spec, image); err != nil {
		t.Fatalf("applyContainerImageDefaults: %v", err)
	}
	want := [][2]string{
		{"A", "1-from-image"},
		{"B", "2-from-image"},
		{"C", "3"},
		{"D", "4"},
	}
	if got := envPairs(spec.Envs); !reflect.DeepEqual(got, want) {
		t.Fatalf("got %#v, want %#v", got, want)
	}
}

// applying the same image twice must be a no-op
func TestApplyContainerImageDefaultsIdempotent(t *testing.T) {
	image := newContainerImageObj(
		[]string{"/bin/sh"},
		[]string{"-c", "date"},
		[]*apis.ContainerKeyValue{{Key: "A", Value: "1"}, {Key: "B", Value: "2"}},
	)
	spec := newContainerSpec([]string{"bash"}, nil, []*apis.ContainerKeyValue{{Key: "A", Value: "x"}})

	if err := applyContainerImageDefaults(spec, image); err != nil {
		t.Fatalf("applyContainerImageDefaults: %v", err)
	}
	first := newContainerSpec(spec.Command, spec.Args, spec.Envs)
	if err := applyContainerImageDefaults(spec, image); err != nil {
		t.Fatalf("applyContainerImageDefaults: %v", err)
	}
	if !reflect.DeepEqual(spec.Command, first.Command) ||
		!reflect.DeepEqual(spec.Args, first.Args) ||
		!reflect.DeepEqual(envPairs(spec.Envs), envPairs(first.Envs)) {
		t.Fatalf("not idempotent: %#v vs %#v", spec, first)
	}
}

func TestApplyContainerImageDefaultsEnvsValueFrom(t *testing.T) {
	// image envs carrying a value_from must survive the merge, and must override
	// a plain value set by the container
	image := newContainerImageObj(nil, nil, []*apis.ContainerKeyValue{
		{
			Key: "SECRET",
			ValueFrom: &apis.ContainerValueSource{
				Credential: &apis.ContainerValueSourceCredential{Id: "cred-1", Key: "password"},
			},
		},
	})
	spec := newContainerSpec(nil, nil, []*apis.ContainerKeyValue{{Key: "SECRET", Value: "plain"}})
	if err := applyContainerImageDefaults(spec, image); err != nil {
		t.Fatalf("applyContainerImageDefaults: %v", err)
	}
	if len(spec.Envs) != 1 || spec.Envs[0].Key != "SECRET" {
		t.Fatalf("unexpected envs %#v", spec.Envs)
	}
	if spec.Envs[0].ValueFrom == nil || spec.Envs[0].ValueFrom.Credential == nil {
		t.Fatalf("value_from lost: %#v", spec.Envs[0])
	}
	if spec.Envs[0].ValueFrom.Credential.Id != "cred-1" {
		t.Fatalf("unexpected value_from %#v", spec.Envs[0].ValueFrom)
	}
}

// TestContainerImageEnvsSerializable guards the sqlchemy storage assumption:
// ContainerImageEnvs must be registered as a serializable type so that the
// compound column can be read back into the model.
func TestContainerImageEnvsSerializable(t *testing.T) {
	envs := imageapi.ContainerImageEnvs{{Key: "A", Value: "1"}}
	obj, err := jsonutils.JSONDeserialize(reflect.TypeOf(&imageapi.ContainerImageEnvs{}), envs.String())
	if err != nil {
		t.Fatalf("JSONDeserialize: %v", err)
	}
	got, ok := obj.(*imageapi.ContainerImageEnvs)
	if !ok {
		t.Fatalf("unexpected deserialized type %T", obj)
	}
	if len(*got) != 1 || (*got)[0].Key != "A" || (*got)[0].Value != "1" {
		t.Fatalf("unexpected envs %#v", *got)
	}
}

func TestMergeContainerImageEnvs(t *testing.T) {
	cases := []struct {
		name      string
		ctrEnvs   []*apis.ContainerKeyValue
		imageEnvs []*apis.ContainerKeyValue
		want      [][2]string
	}{
		{
			name: "both empty",
			want: [][2]string{},
		},
		{
			name:      "image only",
			imageEnvs: []*apis.ContainerKeyValue{{Key: "A", Value: "1"}},
			want:      [][2]string{{"A", "1"}},
		},
		{
			name:    "ctr only",
			ctrEnvs: []*apis.ContainerKeyValue{{Key: "A", Value: "1"}},
			want:    [][2]string{{"A", "1"}},
		},
		{
			name:      "image overrides on key conflict and appends the missing ones",
			ctrEnvs:   []*apis.ContainerKeyValue{{Key: "A", Value: "1"}, {Key: "B", Value: "2"}},
			imageEnvs: []*apis.ContainerKeyValue{{Key: "B", Value: "3"}, {Key: "C", Value: "4"}},
			want:      [][2]string{{"A", "1"}, {"B", "3"}, {"C", "4"}},
		},
		{
			name:      "container order is preserved",
			ctrEnvs:   []*apis.ContainerKeyValue{{Key: "B", Value: "2"}, {Key: "A", Value: "1"}},
			imageEnvs: []*apis.ContainerKeyValue{{Key: "A", Value: "9"}, {Key: "C", Value: "3"}},
			want:      [][2]string{{"B", "2"}, {"A", "9"}, {"C", "3"}},
		},
		{
			name:      "trimmed key matches",
			ctrEnvs:   []*apis.ContainerKeyValue{{Key: " A ", Value: "1"}},
			imageEnvs: []*apis.ContainerKeyValue{{Key: "A", Value: "2"}},
			want:      [][2]string{{"A", "2"}},
		},
		{
			name:      "empty and nil key skipped",
			ctrEnvs:   []*apis.ContainerKeyValue{nil, {Key: "", Value: "y"}, {Key: "A", Value: "1"}},
			imageEnvs: []*apis.ContainerKeyValue{nil, {Key: "  ", Value: "x"}, {Key: "A", Value: "2"}},
			want:      [][2]string{{"A", "2"}},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := envPairs(mergeContainerImageEnvs(c.ctrEnvs, c.imageEnvs))
			if !reflect.DeepEqual(got, c.want) {
				t.Fatalf("got %#v, want %#v", got, c.want)
			}
		})
	}
}
