package utils

import (
	"strings"
	"testing"
)

type MockContainer struct {
	ID   string
	Name string
	Deps []string
}

func TestTopologicalSortContainers(t *testing.T) {
	containers := []MockContainer{
		{ID: "c1", Name: "container1"},
		{ID: "c2", Name: "container2", Deps: []string{"container1"}},
		{ID: "c3", Name: "container3", Deps: []string{"container2"}},
	}

	mockGetDependencies := func(c MockContainer) []string {
		return c.Deps
	}

	err := TopologicalSortContainers(containers, func(c MockContainer) string { return c.Name }, mockGetDependencies)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
}

func TestNewDependencyTopoGraph(t *testing.T) {
	containers := []MockContainer{
		{ID: "c1", Name: "container1"},
		{ID: "c2", Name: "container2", Deps: []string{"container1"}},
		{ID: "c3", Name: "container3", Deps: []string{"container2"}},
	}

	mockGetDependencies := func(c MockContainer) []string {
		return c.Deps
	}

	graph, err := NewDependencyTopoGraph(
		containers,
		func(c MockContainer) string { return c.ID },
		func(c MockContainer) string { return c.Name },
		mockGetDependencies,
	)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(graph.Leafs) != 1 || graph.Leafs[0] != "c1" {
		t.Errorf("Expected leafs [c1], got %v", graph.Leafs)
	}
}

func TestGetNextBatch(t *testing.T) {
	containers := []MockContainer{
		{ID: "c1", Name: "container1"},
		{ID: "c2", Name: "container2", Deps: []string{"container1"}},
		{ID: "c3", Name: "container3", Deps: []string{"container2"}},
	}

	mockGetDependencies := func(c MockContainer) []string {
		return c.Deps
	}

	mockFetchById := func(id string) MockContainer {
		for _, c := range containers {
			if c.ID == id {
				return c
			}
		}
		return MockContainer{}
	}

	graph, err := NewDependencyTopoGraph(
		containers,
		func(c MockContainer) string { return c.ID },
		func(c MockContainer) string { return c.Name },
		mockGetDependencies,
	)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// First batch should be container1
	batch1 := graph.GetNextBatch(mockFetchById)
	if len(batch1) != 1 || batch1[0].Name != "container1" {
		t.Errorf("Expected first batch [container1], got %v", batch1)
	}

	// Second batch should be container2
	batch2 := graph.GetNextBatch(mockFetchById)
	if len(batch2) != 1 || batch2[0].Name != "container2" {
		t.Errorf("Expected second batch [container2], got %v", batch2)
	}

	// Third batch should be container3
	batch3 := graph.GetNextBatch(mockFetchById)
	if len(batch3) != 1 || batch3[0].Name != "container3" {
		t.Errorf("Expected third batch [container3], got %v", batch3)
	}

	// No more batches
	batch4 := graph.GetNextBatch(mockFetchById)
	if batch4 != nil {
		t.Errorf("Expected nil, got %v", batch4)
	}
}

func TestCircularDependency(t *testing.T) {
	containers := []MockContainer{
		{ID: "c1", Name: "container1", Deps: []string{"container2"}},
		{ID: "c2", Name: "container2", Deps: []string{"container1"}},
	}

	mockGetDependencies := func(c MockContainer) []string {
		return c.Deps
	}

	err := TopologicalSortContainers(containers, func(c MockContainer) string { return c.Name }, mockGetDependencies)
	if err == nil {
		t.Fatal("Expected circular dependency error, got nil")
	}
	if !strings.Contains(err.Error(), "circular dependency") {
		t.Errorf("Expected circular dependency error, got %v", err)
	}
}
