package main

import (
	"fmt"
	"strings"
	"time"
)

func validateEventRequest(input CreateEventRequest) error {
	if input.Event.ChatID == nil || *input.Event.ChatID <= 0 {
		return fmt.Errorf("event.chat_id must be provided by verified Telegram context")
	}

	if input.Event.CreatedByUserID == nil || *input.Event.CreatedByUserID <= 0 {
		return fmt.Errorf("event.created_by_user_id must be provided by verified Telegram context")
	}

	if strings.TrimSpace(input.Event.Title) == "" {
		return fmt.Errorf("event.title is required")
	}

	if strings.TrimSpace(input.Event.Timezone) == "" {
		return fmt.Errorf("event.timezone is required")
	}

	location, err := time.LoadLocation(strings.TrimSpace(input.Event.Timezone))
	if err != nil {
		return fmt.Errorf("event.timezone must be a valid IANA timezone (example: America/Bogota)")
	}

	if input.Event.TargetUserID != nil && *input.Event.TargetUserID <= 0 {
		return fmt.Errorf("event.target_user_id must be greater than 0 when provided")
	}

	if input.Event.IsAllDay {
		if input.Event.EventDate == nil || strings.TrimSpace(*input.Event.EventDate) == "" {
			return fmt.Errorf("event.event_date is required when event.is_all_day is true")
		}
		if input.Event.EventAt != nil && strings.TrimSpace(*input.Event.EventAt) != "" {
			return fmt.Errorf("event.event_at must be empty when event.is_all_day is true")
		}

		if _, err := time.Parse("2006-01-02", strings.TrimSpace(*input.Event.EventDate)); err != nil {
			return fmt.Errorf("event.event_date must use YYYY-MM-DD format")
		}
	} else {
		if input.Event.EventAt == nil || strings.TrimSpace(*input.Event.EventAt) == "" {
			return fmt.Errorf("event.event_at is required when event.is_all_day is false")
		}
		if input.Event.EventDate != nil && strings.TrimSpace(*input.Event.EventDate) != "" {
			return fmt.Errorf("event.event_date must be empty when event.is_all_day is false")
		}

		if _, err := parseDateTimeWithTimezone(strings.TrimSpace(*input.Event.EventAt), location); err != nil {
			return fmt.Errorf("event.event_at must be RFC3339 or YYYY-MM-DDTHH:MM")
		}
	}

	if input.Recurrence.OccurrenceCount != nil && *input.Recurrence.OccurrenceCount <= 0 {
		return fmt.Errorf("recurrence.occurrence_count must be greater than 0 when provided")
	}

	if strings.TrimSpace(input.Recurrence.UntilAt) != "" {
		untilAt, err := parseDateTimeWithTimezone(strings.TrimSpace(input.Recurrence.UntilAt), location)
		if err != nil {
			return fmt.Errorf("recurrence.until_at must be RFC3339 or YYYY-MM-DDTHH:MM")
		}

		if !input.Event.IsAllDay {
			eventAt, err := parseDateTimeWithTimezone(strings.TrimSpace(*input.Event.EventAt), location)
			if err != nil {
				return fmt.Errorf("event.event_at must be RFC3339 or YYYY-MM-DDTHH:MM")
			}

			if untilAt.Before(eventAt) {
				return fmt.Errorf("recurrence.until_at cannot be before event.event_at")
			}
		}
	}

	for i, reminder := range input.Reminders {
		if reminder.MessageTemplate != nil {
			trimmed := strings.TrimSpace(*reminder.MessageTemplate)
			if len(trimmed) > 1000 {
				return fmt.Errorf("reminders[%d].message_template must be 1000 chars or less", i)
			}
		}
	}

	return nil
}

func parseDateTimeWithTimezone(value string, location *time.Location) (time.Time, error) {
	if parsed, err := time.Parse(time.RFC3339, value); err == nil {
		return parsed, nil
	}

	return time.ParseInLocation("2006-01-02T15:04", value, location)
}
