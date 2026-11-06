package commands_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/joaquinBrassinne/MOVO/internal/search"
	"github.com/joaquinBrassinne/MOVO/internal/search/app/commands"
)

type fakeResolver struct {
	id  int64
	err error
}

func (f fakeResolver) PublicBusinessID(context.Context, string) (int64, error) { return f.id, f.err }

type recordingCounter struct {
	events []search.Event
	err    error
}

func (c *recordingCounter) Increment(_ context.Context, e search.Event, _ time.Time) error {
	if c.err != nil {
		return c.err
	}
	c.events = append(c.events, e)
	return nil
}

func TestRecordProfileView(t *testing.T) {
	t.Parallel()

	lookupFailed := errors.New("database down")
	tests := []struct {
		name       string
		resolver   fakeResolver
		counter    *recordingCounter
		userAgent  string
		wantErr    error
		wantEvents []search.Event
	}{
		{
			name:       "a public business gets one profile view with no branch",
			resolver:   fakeResolver{id: 7},
			counter:    &recordingCounter{},
			userAgent:  chrome,
			wantEvents: []search.Event{{Metric: search.ProfileView, BusinessID: 7}},
		},
		{
			name:      "a bot is not counted",
			resolver:  fakeResolver{id: 7},
			counter:   &recordingCounter{},
			userAgent: "Googlebot/2.1",
		},
		{
			name:      "a business that is not public is not counted and not reported",
			resolver:  fakeResolver{err: search.ErrBusinessNotFound},
			counter:   &recordingCounter{},
			userAgent: chrome,
		},
		{
			name:      "a counter that fails is swallowed",
			resolver:  fakeResolver{id: 7},
			counter:   &recordingCounter{err: errors.New("write failed")},
			userAgent: chrome,
		},
		{
			name:      "a failed lookup is an error",
			resolver:  fakeResolver{err: lookupFailed},
			counter:   &recordingCounter{},
			userAgent: chrome,
			wantErr:   lookupFailed,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			record := commands.NewRecordMetric(tt.counter, nil)
			err := commands.NewRecordProfileView(tt.resolver, record, nil).Handle(t.Context(), "studio", tt.userAgent)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("err = %v, want %v", err, tt.wantErr)
			}
			if len(tt.counter.events) != len(tt.wantEvents) {
				t.Fatalf("events = %v, want %v", tt.counter.events, tt.wantEvents)
			}
			for i := range tt.wantEvents {
				if tt.counter.events[i] != tt.wantEvents[i] {
					t.Errorf("event %d = %+v, want %+v", i, tt.counter.events[i], tt.wantEvents[i])
				}
			}
		})
	}
}
