package main

import (
	"bufio"
	"github.com/goccy/go-graphviz/cgraph"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"

	"github.com/goccy/go-graphviz"
)

var defaultDirsToSkip = []string{"target"}
var defaultFilesToSkip = []string{"pom.xml", "artifact.xml"}

type DepsParser struct {
	deps              map[CarName]map[CarName]bool
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

	deps := map[CarName]map[CarName]bool{}
	for carName, _ := range *artifactsMap {
		deps[carName] = map[CarName]bool{}
	}

	return &DepsParser{
		deps:              deps,
		artifactsToCarMap: artifactsToCarMap,
		artifactsRegex:    allArtifactsRegex,
		dirsToSkip:        dirsToSkip,
		filesToSkip:       filesToSkip,
	}
}

func FindDependencies(rootPath string, outPath string, carsToAnalyse []CarName) {
	artifactsMap := NewArtifactParser().Parse(rootPath)
	log.Printf("Analysed artifact.xml files")
	depsParser := NewDepsParser(artifactsMap, defaultDirsToSkip, defaultFilesToSkip)
	carDependenciesMap := depsParser.findDeps(rootPath, artifactsMap)
	renderGraph(carDependenciesMap, outPath, carsToAnalyse)
}

func renderGraph(dependenciesMap *map[CarName]map[CarName]bool, outPath string, carNames []CarName) {
	analyseAllCars := len(carNames) == 0
	carsToAnalyse := make(map[CarName]bool)
	for _, carName := range carNames {
		carsToAnalyse[carName] = true
	}

	g := graphviz.New()
	graph, err := g.Graph()
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		if err := graph.Close(); err != nil {
			log.Fatal(err)
		}
		g.Close()
	}()

	nodeMap := map[CarName]*cgraph.Node{}
	for carFrom, depToCars := range *dependenciesMap {
		for carTo, haveDependency := range depToCars {
			if nodeMap[carFrom] == nil && (analyseAllCars || carsToAnalyse[carFrom]) {
				n1, err := graph.CreateNode(string(carFrom))
				if err != nil {
					panic(err)
				}
				nodeMap[carFrom] = n1
			}
			if nodeMap[carTo] == nil && (analyseAllCars || carsToAnalyse[carFrom]) {
				n2, err := graph.CreateNode(string(carTo))
				if err != nil {
					panic(err)
				}
				nodeMap[carTo] = n2
			}

			if carFrom != carTo && haveDependency && (analyseAllCars || carsToAnalyse[carFrom]) {
				_, err := graph.CreateEdge("", nodeMap[carFrom], nodeMap[carTo])
				if err != nil {
					log.Fatal(err)
				}
			}
		}
	}

	if err := g.RenderFilename(graph, graphviz.PNG, filepath.Join(outPath, "graph.png")); err != nil {
		panic(err)
	}

	f, err := os.Create(filepath.Join(outPath, "graph.dot"))
	defer f.Close()
	w := bufio.NewWriter(f)
	if err := g.Render(graph, graphviz.XDOT, w); err != nil {
		panic(err)
	}
	w.Flush()
}

func (d *DepsParser) findDeps(path string, artifactsMap *CarArtifacts) *map[CarName]map[CarName]bool {
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
	d.addCarDependencies(foundArtifactsDeps, curFileCarName)
}

func (d *DepsParser) addCarDependencies(foundArtifactsDeps []string, curFileCarName CarName) {
	d.Lock()
	defer d.Unlock()
	for _, artifactName := range foundArtifactsDeps {
		d.deps[curFileCarName][d.artifactsToCarMap[ArtifactName(artifactName)]] = true
	}
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
