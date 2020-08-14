package main

type Artifacts struct {
	Artifacts []*Artifact `xml:"artifact"`
}

type Artifact struct {
	Name string `xml:"name,attr"`
	Type string `xml:"type,attr"`
	Item Item   `xml:"item"`
}

type Item struct {
	File string `xml:"file"`
	Path string `xml:"path"`
}
