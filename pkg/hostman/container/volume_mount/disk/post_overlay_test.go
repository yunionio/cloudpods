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

package disk

import (
	"sort"
	"testing"

	"yunion.io/x/onecloud/pkg/apis"
)

func TestGetMinPathMapKeyLength(t *testing.T) {
	tests := []struct {
		name     string
		ov       *apis.ContainerVolumeMountDiskPostOverlay
		expected int
	}{
		{
			name: "Image is nil",
			ov: &apis.ContainerVolumeMountDiskPostOverlay{
				Image: nil,
			},
			expected: 1 << 30,
		},
		{
			name: "PathMap is empty",
			ov: &apis.ContainerVolumeMountDiskPostOverlay{
				Image: &apis.ContainerVolumeMountDiskPostImageOverlay{
					Id:      "test-id",
					PathMap: make(map[string]string),
				},
			},
			expected: 1 << 30,
		},
		{
			name: "Single path in PathMap",
			ov: &apis.ContainerVolumeMountDiskPostOverlay{
				Image: &apis.ContainerVolumeMountDiskPostImageOverlay{
					Id: "test-id",
					PathMap: map[string]string{
						".steam": "/home/.steam",
					},
				},
			},
			expected: 6, // len(".steam")
		},
		{
			name: "Multiple paths in PathMap - shortest first",
			ov: &apis.ContainerVolumeMountDiskPostOverlay{
				Image: &apis.ContainerVolumeMountDiskPostImageOverlay{
					Id: "test-id",
					PathMap: map[string]string{
						".steam/debian-installation/steamapps/appmanifest_553420.acf": "/home/.steam/debian-installation/steamapps/appmanifest_553420.acf",
						".steam": "/home/.steam",
					},
				},
			},
			expected: 6, // len(".steam") is the shortest
		},
		{
			name: "Multiple paths in PathMap - shortest in middle",
			ov: &apis.ContainerVolumeMountDiskPostOverlay{
				Image: &apis.ContainerVolumeMountDiskPostImageOverlay{
					Id: "test-id",
					PathMap: map[string]string{
						".steam/debian-installation/steamapps/appmanifest_553420.acf": "/home/.steam/debian-installation/steamapps/appmanifest_553420.acf",
						".steam":                     "/home/.steam",
						".steam/debian-installation": "/home/.steam/debian-installation",
					},
				},
			},
			expected: 6, // len(".steam") is the shortest
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getMinPathMapKeyLength(tt.ov)
			if result != tt.expected {
				t.Errorf("getMinPathMapKeyLength() = %d, want %d", result, tt.expected)
			}
		})
	}
}

func TestMountPostOverlaysSorting(t *testing.T) {
	// 创建测试数据：路径长的在前，路径短的在后
	ovs := []*apis.ContainerVolumeMountDiskPostOverlay{
		{
			ContainerTargetDir: "long-path",
			Image: &apis.ContainerVolumeMountDiskPostImageOverlay{
				Id: "image-1",
				PathMap: map[string]string{
					".steam/debian-installation/steamapps/appmanifest_553420.acf": "/home/.steam/debian-installation/steamapps/appmanifest_553420.acf",
				},
			},
		},
		{
			ContainerTargetDir: "short-path",
			Image: &apis.ContainerVolumeMountDiskPostImageOverlay{
				Id: "image-2",
				PathMap: map[string]string{
					".steam": "/home/.steam",
				},
			},
		},
	}

	// 验证排序前的顺序
	if getMinPathMapKeyLength(ovs[0]) < getMinPathMapKeyLength(ovs[1]) {
		t.Fatal("Test setup error: ovs should be in reverse order initially")
	}

	// 手动执行排序逻辑（模拟 mountPostOverlays 中的排序）
	sort.Slice(ovs, func(i, j int) bool {
		return getMinPathMapKeyLength(ovs[i]) < getMinPathMapKeyLength(ovs[j])
	})

	// 验证排序后的顺序：路径短的应该在前面
	if getMinPathMapKeyLength(ovs[0]) >= getMinPathMapKeyLength(ovs[1]) {
		t.Errorf("Sorting failed: expected shorter path first, got lengths %d and %d",
			getMinPathMapKeyLength(ovs[0]), getMinPathMapKeyLength(ovs[1]))
	}

	// 验证具体路径
	if ovs[0].ContainerTargetDir != "short-path" {
		t.Errorf("Expected 'short-path' first, got '%s'", ovs[0].ContainerTargetDir)
	}
	if ovs[1].ContainerTargetDir != "long-path" {
		t.Errorf("Expected 'long-path' second, got '%s'", ovs[1].ContainerTargetDir)
	}
}

