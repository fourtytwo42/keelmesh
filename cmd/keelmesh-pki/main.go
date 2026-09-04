package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/fourtytwo42/keelmesh/internal/coordination"
)

func main() {
	output := flag.String("output", "", "private output directory on VM 214")
	validDays := flag.Int("valid-days", 30, "certificate validity in days")
	flag.Parse()
	if *output == "" {
		fmt.Fprintln(os.Stderr, "--output is required")
		os.Exit(2)
	}
	if err := coordination.GeneratePKI(coordination.PKIConfig{OutputDir: *output, ValidFor: time.Duration(*validDays) * 24 * time.Hour}); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
