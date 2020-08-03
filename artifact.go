package main

import (
	"encoding/xml"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

type ArtifactParser struct {
	sync.Mutex
	group        sync.WaitGroup
	artifactsMap map[string][]Artifact
}

func (p *ArtifactParser) Parse(path string) *CarArtifacts {
	err := filepath.Walk(path, func(path string, info os.FileInfo, err error) error {
		if !info.IsDir() && filepath.Base(path) == "artifact.xml" {
			go p.parseArtifactXml(path)
		}
		return nil
	})
	if err != nil {
		panic(err)
	}
	p.group.Wait()

	carArtifacts := make(CarArtifacts)
	for carName, artifacts := range p.artifactsMap {
		var artifactNames []ArtifactName
		for _, artifact := range artifacts {
			artifactNames = append(artifactNames, ArtifactName(artifact.Name))
		}
		carArtifacts[CarName(carName)] = artifactNames
	}
	return &carArtifacts
}

func (p *ArtifactParser) parseArtifactXml(artifactXmlPath string) {
	p.group.Add(1)
	defer p.group.Done()

	artifactXmlPathParts := strings.Split(artifactXmlPath, string(os.PathSeparator))
	carName := artifactXmlPathParts[len(artifactXmlPathParts)-3]

	artifactsFromXml := p.getArtifactsFromXml(artifactXmlPath)

	p.Lock()
	p.artifactsMap[carName] = append(p.artifactsMap[carName], *artifactsFromXml...)
	p.Unlock()
}

func (p *ArtifactParser) getArtifactsFromXml(path string) *[]Artifact {
	xmlFile, err := os.Open(path)
	if err != nil {
		log.Fatalln(err)
	}
	defer xmlFile.Close()

	byteValue, _ := ioutil.ReadAll(xmlFile)
	var artifacts Artifacts
	err = xml.Unmarshal(byteValue, &artifacts)
	if err != nil {
		log.Fatalln(err)
	}
	return &artifacts.Artifacts
}

func NewArtifactParser() *ArtifactParser {
	return &ArtifactParser{
		artifactsMap: make(map[string][]Artifact),
	}
}
