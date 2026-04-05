# Organizing Reminders

## Problem

I use recurring reminders a lot. Sometimes a lot of events occur on the same day. Need way to reduce the clutter.

## Solution

1. Export reminders to ICS files
2. Parse ICS files
3. Map event occurences over the next year
4. Sort days by number of occurences
5. Print list of days by most occurences. List events of day. List recurrence interval
6. Manually organize as you see fit

## Setup

1. Clone this repository `git clone git@github.com:JohnWebb4/organizing-reminders.git` 
2. Export your reminders to ICS files.
  - For Apple reminders,
    1. Create a shortcut
    2. Add "Find Reminder" input
    3. Add "Save File" output with "Reminders" type
    4. Run program
    5. When prompted for save location save to "reminders" folder in this repository
3. Run program `go run main.go`
4. Great success
