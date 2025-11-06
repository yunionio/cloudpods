package utils

import (
	"yunion.io/x/pkg/errors"
)

type GetObjIdName[T any] func(T) string
type GetDependencies[T any] func(T) []string

func TopologicalSortContainers[T any](objs []T, getName GetObjIdName[T], getDependencies GetDependencies[T]) error {
	if len(objs) == 0 {
		return nil
	}

	// Build a dependency graph and an in-degree table
	graph := make(map[string][]string)
	inDegree := make(map[string]int)

	// init graph and inDegree
	for _, obj := range objs {
		inDegree[getName(obj)] = 0
	}
	for _, obj := range objs {
		oName := getName(obj)
		for _, dep := range getDependencies(obj) {
			if _, exists := inDegree[dep]; !exists {
				return errors.Errorf("The dependent container %s does not exist.", dep)
			}
			graph[dep] = append(graph[dep], oName)
			inDegree[oName]++
		}
	}

	// Topological sorting: use a queue to process nodes with an in-degree of 0
	queue := []string{}
	for name, degree := range inDegree {
		if degree == 0 {
			queue = append(queue, name)
		}
	}

	sorted := []string{}
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		sorted = append(sorted, current)

		// Decrease the in-degree of neighboring nodes
		for _, neighbor := range graph[current] {
			inDegree[neighbor]--
			if inDegree[neighbor] == 0 {
				queue = append(queue, neighbor)
			}
		}
	}

	// Check whether a cycle exists
	if len(sorted) != len(objs) {
		return errors.Errorf("There is a circular dependency among the dependencies.")
	}

	return nil
}

type DependencyTopoGraph[T any] struct {
	Graph  map[string][]string `json:"graph,omitempty"`
	Degree map[string]int      `json:"degree,omitempty"`
	Leafs  []string            `json:"leafs"` // containers whose in-degree is zero
	Finish *bool               `json:"finish,omitempty"`
}

func NewDependencyTopoGraph[T any](
	objs []T,
	getId GetObjIdName[T],
	getName GetObjIdName[T],
	getDependencies GetDependencies[T],
) (*DependencyTopoGraph[T], error) {
	depGraph := &DependencyTopoGraph[T]{
		Graph:  make(map[string][]string),
		Degree: make(map[string]int),
		Leafs:  make([]string, 0, len(objs)),
	}

	nameToUUID := make(map[string]string)

	for _, obj := range objs {
		uuid := getId(obj)
		name := getName(obj)

		depGraph.Degree[uuid] = 0
		nameToUUID[name] = uuid
	}

	for _, obj := range objs {
		for _, dep := range getDependencies(obj) {
			depId := nameToUUID[dep]
			uuid := getId(obj)
			depGraph.Graph[depId] = append(depGraph.Graph[depId], uuid)
			depGraph.Degree[uuid]++
		}
	}

	for uuid, indegree := range depGraph.Degree {
		if indegree == 0 {
			depGraph.Leafs = append(depGraph.Leafs, uuid)
		}
	}

	return depGraph, nil
}

type FetchObjById[T any] func(string) T

func (dep *DependencyTopoGraph[T]) GetNextBatch(fetchById FetchObjById[T]) []T {
	if len(dep.Leafs) == 0 {
		return nil
	}

	objs := make([]T, 0, len(dep.Leafs))

	nextLeafs := make([]string, 0)

	for _, uuid := range dep.Leafs {
		objs = append(objs, fetchById(uuid))
		for _, neighbor := range dep.Graph[uuid] {
			dep.Degree[neighbor]--
			if dep.Degree[neighbor] == 0 {
				nextLeafs = append(nextLeafs, neighbor)
			}
		}
	}

	// log.Infof("Get next batch:\n Leafs: %s\n nextLeafs: %s", dep.Leafs, nextLeafs)

	dep.Leafs = nextLeafs

	if len(nextLeafs) == 0 {
		finish := true
		dep.Finish = &finish
	}

	return objs
}
