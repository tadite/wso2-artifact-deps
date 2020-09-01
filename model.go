package main

type CarArtifacts map[string][]string

type CarDependency struct {
	HaveDependency       bool
	ArtifactDependencies map[string]map[string]bool
}

func NewFalseCarDependency() *CarDependency {
	return &CarDependency{
		HaveDependency: false,
	}
}

func NewCarDependency() *CarDependency {
	return &CarDependency{
		HaveDependency:       true,
		ArtifactDependencies: map[string]map[string]bool{},
	}
}