func TestMountPostOverlaysSortingWithNilImage(t *testing.T) {
	// 测试包含 nil Image 的情况
	ovs := []*apis.ContainerVolumeMountDiskPostOverlay{
		{
			ContainerTargetDir: "no-image",
			Image:              nil,
		},
		{
			ContainerTargetDir: "short-path",
			Image: &apis.ContainerVolumeMountDiskPostImageOverlay{
				Id: "image-1",
				PathMap: map[string]string{
					".steam": "/home/.steam",
				},
			},
		},
	}

	// 手动执行排序逻辑
	sort.Slice(ovs, func(i, j int) bool {
		return getMinPathMapKeyLength(ovs[i]) < getMinPathMapKeyLength(ovs[j])
	})

	// 验证：有 Image 的应该排在 nil Image 的前面
	if ovs[0].ContainerTargetDir != "short-path" {
		t.Errorf("Expected 'short-path' first, got '%s'", ovs[0].ContainerTargetDir)
	}
	if ovs[1].ContainerTargetDir != "no-image" {
		t.Errorf("Expected 'no-image' second, got '%s'", ovs[1].ContainerTargetDir)
	}
}

func TestMountPostOverlaysSortingWithEmptyPathMap(t *testing.T) {
	// 测试包含空 PathMap 的情况
	ovs := []*apis.ContainerVolumeMountDiskPostOverlay{
		{
			ContainerTargetDir: "empty-pathmap",
			Image: &apis.ContainerVolumeMountDiskPostImageOverlay{
				Id:      "image-1",
				PathMap: make(map[string]string),
			},
		},
		{
			ContainerTargetDir: "short-path",
			Image: &apis.ContainerVolumeMountDiskPostImageOverlay{
				Id: "image-2",
				PathMap: map[string]string{
					".steam": "/home/.steam",
				},
			},
		},
	}

	// 手动执行排序逻辑
	sort.Slice(ovs, func(i, j int) bool {
		return getMinPathMapKeyLength(ovs[i]) < getMinPathMapKeyLength(ovs[j])
	})

	// 验证：有 PathMap 的应该排在空 PathMap 的前面
	if ovs[0].ContainerTargetDir != "short-path" {
		t.Errorf("Expected 'short-path' first, got '%s'", ovs[0].ContainerTargetDir)
	}
	if ovs[1].ContainerTargetDir != "empty-pathmap" {
		t.Errorf("Expected 'empty-pathmap' second, got '%s'", ovs[1].ContainerTargetDir)
	}
}

func TestMountPostOverlaysSortingMultiplePaths(t *testing.T) {
	// 测试多个 overlay 的排序
	ovs := []*apis.ContainerVolumeMountDiskPostOverlay{
		{
			ContainerTargetDir: "longest",
			Image: &apis.ContainerVolumeMountDiskPostImageOverlay{
				Id: "image-1",
				PathMap: map[string]string{
					".steam/debian-installation/steamapps/appmanifest_553420.acf": "/home/.steam/debian-installation/steamapps/appmanifest_553420.acf",
				},
			},
		},
		{
			ContainerTargetDir: "shortest",
			Image: &apis.ContainerVolumeMountDiskPostImageOverlay{
				Id: "image-2",
				PathMap: map[string]string{
					".steam": "/home/.steam",
				},
			},
		},
		{
			ContainerTargetDir: "medium",
			Image: &apis.ContainerVolumeMountDiskPostImageOverlay{
				Id: "image-3",
				PathMap: map[string]string{
					".steam/debian-installation": "/home/.steam/debian-installation",
				},
			},
		},
	}

	// 手动执行排序逻辑
	sort.Slice(ovs, func(i, j int) bool {
		return getMinPathMapKeyLength(ovs[i]) < getMinPathMapKeyLength(ovs[j])
	})

	// 验证排序顺序：shortest (6) < medium (25) < longest (67)
	expectedOrder := []string{"shortest", "medium", "longest"}
	for i, expected := range expectedOrder {
		if ovs[i].ContainerTargetDir != expected {
			t.Errorf("Position %d: expected '%s', got '%s'", i, expected, ovs[i].ContainerTargetDir)
		}
	}
}

