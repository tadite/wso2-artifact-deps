package main

import (
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
)

var defaultDirsToSkip = []string{"target"}
var defaultFilesToSkip = []string{"pom.xml", "artifact.xml"}

type DepsParser struct {
	deps              map[CarName]map[CarName]*CarDependency
	artifactsToCarMap map[ArtifactName]CarName
	artifactsRegex    *regexp.Regexp
	dirsToSkip        []string
	filesToSkip       []string

	sync.Mutex
	group sync.WaitGroup
}

func NewDepsParser(artifactsMap *CarArtifacts, dirsToSkip []string, filesToSkip []string) *DepsParser {
	var artifactsToCarMap = make(map[ArtifactName]CarName)
	var allArtifacts []string
	for carName, artifactNames := range *artifactsMap {
		for _, artifactName := range artifactNames {
			artifactsToCarMap[artifactName] = carName
			allArtifacts = append(allArtifacts, regexp.QuoteMeta(string(artifactName)))
		}
	}

	var allArtifactsRegexStr = strings.Join(allArtifacts, "|")
	var allArtifactsRegex = regexp.MustCompile(allArtifactsRegexStr)

	deps := map[CarName]map[CarName]*CarDependency{}
	for carName, _ := range *artifactsMap {
		deps[carName] = map[CarName]*CarDependency{}
	}

	return &DepsParser{
		deps:              deps,
		artifactsToCarMap: artifactsToCarMap,
		artifactsRegex:    allArtifactsRegex,
		dirsToSkip:        dirsToSkip,
		filesToSkip:       filesToSkip,
	}
}

func FindDependencies(rootPath string, outPath string, carsToAnalyse []CarName, ignoreCarRegex string) {
	artifactsMap := NewArtifactParser().Parse(rootPath)
	log.Printf("Analysed artifact.xml files")
	depsParser := NewDepsParser(artifactsMap, defaultDirsToSkip, defaultFilesToSkip)
	carDependenciesMap := depsParser.findDeps(rootPath, artifactsMap)
	renderGraph(carDependenciesMap, outPath, carsToAnalyse, ignoreCarRegex)
	printGraph(carDependenciesMap, outPath, carsToAnalyse, ignoreCarRegex)
}

func (d *DepsParser) findDeps(path string, artifactsMap *CarArtifacts) *map[CarName]map[CarName]*CarDependency {
	artifactsCount := 0
	for _, artifactNames := range *artifactsMap {
		artifactsCount += len(artifactNames)
	}
	fileCounter := 0

	err := filepath.Walk(path, func(path string, info os.FileInfo, err error) error {
		if info.IsDir() {
			if d.isSkipDir(info) {
				return filepath.SkipDir
			}
		} else {
			if d.isSkipFile(path, info) {
				return nil
			}
			// process file
			carName := getCarName(path)
			if len(carName) == 0 {
				return nil
			}
			go d.parseEsbXml(path, CarName(carName))
			fileCounter++
			log.Printf("started %d file analyses", fileCounter)
		}
		return nil
	})
	if err != nil {
		panic(err)
	}
	d.group.Wait()
	return &d.deps
}

func (d *DepsParser) parseEsbXml(path string, curFileCarName CarName) {
	d.group.Add(1)
	defer d.group.Done()

	xmlFile, err := os.Open(path)
	if err != nil {
		panic(err)
	}
	defer xmlFile.Close()

	textBytes, err := ioutil.ReadAll(xmlFile)
	if err != nil {
		panic(err)
	}
	text := string(textBytes)

	foundArtifactsDeps := d.artifactsRegex.FindAllString(text, -1)
	d.addCarDependencies(foundArtifactsDeps, curFileCarName, path)
}

func (d *DepsParser) addCarDependencies(foundArtifactsDeps []string, curFileCarName CarName, fromPath string) {
	d.Lock()
	defer d.Unlock()
	fromArtifact := ArtifactName(fileNameWithoutExtension(fromPath))
	for _, toArtifactStr := range foundArtifactsDeps {
		toArtifact := ArtifactName(toArtifactStr)
		toCarName := d.artifactsToCarMap[toArtifact]
		if curFileCarName != toCarName {
			if d.deps[curFileCarName][toCarName] == nil {
				d.deps[curFileCarName][toCarName] = NewCarDependency()
			}
			artifactDeps := &d.deps[curFileCarName][toCarName].ArtifactDependencies
			if (*artifactDeps)[fromArtifact] == nil {
				(*artifactDeps)[fromArtifact] = map[ArtifactName]bool{}
			}
			(*artifactDeps)[fromArtifact][toArtifact] = true
		}
	}
}

func fileNameWithoutExtension(fileName string) string {
	return strings.TrimSuffix(filepath.Base(fileName), filepath.Ext(fileName))
}

func getCarName(path string) string {
	pathParts := strings.Split(path, string(os.PathSeparator))
	var carName string
	for i := len(pathParts) - 1; i > 0; i-- {
		if pathParts[i] == "src" && i-2 > 0 {
			carName = pathParts[i-2]
		}
	}
	return carName
}

func (d *DepsParser) isSkipFile(path string, info os.FileInfo) bool {
	for _, file := range d.filesToSkip {
		if info.Name() == file {
			return true
		}
	}
	if filepath.Ext(path) != ".xml" {
		return true
	}
	return false
}

func (d *DepsParser) isSkipDir(info os.FileInfo) bool {
	for _, dir := range d.dirsToSkip {
		if info.Name() == dir {
			return true
		}
	}
	return false
}
