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
	flag.Parse()

	cars := strings.Split(*carNamesPtr, ",")
	var carNames = make([]CarName, len(cars))
	for _, car := range cars {
		carNames = append(carNames, CarName(strings.Trim(car, " ")))
	}
	start := time.Now()
	FindDependencies(*rootPathPtr, *outPathPtr, carNames)
	elapsed := time.Since(start)
	log.Printf("Took %s", elapsed)
}
