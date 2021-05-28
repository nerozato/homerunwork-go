package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

func main() {
	start := time.Now()
	args := os.Args[1:]
	if len(args) == 0 {
		s := CreateServer()
		signals := make(chan os.Signal, 1)
		signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)
		var wg sync.WaitGroup
		wg.Add(1)
		go func() {
			s.Start()
		}()
		go func() {
			defer wg.Done()
			signal := <-signals
			s.logger.Infow("signal received", "signal", signal)
			s.Stop(s.ctx)
			s.wg.Wait()
		}()
		wg.Wait()
		return
	}

	//handle commands
	ctx := context.Background()
	logger, err := InitLogger(ctx, "")
	if err != nil {
		panic(fmt.Sprintf("log failure: %v", err))
	}
	cmd := args[0]
	switch cmd {
	case "list-calendars":
		cals, err := ListCalendarsGoogle(ctx)
		if err != nil {
			logger.Errorw("list calendars", "error", err)
		}
		for _, cal := range cals {
			logger.Infow("calendar", "id", cal.Id, "title", cal.Summary)
		}
		logger.Infow("list calendars", "count", len(cals), "elapsedMS", FormatElapsedMS(start))
	case "delete-calendars":
		count, err := DeleteCalendarsGoogle(ctx)
		if err != nil {
			logger.Errorw("delete calendars", "error", err)
		}
		logger.Infow("delete calendars", "count", count, "elapsedMS", FormatElapsedMS(start))
	case "get-event":
		if len(args) != 3 {
			logger.Errorw("require the calendar id and event id")
			return
		}
		event, err := GetEventGoogle(ctx, &args[1], &args[2], "")
		if err != nil {
			logger.Errorw("get event", "error", err)
		}
		logger.Infow("event", "calendarId", args[1], "id", args[2], "event", event)
		logger.Infow("get event", "elapsedMS", FormatElapsedMS(start))
	case "list-events":
		if len(args) != 2 {
			logger.Errorw("require the calendar id")
			return
		}
		events, err := ListEventsAndInstancesGoogle(ctx, &args[1], time.Time{}, time.Time{})
		if err != nil {
			logger.Errorw("list events", "error", err)
		}
		for _, event := range events {
			logger.Infow("event", "calendarId", args[1], "id", event.Id, "title", event.Summary)
		}
		logger.Infow("list events", "count", len(events), "elapsedMS", FormatElapsedMS(start))
	case "delete-event":
		if len(args) != 3 {
			logger.Errorw("require the calendar id and event id")
			return
		}
		err := DeleteEventGoogle(ctx, &args[1], &args[2])
		if err != nil {
			logger.Errorw("delete event", "error", err)
		}
		logger.Infow("delete event", "elapsedMS", FormatElapsedMS(start))
	case "delete-events":
		if len(args) != 2 {
			logger.Errorw("require the calendar id")
			return
		}
		events, err := ListEventsGoogle(ctx, &args[1], time.Time{}, time.Time{})
		if err != nil {
			logger.Errorw("list events", "error", err)
		}
		count := 0
		for _, event := range events {
			logger.Infow("delete event", "calendarId", args[1], "id", event.Id)
			err := DeleteEventGoogle(ctx, &args[1], &event.Id)
			if err != nil {
				logger.Errorw("delete calendars", "error", err)
			}
			count++
		}
		logger.Infow("delete events", "count", count, "elapsedMS", FormatElapsedMS(start))
	case "delete-events-all":
		cals, err := ListCalendarsGoogle(ctx)
		if err != nil {
			logger.Errorw("list calendars", "error", err)
		}
		count := 0
		for _, cal := range cals {
			events, err := ListEventsGoogle(ctx, &cal.Id, time.Time{}, time.Time{})
			if err != nil {
				logger.Errorw("list events", "error", err)
			}
			for _, event := range events {
				logger.Infow("delete event", "calendarId", cal.Id, "id", event.Id)
				err := DeleteEventGoogle(ctx, &cal.Id, &event.Id)
				if err != nil {
					logger.Errorw("delete calendars", "error", err)
				}
				count++
			}
		}
		logger.Infow("delete events all", "count", count, "elapsedMS", FormatElapsedMS(start))
	case "check-instances":
		if len(args) != 3 {
			logger.Errorw("require the calendar id and event id")
			return
		}
		ok, err := CheckInstancesGoogle(ctx, &args[1], &args[2])
		if err != nil {
			logger.Errorw("check instances", "error", err)
		}
		logger.Infow("check instances", "ok", ok, "elapsedMS", FormatElapsedMS(start))
	case "validate-timezones":
		ok := ValidateTimeZoneList()
		logger.Infow("validate timezones", "count", len(TimeZoneList), "ok", ok, "elapsedMS", FormatElapsedMS(start))
	default:
		logger.Errorw("unknown command")
	}
}
