package main

type CarName string
type ArtifactName string
type CarArtifacts map[CarName][]ArtifactName

type CarDependency struct {
	HaveDependency       bool
	ArtifactDependencies map[ArtifactName]map[ArtifactName]bool
}

func NewFalseCarDependency() *CarDependency {
	return &CarDependency{
		HaveDependency: false,
	}
}

func NewCarDependency() *CarDependency {
	return &CarDependency{
		HaveDependency:       true,
		ArtifactDependencies: map[ArtifactName]map[ArtifactName]bool{},
	}
}
