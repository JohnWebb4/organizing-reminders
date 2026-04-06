# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

A Go CLI tool that parses Apple Reminders exported as ICS files, maps recurring event occurrences over the next year, and prints days sorted by number of occurrences to help identify scheduling clutter.

## Commands

```bash
# Run the program (pass path to folder containing .ics files)
go run main.go ./reminders

# Build
go build ./

# Test
go test ./

# Run a single test
go test -run TestName ./
```

## Architecture

- `main.go` — entry point; accepts a folder path as the first argument
- `reminders/` — sample ICS files exported from Apple Reminders (not source code)

The planned pipeline (per README):
1. Parse ICS files from the given folder
2. Map each recurring reminder's occurrences over the next year
3. Sort days by occurrence count
4. Print days with the most reminders, listing events and their recurrence intervals
