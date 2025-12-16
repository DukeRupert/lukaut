package worker

import (
	"context"
	"testing"
	"time"
)

func TestConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  Config
		wantErr bool
	}{
		{
			name:    "valid default config",
			config:  DefaultConfig(),
			wantErr: false,
		},
		{
			name: "concurrency too low",
			config: Config{
				Concurrency:       0,
				PollInterval:      5 * time.Second,
				JobTimeout:        5 * time.Minute,
				ShutdownTimeout:   30 * time.Second,
				StaleJobThreshold: 10 * time.Minute,
			},
			wantErr: true,
		},
		{
			name: "concurrency too high",
			config: Config{
				Concurrency:       101,
				PollInterval:      5 * time.Second,
				JobTimeout:        5 * time.Minute,
				ShutdownTimeout:   30 * time.Second,
				StaleJobThreshold: 10 * time.Minute,
			},
			wantErr: true,
		},
		{
			name: "poll interval too short",
			config: Config{
				Concurrency:       2,
				PollInterval:      500 * time.Millisecond,
				JobTimeout:        5 * time.Minute,
				ShutdownTimeout:   30 * time.Second,
				StaleJobThreshold: 10 * time.Minute,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Config.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestIsPermanent(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "permanent error",
			err:  NewPermanentError(context.Canceled),
			want: true,
		},
		{
			name: "regular error",
			err:  context.Canceled,
			want: false,
		},
		{
			name: "nil error",
			err:  nil,
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsPermanent(tt.err); got != tt.want {
				t.Errorf("IsPermanent() = %v, want %v", got, tt.want)
			}
		})
	}
}
