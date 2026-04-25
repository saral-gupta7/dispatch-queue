package task

import "testing"

func TestStatusIsValid(t *testing.T) {
	tests := []struct {
		name   string
		status Status
		want   bool
	}{
		{name: "pending is valid", status: StatusPending, want: true},
		{name: "running is valid", status: StatusRunning, want: true},
		{name: "completed is valid", status: StatusCompleted, want: true},
		{name: "failed is valid", status: StatusFailed, want: true},
		{name: "dead is valid", status: StatusDead, want: true},
		{name: "unknown status is invalid", status: Status("banana"), want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.status.IsValid()

			if got != tt.want {
				t.Fatalf("IsValid() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTaskIsTerminal(t *testing.T) {
	tests := []struct {
		name string
		task Task
		want bool
	}{
		{name: "completed is terminal", task: Task{Status: StatusCompleted}, want: true},
		{name: "dead is terminal", task: Task{Status: StatusDead}, want: true},
		{name: "pending is not terminal", task: Task{Status: StatusPending}, want: false},
		{name: "running is not terminal", task: Task{Status: StatusRunning}, want: false},
		{name: "failed is not terminal", task: Task{Status: StatusFailed}, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.task.IsTerminal()

			if got != tt.want {
				t.Fatalf("IsTerminal() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTaskCanRetry(t *testing.T) {
	tests := []struct {
		name string
		task Task
		want bool
	}{
		{
			name: "zero attempts used can retry",
			task: Task{Attempts: 0, MaxAttempts: 3},
			want: true,
		},
		{
			name: "some attempts remaining can retry",
			task: Task{Attempts: 2, MaxAttempts: 3},
			want: true,
		},
		{
			name: "max attempts reached cannot retry",
			task: Task{Attempts: 3, MaxAttempts: 3},
			want: false,
		},
		{
			name: "attempts above max cannot retry",
			task: Task{Attempts: 4, MaxAttempts: 3},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.task.CanRetry()

			if got != tt.want {
				t.Fatalf("CanRetry() = %v, want %v", got, tt.want)
			}
		})
	}
}
