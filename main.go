package main

import (
	"flag"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func main() {
	progPath := filepath.Dir(os.Args[0])
	rootPathPtr := flag.String("path", progPath, "path to project root")
	outPathPtr := flag.String("outPath", progPath, "path where result will be saved")
	carNamesPtr := flag.String("carsToAnalyse", "", "names of car-apps to analyse")
	ignoreCarRegexPtr := flag.String("ignoreCarRegex", "", "regex for ignoring analyse of cars")
	findByRegexPtr := flag.Bool("findByRegex", false, "if 'true' then artifacts will be found using regex, otherwise by xml parsing")
	renderBothFindTypesPtr := flag.Bool("renderBothFindTypes", false, "if 'true' then both find types will be rendered")
	flag.Parse()

	cars := strings.Split(*carNamesPtr, ",")
	carNames := []string{}
	for _, car := range cars {
		carName := strings.Trim(car, " ")
		if len(carName) > 0 {
			carNames = append(carNames, string(carName))
		}
	}
	start := time.Now()
	FindDependencies(*rootPathPtr, *outPathPtr, carNames, *ignoreCarRegexPtr, *findByRegexPtr, *renderBothFindTypesPtr)
	elapsed := time.Since(start)
	log.Printf("Took %s", elapsed)
}
