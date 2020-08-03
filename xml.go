package main

type Artifacts struct {
	Artifacts []Artifact `xml:"artifact"`
}

type Artifact struct {
	Name string `xml:"name,attr"`
	Type string `xml:"type,attr"`
}
