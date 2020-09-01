package main

import (
	"bufio"
	"github.com/goccy/go-graphviz"
	"github.com/goccy/go-graphviz/cgraph"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

func printGraph(dependenciesMap *map[string]map[string]*CarDependency, outPath string, carNames []string, ignoreCarRegex string, fileNamePrefix string) {
	isCarAllowed := createIsCarAllowedFunc(carNames, ignoreCarRegex)

	f, _ := os.Create(filepath.Join(outPath, fileNamePrefix+"graph.txt"))
	defer f.Close()
	w := bufio.NewWriter(f)

	keys := getSortedMapKeysFromFullDepsMap(dependenciesMap)

	for _, carFrom := range keys {
		if isCarAllowed(carFrom) {
			depToKeys := getSortedMapKeyFromPartDepsMap((*dependenciesMap)[carFrom])
			var printedDeps bool
			for _, carTo := range depToKeys {
				dependency := (*dependenciesMap)[carFrom][carTo]
				if carFrom != carTo && dependency.HaveDependency && isCarAllowed(carFrom) && isCarAllowed(carTo) {
					printedDeps = true
					w.WriteString(carFrom + " -> " + carTo + "\n")
					fromArtifactLen := calcMaxFromArtifactDepLen(dependency)
					fromArtifacts := getSortedMapKeysFromArtifactFullMap(dependency.ArtifactDependencies)
					for _, fromArtifact := range fromArtifacts {
						toArtifactsDepsMap := (dependency.ArtifactDependencies)[fromArtifact]
						toArtifactsDeps := getSortedMapKeysFromArtifactsPartMap(toArtifactsDepsMap)
						for _, toArtifact := range toArtifactsDeps {
							padding := strings.Repeat(" ", fromArtifactLen-len(fromArtifact))
							w.WriteString("  " + fromArtifact + padding + " -> " + toArtifact + "\n")
						}
					}
				}
			}
			if !printedDeps {
				w.WriteString(carFrom + "\n")
			}
			w.WriteString("\n")
		}
	}

	w.Flush()
}

func getSortedMapKeysFromArtifactsPartMap(m map[string]bool) []string {
	keys := make([]string, len(m))
	i := 0
	for k := range m {
		keys[i] = k
		i++
	}
	sort.Strings(keys)
	return keys
}

func getSortedMapKeysFromArtifactFullMap(m map[string]map[string]bool) []string {
	keys := make([]string, len(m))
	i := 0
	for k := range m {
		keys[i] = k
		i++
	}
	sort.Strings(keys)
	return keys
}

func getSortedMapKeyFromPartDepsMap(m map[string]*CarDependency) []string {
	keys := make([]string, len(m))
	i := 0
	for k := range m {
		keys[i] = k
		i++
	}
	sort.Strings(keys)
	return keys
}

func getSortedMapKeysFromFullDepsMap(dependenciesMap *map[string]map[string]*CarDependency) []string {
	keys := make([]string, len(*dependenciesMap))
	i := 0
	for k := range *dependenciesMap {
		keys[i] = k
		i++
	}
	sort.Strings(keys)
	return keys
}

func calcMaxFromArtifactDepLen(dependency *CarDependency) int {
	maxArtifactLen := -1
	for fromArtifact := range dependency.ArtifactDependencies {
		if maxArtifactLen < len(fromArtifact) {
			maxArtifactLen = len(fromArtifact)
		}
	}
	return maxArtifactLen
}

func renderGraph(dependenciesMap *map[string]map[string]*CarDependency, outPath string, carNames []string, ignoreCarRegex string, fileNamePrefix string) {
	isCarAllowed := createIsCarAllowedFunc(carNames, ignoreCarRegex)

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

	nodeMap := map[string]*cgraph.Node{}
	appendNodeToGraph := func(carName string) {
		if nodeMap[carName] == nil {
			n1, err := graph.CreateNode(string(carName))
			if err != nil {
				panic(err)
			}
			nodeMap[carName] = n1
		}
	}

	for carFrom, depToCars := range *dependenciesMap {
		if isCarAllowed(carFrom) {
			appendNodeToGraph(carFrom)
		}

		for carTo, dependency := range depToCars {
			if isCarAllowed(carTo) {
				appendNodeToGraph(carTo)
			}

			if carFrom != carTo && dependency.HaveDependency && nodeMap[carFrom] != nil && nodeMap[carTo] != nil {
				_, err := graph.CreateEdge("", nodeMap[carFrom], nodeMap[carTo])
				if err != nil {
					log.Fatal(err)
				}
			}
		}
	}

	if err := g.RenderFilename(graph, graphviz.PNG, filepath.Join(outPath, fileNamePrefix+"graph.png")); err != nil {
		panic(err)
	}

	f, err := os.Create(filepath.Join(outPath, fileNamePrefix+"graph.dot"))
	defer f.Close()
	w := bufio.NewWriter(f)
	if err := g.Render(graph, graphviz.XDOT, w); err != nil {
		panic(err)
	}
	w.Flush()
}

func createIsCarAllowedFunc(carNames []string, ignoreCarRegex string) func(carName string) bool {
	analyseAllCars := len(carNames) == 0
	carsToAnalyse := make(map[string]bool)
	for _, carName := range carNames {
		carsToAnalyse[carName] = true
	}
	useRegexIgnore := len(ignoreCarRegex) > 0
	isCarAllowed := func(carName string) bool {
		if (analyseAllCars && !useRegexIgnore) || carsToAnalyse[carName] {
			return true
		}
		if useRegexIgnore {
			ignoreByRegex, _ := regexp.MatchString(ignoreCarRegex, string(carName))
			if !ignoreByRegex {
				return true
			}
		}

		return false
	}
	return isCarAllowed
}

func renderBothTypesGraph(dependenciesMapRegex *map[string]map[string]*CarDependency,
	dependenciesMapMarshalling *map[string]map[string]*CarDependency,
	outPath string, carNames []string, ignoreCarRegex string) {
	isCarAllowed := createIsCarAllowedFunc(carNames, ignoreCarRegex)

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

	nodeMap := map[string]*cgraph.Node{}
	appendNodeToGraph := func(carName string) {
		if nodeMap[carName] == nil {
			n1, err := graph.CreateNode(string(carName))
			if err != nil {
				panic(err)
			}
			nodeMap[carName] = n1
		}
	}

	edgesMap := map[string]map[string]*cgraph.Edge{}
	appendEdges := func(deps *map[string]map[string]*CarDependency, color string, bothColor string) {
		for carFrom, depToCars := range *deps {
			if isCarAllowed(carFrom) {
				appendNodeToGraph(carFrom)
			}

			for carTo, dependency := range depToCars {
				if isCarAllowed(carTo) {
					appendNodeToGraph(carTo)
				}

				if carFrom != carTo && dependency.HaveDependency && nodeMap[carFrom] != nil && nodeMap[carTo] != nil {
					var edge *cgraph.Edge
					if edgesMap[carFrom][carTo] != nil {
						edge = edgesMap[carFrom][carTo]
						edge.SetColor(bothColor)
					} else {
						edge, err = graph.CreateEdge("", nodeMap[carFrom], nodeMap[carTo])
						if err != nil {
							log.Fatal(err)
						}
						edge.SetColor(color)
					}

					if edgesMap[carFrom] == nil {
						edgesMap[carFrom] = map[string]*cgraph.Edge{}
					}
					edgesMap[carFrom][carTo] = edge
				}
			}
		}
	}
	appendEdges(dependenciesMapRegex, "blue", "red")
	appendEdges(dependenciesMapMarshalling, "green", "red")

	if err := g.RenderFilename(graph, graphviz.PNG, filepath.Join(outPath, "both-graph.png")); err != nil {
		panic(err)
	}

	f, err := os.Create(filepath.Join(outPath, "both-graph.dot"))
	defer f.Close()
	w := bufio.NewWriter(f)
	if err := g.Render(graph, graphviz.XDOT, w); err != nil {
		panic(err)
	}
	w.Flush()
}
