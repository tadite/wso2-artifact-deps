package main

import (
	"github.com/beevik/etree"
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
	deps              map[string]map[string]*CarDependency
	artifactsToCarMap map[string]string
	artifactsRegex    *regexp.Regexp
	dirsToSkip        []string
	filesToSkip       []string
	findByRegex       bool

	parseEsbXml func(dp *DepsParser, path string, curFileCarName string)

	sync.Mutex
	group sync.WaitGroup
}

func NewDepsParser(artifactsMap *CarArtifacts, dirsToSkip []string, filesToSkip []string, findByRegex bool) *DepsParser {
	var artifactsToCarMap = make(map[string]string)
	var allArtifacts []string
	for carName, artifactNames := range *artifactsMap {
		for _, artifactName := range artifactNames {
			artifactsToCarMap[artifactName] = carName
			allArtifacts = append(allArtifacts, regexp.QuoteMeta(string(artifactName)))
		}
	}

	var allArtifactsRegexStr = strings.Join(allArtifacts, "|")
	var allArtifactsRegex = regexp.MustCompile(allArtifactsRegexStr)

	deps := map[string]map[string]*CarDependency{}
	for carName, _ := range *artifactsMap {
		deps[carName] = map[string]*CarDependency{}
	}

	parseEsbXmlFunc := parseEsbXmlByMarshalling
	if findByRegex {
		parseEsbXmlFunc = parseEsbXmlByRegex
	}

	return &DepsParser{
		deps:              deps,
		artifactsToCarMap: artifactsToCarMap,
		artifactsRegex:    allArtifactsRegex,
		dirsToSkip:        dirsToSkip,
		filesToSkip:       filesToSkip,
		parseEsbXml:       parseEsbXmlFunc,
		findByRegex:       findByRegex,
	}
}

func FindDependencies(rootPath string, outPath string, carsToAnalyse []string, ignoreCarRegex string, findByRegex bool, renderBothFindTypes bool) {
	artifactsMap := NewArtifactParser().Parse(rootPath)
	log.Printf("Analysed artifact.xml files")
	depsParser := NewDepsParser(artifactsMap, defaultDirsToSkip, defaultFilesToSkip, findByRegex)
	carDependenciesMap := depsParser.findDeps(rootPath, artifactsMap)
	renderGraph(carDependenciesMap, outPath, carsToAnalyse, ignoreCarRegex, depsParser.getTypePrefix())
	printGraph(carDependenciesMap, outPath, carsToAnalyse, ignoreCarRegex, depsParser.getTypePrefix())
	if renderBothFindTypes {
		var carDependenciesByRegex *map[string]map[string]*CarDependency
		var carDependenciesByMarshalling *map[string]map[string]*CarDependency
		if findByRegex {
			carDependenciesByRegex = carDependenciesMap
			depsParser := NewDepsParser(artifactsMap, defaultDirsToSkip, defaultFilesToSkip, !findByRegex)
			carDependenciesByMarshalling = depsParser.findDeps(rootPath, artifactsMap)
			renderGraph(carDependenciesByMarshalling, outPath, carsToAnalyse, ignoreCarRegex, depsParser.getTypePrefix())
			printGraph(carDependenciesByMarshalling, outPath, carsToAnalyse, ignoreCarRegex, depsParser.getTypePrefix())
		} else {
			carDependenciesByMarshalling = carDependenciesMap
			depsParser := NewDepsParser(artifactsMap, defaultDirsToSkip, defaultFilesToSkip, !findByRegex)
			carDependenciesByRegex = depsParser.findDeps(rootPath, artifactsMap)
			renderGraph(carDependenciesByRegex, outPath, carsToAnalyse, ignoreCarRegex, depsParser.getTypePrefix())
			printGraph(carDependenciesByRegex, outPath, carsToAnalyse, ignoreCarRegex, depsParser.getTypePrefix())
		}
		renderBothTypesGraph(carDependenciesByRegex, carDependenciesByMarshalling, outPath, carsToAnalyse, ignoreCarRegex)
	}
}

func (d *DepsParser) getTypePrefix() string {
	if d.findByRegex {
		return "regex-"
	} else {
		return "xml-"
	}
}

func (d *DepsParser) findDeps(path string, artifactsMap *CarArtifacts) *map[string]map[string]*CarDependency {
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
			go d.parseEsbXml(d, path, string(carName))
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

func parseEsbXmlByMarshalling(dp *DepsParser, path string, curFileCarName string) {
	dp.group.Add(1)
	defer dp.group.Done()

	doc := etree.NewDocument()
	if err := doc.ReadFromFile(path); err != nil {
		panic(err)
	}
	foundArtifacts := FindArtifactsInDoc(doc)

	dp.addCarDependencies(foundArtifacts, curFileCarName, path)
}

func parseEsbXmlByRegex(dp *DepsParser, path string, curFileCarName string) {
	dp.group.Add(1)
	defer dp.group.Done()

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

	foundArtifactsDeps := dp.artifactsRegex.FindAllString(text, -1)
	dp.addCarDependencies(foundArtifactsDeps, curFileCarName, path)
}

func (d *DepsParser) addCarDependencies(foundArtifactsDeps []string, curFileCarName string, fromPath string) {
	d.Lock()
	defer d.Unlock()
	fromArtifact := string(fileNameWithoutExtension(fromPath))
	for _, toArtifactStr := range foundArtifactsDeps {
		toArtifact := string(toArtifactStr)
		toCarName := d.artifactsToCarMap[toArtifact]
		if len(toCarName) > 0 && curFileCarName != toCarName {
			if d.deps[curFileCarName][toCarName] == nil {
				d.deps[curFileCarName][toCarName] = NewCarDependency()
			}
			artifactDeps := &d.deps[curFileCarName][toCarName].ArtifactDependencies
			if (*artifactDeps)[fromArtifact] == nil {
				(*artifactDeps)[fromArtifact] = map[string]bool{}
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