func TestUnmountPostOverlaysSorting(t *testing.T) {
	// 创建测试数据：路径短的在前，路径长的在后（模拟 mount 后的顺序）
	ovs := []*apis.ContainerVolumeMountDiskPostOverlay{
		{
			ContainerTargetDir: "short-path",
			Image: &apis.ContainerVolumeMountDiskPostImageOverlay{
				Id: "image-2",
				PathMap: map[string]string{
					".steam": "/home/.steam",
				},
			},
		},
		{
			ContainerTargetDir: "long-path",
			Image: &apis.ContainerVolumeMountDiskPostImageOverlay{
				Id: "image-1",
				PathMap: map[string]string{
					".steam/debian-installation/steamapps/appmanifest_553420.acf": "/home/.steam/debian-installation/steamapps/appmanifest_553420.acf",
				},
			},
		},
	}

	// 验证排序前的顺序（路径短的在前）
	if getMinPathMapKeyLength(ovs[0]) >= getMinPathMapKeyLength(ovs[1]) {
		t.Fatal("Test setup error: ovs should be in ascending order initially")
	}

	// 手动执行反向排序逻辑（模拟 unmountPostOverlays 中的排序）
	sort.Slice(ovs, func(i, j int) bool {
		return getMinPathMapKeyLength(ovs[i]) > getMinPathMapKeyLength(ovs[j])
	})

	// 验证排序后的顺序：路径长的应该在前面
	if getMinPathMapKeyLength(ovs[0]) <= getMinPathMapKeyLength(ovs[1]) {
		t.Errorf("Sorting failed: expected longer path first, got lengths %d and %d",
			getMinPathMapKeyLength(ovs[0]), getMinPathMapKeyLength(ovs[1]))
	}

	// 验证具体路径：路径长的应该在前面
	if ovs[0].ContainerTargetDir != "long-path" {
		t.Errorf("Expected 'long-path' first, got '%s'", ovs[0].ContainerTargetDir)
	}
	if ovs[1].ContainerTargetDir != "short-path" {
		t.Errorf("Expected 'short-path' second, got '%s'", ovs[1].ContainerTargetDir)
	}
}

func TestUnmountPostOverlaysSortingMultiplePaths(t *testing.T) {
	// 测试多个 overlay 的反向排序
	ovs := []*apis.ContainerVolumeMountDiskPostOverlay{
		{
			ContainerTargetDir: "shortest",
			Image: &apis.ContainerVolumeMountDiskPostImageOverlay{
				Id: "image-2",
				PathMap: map[string]string{
					".steam": "/home/.steam",
				},
			},
		},
		{
			ContainerTargetDir: "medium",
			Image: &apis.ContainerVolumeMountDiskPostImageOverlay{
				Id: "image-3",
				PathMap: map[string]string{
					".steam/debian-installation": "/home/.steam/debian-installation",
				},
			},
		},
		{
			ContainerTargetDir: "longest",
			Image: &apis.ContainerVolumeMountDiskPostImageOverlay{
				Id: "image-1",
				PathMap: map[string]string{
					".steam/debian-installation/steamapps/appmanifest_553420.acf": "/home/.steam/debian-installation/steamapps/appmanifest_553420.acf",
				},
			},
		},
	}

	// 手动执行反向排序逻辑
	sort.Slice(ovs, func(i, j int) bool {
		return getMinPathMapKeyLength(ovs[i]) > getMinPathMapKeyLength(ovs[j])
	})

	// 验证排序顺序：longest (67) > medium (25) > shortest (6) - 与 mount 顺序相反
	expectedOrder := []string{"longest", "medium", "shortest"}
	for i, expected := range expectedOrder {
		if ovs[i].ContainerTargetDir != expected {
			t.Errorf("Position %d: expected '%s', got '%s'", i, expected, ovs[i].ContainerTargetDir)
		}
	}
}
