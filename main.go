package main

import (
	"fmt"
	"os"
	"path/filepath"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: organizing-reminders <folder>")
		os.Exit(1)
	}
	folder := os.Args[1]

	matches, err := filepath.Glob(filepath.Join(folder, "*.ics"))
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	var reminders []*Reminder
	for _, path := range matches {
		rem, err := parseICS(path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "warning: skipping %s: %v\n", path, err)
			continue
		}

		if rem.Status != "COMPLETED" {
			fmt.Printf("Reminder: %s\n", rem)
			reminders = append(reminders, rem)
		}
	}

	fmt.Printf("Loaded %d reminders\n", len(reminders))
}
