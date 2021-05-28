package main

import (
	"fmt"
	"math"
	"time"

	"github.com/graham/rrule"
	"github.com/pkg/errors"
)

//RecurrenceRuleSeparator : separator between recurrence rules
const RecurrenceRuleSeparator = "|"

//RecurrenceInterval : recurrence interval
type RecurrenceInterval int

//recurrence intervals
const (
	RecurrenceIntervalOnce RecurrenceInterval = iota
	RecurrenceIntervalDaily
	RecurrenceIntervalWeekly
	RecurrenceIntervalEveryTwoWeeks
	RecurrenceIntervalMonthly
)

//RecurrenceFreq : recurrence frequency
type RecurrenceFreq struct {
	Label    string
	Value    RecurrenceInterval
	Freq     rrule.FrequencyValue
	Interval int
}

//recurrence frequencies
var (
	RecurrenceFreqOnce = RecurrenceFreq{
		Label:    "One-Time Only",
		Value:    RecurrenceIntervalOnce,
		Freq:     0,
		Interval: 1,
	}
	RecurrenceFreqWeekly = RecurrenceFreq{
		Label:    "Weekly",
		Value:    RecurrenceIntervalWeekly,
		Freq:     rrule.WEEKLY,
		Interval: 1,
	}
	RecurrenceFreqTwoWeeks = RecurrenceFreq{
		Label:    "Every Two Weeks",
		Value:    RecurrenceIntervalEveryTwoWeeks,
		Freq:     rrule.WEEKLY,
		Interval: 2,
	}
	RecurrenceFreqMonthly = RecurrenceFreq{
		Label:    "Monthly",
		Value:    RecurrenceIntervalMonthly,
		Freq:     rrule.MONTHLY,
		Interval: 1,
	}
)

//RecurrenceFreqs : recurrence frequencies
var RecurrenceFreqs []RecurrenceFreq = []RecurrenceFreq{
	RecurrenceFreqOnce,
	RecurrenceFreqWeekly,
	RecurrenceFreqTwoWeeks,
}

//LoadRecurrenceFreq : load a recurrence frequency by the interval
func LoadRecurrenceFreq(interval RecurrenceInterval) *RecurrenceFreq {
	for _, freq := range RecurrenceFreqs {
		if interval == freq.Value {
			return &freq
		}
	}
	return nil
}

//ParseRecurrenceFreq : parse a recurrence frequency by label
func ParseRecurrenceFreq(freqStr *string) *RecurrenceFreq {
	if freqStr == nil {
		return nil
	}
	freq := *freqStr
	for _, recurrenceFreq := range RecurrenceFreqs {
		if freq == recurrenceFreq.Label {
			return &recurrenceFreq
		}
	}
	return nil
}

//FindRecurrenceRuleByDay : find the day of the week, including the week offset, in the month
func FindRecurrenceRuleByDay(t time.Time) rrule.ForDay {
	dayOfWeek := rrule.ForDay{
		Weekday: t.Weekday(),
	}

	//compute the the week of the month, assuming mon. as the start of the week
	offset := int(math.Ceil(float64((t.Day() - 1 - int(t.Weekday()))) / 7))

	//signal the last week
	if offset > 4 {
		offset = -1
	}
	dayOfWeek.Offset = offset
	return dayOfWeek
}

//CreateRecurrenceRule : create a recurrence rule based on the frequency
func CreateRecurrenceRule(freq *RecurrenceFreq, date time.Time) (string, error) {
	if freq == nil {
		return "", nil
	}
	if freq.Freq == 0 {
		return "", nil
	}

	//create the rule
	rule := rrule.RecurringRule{
		Frequency:     freq.Freq,
		Interval:      freq.Interval,
		WorkWeekStart: time.Monday,
	}

	//handle the monthly case and specify the day-of-week, including week offset
	if rule.Frequency == rrule.MONTHLY {
		rule.ByDay = []rrule.ForDay{FindRecurrenceRuleByDay(date)}
	}
	return rule.RecurString(), nil
}

//RecurrenceRule : recurrence rule
type RecurrenceRule struct {
	*rrule.RecurringRule
}

//ParseRecurrenceRule : parse recurrence rule
func ParseRecurrenceRule(ruleStr string) (*RecurrenceRule, error) {
	rule, err := rrule.Parse(ruleStr)
	if err != nil {
		return nil, errors.Wrap(err, fmt.Sprintf("parse rule: %s", ruleStr))
	}
	ruleWrapper := &RecurrenceRule{rule}
	return ruleWrapper, nil
}
