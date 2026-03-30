package main

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type CreateEventResult struct {
	EventID      int64 `json:"event_id"`
	RecurrenceID int64 `json:"recurrence_id"`
}

func insertEventWithRelations(ctx context.Context, pool *pgxpool.Pool, input CreateEventRequest) (CreateEventResult, error) {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return CreateEventResult{}, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	eventDate, eventAt, untilAt, err := normalizeEventDateTimes(input)
	if err != nil {
		return CreateEventResult{}, err
	}

	var eventID int64
	err = tx.QueryRow(
		ctx,
		`INSERT INTO events (
			chat_id,
			created_by_user_id,
			target_user_id,
			type,
			title,
			description,
			is_all_day,
			event_date,
			event_at,
			timezone,
			is_active
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)
		RETURNING id`,
		*input.Event.ChatID,
		*input.Event.CreatedByUserID,
		input.Event.TargetUserID,
		input.Event.Type,
		strings.TrimSpace(input.Event.Title),
		normalizeStringPtr(input.Event.Description),
		input.Event.IsAllDay,
		eventDate,
		eventAt,
		strings.TrimSpace(input.Event.Timezone),
		input.Event.IsActive,
	).Scan(&eventID)
	if err != nil {
		return CreateEventResult{}, fmt.Errorf("insert events: %w", err)
	}

	var recurrenceID int64
	err = tx.QueryRow(
		ctx,
		`INSERT INTO event_recurrence (
			event_id,
			frequency,
			interval_value,
			until_at,
			occurrence_count,
			next_run_at
		) VALUES ($1,$2,$3,$4,$5,$6)
		RETURNING id`,
		eventID,
		input.Recurrence.Frequency,
		input.Recurrence.IntervalValue,
		untilAt,
		input.Recurrence.OccurrenceCount,
		computeInitialNextRun(input, eventDate, eventAt),
	).Scan(&recurrenceID)
	if err != nil {
		return CreateEventResult{}, fmt.Errorf("insert event_recurrence: %w", err)
	}

	for _, reminder := range input.Reminders {
		_, err = tx.Exec(
			ctx,
			`INSERT INTO event_reminders (
				event_id,
				offset_minutes,
				is_active,
				message_template
			) VALUES ($1,$2,$3,$4)`,
			eventID,
			reminder.OffsetMinutes,
			reminder.IsActive,
			normalizeStringPtr(reminder.MessageTemplate),
		)
		if err != nil {
			return CreateEventResult{}, fmt.Errorf("insert event_reminders: %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return CreateEventResult{}, fmt.Errorf("commit tx: %w", err)
	}

	return CreateEventResult{EventID: eventID, RecurrenceID: recurrenceID}, nil
}

func normalizeEventDateTimes(input CreateEventRequest) (*time.Time, *time.Time, *time.Time, error) {
	location, err := time.LoadLocation(strings.TrimSpace(input.Event.Timezone))
	if err != nil {
		return nil, nil, nil, fmt.Errorf("invalid event timezone: %w", err)
	}

	var eventDatePtr *time.Time
	if input.Event.EventDate != nil && strings.TrimSpace(*input.Event.EventDate) != "" {
		d, err := time.ParseInLocation("2006-01-02", strings.TrimSpace(*input.Event.EventDate), location)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("parse event_date: %w", err)
		}
		eventDatePtr = &d
	}

	var eventAtPtr *time.Time
	if input.Event.EventAt != nil && strings.TrimSpace(*input.Event.EventAt) != "" {
		t, err := parseDateTimeWithTimezone(strings.TrimSpace(*input.Event.EventAt), location)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("parse event_at: %w", err)
		}
		eventAtPtr = &t
	}

	var untilAtPtr *time.Time
	if strings.TrimSpace(input.Recurrence.UntilAt) != "" {
		u, err := parseDateTimeWithTimezone(strings.TrimSpace(input.Recurrence.UntilAt), location)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("parse recurrence.until_at: %w", err)
		}
		untilAtPtr = &u
	}

	return eventDatePtr, eventAtPtr, untilAtPtr, nil
}

func computeInitialNextRun(input CreateEventRequest, eventDate *time.Time, eventAt *time.Time) *time.Time {
	if input.Recurrence.Frequency == "none" {
		if input.Event.IsAllDay && eventDate != nil {
			t := eventDate.UTC()
			return &t
		}

		if eventAt != nil {
			t := eventAt.UTC()
			return &t
		}

		return nil
	}

	if input.Event.IsAllDay && eventDate != nil {
		t := eventDate.UTC()
		return &t
	}

	if eventAt != nil {
		t := eventAt.UTC()
		return &t
	}

	return nil
}

func normalizeStringPtr(value *string) *string {
	if value == nil {
		return nil
	}

	trimmed := strings.TrimSpace(*value)
	if trimmed == "" {
		return nil
	}

	return &trimmed
}
