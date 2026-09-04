package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/fourtytwo42/keelmesh/internal/coordination"
)

func main() {
	output := flag.String("output", "", "private output directory on VM 214")
	validDays := flag.Int("valid-days", 30, "certificate validity in days")
	topology := flag.String("topology", "", "optional JSON array of PKI node specifications for local verification")
	rotateFrom := flag.String("rotate-from", "", "stage a rotation from an existing PKI directory")
	overlapMinutes := flag.Int("overlap-minutes", 15, "bounded old/new trust overlap recorded for a rotation")
	flag.Parse()
	if *output == "" {
		fmt.Fprintln(os.Stderr, "--output is required")
		os.Exit(2)
	}
	config := coordination.PKIConfig{OutputDir: *output, ValidFor: time.Duration(*validDays) * 24 * time.Hour}
	if *topology != "" {
		encoded, err := os.ReadFile(*topology)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		if err := json.Unmarshal(encoded, &config.Nodes); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	}
	var err error
	if *rotateFrom != "" {
		err = coordination.StagePKIRotation(*rotateFrom, *output, config.Nodes, time.Duration(*overlapMinutes)*time.Minute)
	} else {
		err = coordination.GeneratePKI(config)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
