package internal

import (
	"fmt"
	"testing"
)

func TestNewDBConnector(t *testing.T) {
	tests := []struct {
		name        string
		cfg         Config
		wantErr     bool
		expectedType string
	}{
		{
			name: "valid sqlite",
			cfg: Config{DBDriver: "sqlite", DBPath: ":memory:"},
			wantErr: false,
			expectedType: "*internal.SQLiteConnector",
		},
		{
			name: "valid postgres",
			cfg: Config{DBDriver: "postgres"},
			wantErr: false,
			expectedType: "*internal.PostgresConnector",
		},
		{
			name: "invalid driver",
			cfg: Config{DBDriver: "mysql"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			connector, err := NewDBConnector(tt.cfg)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewDBConnector() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				// Assert type
				if gotType := fmt.Sprintf("%T", connector); gotType != tt.expectedType {
					t.Errorf("NewDBConnector() = %v, want %v", gotType, tt.expectedType)
				}
			}
		})
	}
}
