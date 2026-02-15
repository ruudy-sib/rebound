package valueobject

import (
	"testing"
)

func TestNewTaskID(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{
			name:    "valid ID",
			input:   "task-1",
			want:    "task-1",
			wantErr: false,
		},
		{
			name:    "ID with spaces is trimmed",
			input:   "  task-2  ",
			want:    "task-2",
			wantErr: false,
		},
		{
			name:    "empty string returns error",
			input:   "",
			want:    "",
			wantErr: true,
		},
		{
			name:    "whitespace only returns error",
			input:   "   ",
			want:    "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewTaskID(tt.input)
			if (err != nil) != tt.wantErr {
				t.Fatalf("NewTaskID(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
			if !tt.wantErr && got.String() != tt.want {
				t.Fatalf("NewTaskID(%q).String() = %q, want %q", tt.input, got.String(), tt.want)
			}
		})
	}
}

func TestTaskID_Equals(t *testing.T) {
	id1, _ := NewTaskID("task-1")
	id2, _ := NewTaskID("task-1")
	id3, _ := NewTaskID("task-2")

	if !id1.Equals(id2) {
		t.Fatal("expected task-1 to equal task-1")
	}
	if id1.Equals(id3) {
		t.Fatal("expected task-1 to not equal task-2")
	}
}
