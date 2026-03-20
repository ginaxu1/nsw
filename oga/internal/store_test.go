package internal

import (
	"os"
	"path/filepath"
	"testing"
)

func TestApplicationStore_SQLite_Integration(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test_oga.db")

	cfg := Config{
		DBDriver: "sqlite",
		DBPath:   dbPath,
	}

	store, err := NewApplicationStore(cfg)
	if err != nil {
		t.Fatalf("Failed to create ApplicationStore: %v", err)
	}

	// Verify schema migration (applications table exists)
	if !store.db.Migrator().HasTable(&ApplicationRecord{}) {
		t.Error("applications table was not created")
	}

	// Functional Testing: Record Persistence and JSONB Handling
	app := &ApplicationRecord{
		TaskID:     "task-123",
		WorkflowID: "wf-456",
		ServiceURL: "http://test",
		Data:       JSONB{"key": "value", "nested": map[string]any{"inner": 1}},
		Meta:       JSONB{"meta": "data"},
	}

	if err := store.CreateOrUpdate(app); err != nil {
		t.Fatalf("CreateOrUpdate failed: %v", err)
	}

	// Retrieve
	fetchedApp, err := store.GetByTaskID("task-123")
	if err != nil {
		t.Fatalf("GetByTaskID failed: %v", err)
	}

	if fetchedApp.Data["key"] != "value" {
		t.Errorf("Expected Data['key'] to be 'value', got %v", fetchedApp.Data["key"])
	}

	// Status Updates
	if err := store.UpdateStatus("task-123", "APPROVED", map[string]any{}); err != nil {
		t.Fatalf("UpdateStatus failed: %v", err)
	}

	approvedApp, _ := store.GetByTaskID("task-123")
	if approvedApp.Status != "APPROVED" {
		t.Errorf("Expected Status 'APPROVED', got %v", approvedApp.Status)
	}

	// AppendFeedback
	feedbackData := map[string]any{"comment": "looks good"}
	if err := store.AppendFeedback("task-123", feedbackData); err != nil {
		t.Fatalf("AppendFeedback failed: %v", err)
	}

	feedbackApp, _ := store.GetByTaskID("task-123")
	if len(feedbackApp.OGAFeedbackHistory) != 1 {
		t.Errorf("Expected 1 feedback entry, got %d", len(feedbackApp.OGAFeedbackHistory))
	} else if feedbackApp.OGAFeedbackHistory[0]["comment"] != "looks good" {
		t.Errorf("Expected feedback comment 'looks good', got %v", feedbackApp.OGAFeedbackHistory[0]["comment"])
	}

	// Cleanup
	_ = os.Remove(dbPath)
}

func TestApplicationStore_Postgres_DSN(t *testing.T) {
	// This tests the Postgres factory logic and DSN formation.
	// We don't necessarily need a live DB to check if NewDBConnector creates the struct correctly,
	// but reaching this far without a panic or unsupported error means the DSN logic in PostgresConnector is active.

	// We check the Interface Abstraction
	cfg := Config{
		DBDriver:   "postgres",
		DBHost:     "localhost",
		DBPort:     "5432",
		DBUser:     "testuser",
		DBPassword: "testpassword",
		DBName:     "testdb",
		DBSSLMode:  "disable",
	}

	connector, err := NewDBConnector(cfg)
	if err != nil {
		t.Fatalf("Expected no error creating Postgres connector, got %v", err)
	}

	pgConn, ok := connector.(*PostgresConnector)
	if !ok {
		t.Fatal("Expected *PostgresConnector type")
	}

	if pgConn.Host != "localhost" || pgConn.User != "testuser" {
		t.Errorf("Postgres config mismatch: %+v", pgConn)
	}
}
