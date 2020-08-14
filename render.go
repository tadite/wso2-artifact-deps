package main

import (
	"bufio"
	"github.com/goccy/go-graphviz"
	"github.com/goccy/go-graphviz/cgraph"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

func printGraph(dependenciesMap *map[CarName]map[CarName]*CarDependency, outPath string, carNames []CarName, ignoreCarRegex string) {
	isCarAllowed := createIsCarAllowedFunc(carNames, ignoreCarRegex)

	f, _ := os.Create(filepath.Join(outPath, "graph.txt"))
	defer f.Close()
	w := bufio.NewWriter(f)

	for carFrom, depToCars := range *dependenciesMap {
		if isCarAllowed(carFrom) {
			//w.WriteString(string(carFrom + "\n"))
			for carTo, dependency := range depToCars {
				if carFrom != carTo && dependency.HaveDependency && isCarAllowed(carFrom) && isCarAllowed(carTo) {
					w.WriteString(string(carFrom + " -> " + carTo + "\n"))
					fromArtifactLen := calcMaxFromArtifactDepLen(dependency)
					for fromArtifact := range dependency.ArtifactDependencies {
						toArtifactsDeps := (dependency.ArtifactDependencies)[fromArtifact]
						for toArtifact := range toArtifactsDeps {
							padding := strings.Repeat(" ", fromArtifactLen-len(fromArtifact))
							w.WriteString("  " + string(fromArtifact) + padding + " -> " + string(toArtifact) + "\n")
						}
					}
				}
			}
			w.WriteString("\n")
		}
	}

	w.Flush()
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

func renderGraph(dependenciesMap *map[CarName]map[CarName]*CarDependency, outPath string, carNames []CarName, ignoreCarRegex string) {
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

	nodeMap := map[CarName]*cgraph.Node{}
	appendNodeToGraph := func(carName CarName) {
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

func createIsCarAllowedFunc(carNames []CarName, ignoreCarRegex string) func(carName CarName) bool {
	analyseAllCars := len(carNames) == 0
	carsToAnalyse := make(map[CarName]bool)
	for _, carName := range carNames {
		carsToAnalyse[carName] = true
	}
	useRegexIgnore := len(ignoreCarRegex) > 0
	isCarAllowed := func(carName CarName) bool {
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
