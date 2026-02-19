package router

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/OpenNSW/nsw/internal/auth"
	"github.com/OpenNSW/nsw/internal/workflow/service"
)

func setupRouterTestDB(t *testing.T) (*gorm.DB, sqlmock.Sqlmock) {
	mockDB, sqlMock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}

	dialector := postgres.New(postgres.Config{
		Conn:       mockDB,
		DriverName: "postgres",
	})

	db, err := gorm.Open(dialector, &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a gorm database connection", err)
	}

	return db, sqlMock
}

// Mock Helper to create context with Auth
func withAuthContext(ctx context.Context, traderID string) context.Context {
	authCtx := &auth.AuthContext{
		TraderContext: &auth.TraderContext{
			TraderID:      traderID,
			TraderContext: json.RawMessage(`{}`),
		},
	}
	return context.WithValue(ctx, auth.AuthContextKey, authCtx)
}

// --- ConsignmentRouter Tests ---

func TestConsignmentRouter_HandleGetConsignmentByID(t *testing.T) {
	db, sqlMock := setupRouterTestDB(t)
	// Mock dependencies for Service
	// We can pass nil for repository mocks if we don't trigger them (GetConsignmentByID uses DB directly mostly)
	svc := service.NewConsignmentService(db, nil, nil)
	// Pass nil for second argument of NewConsignmentRouter
	r := NewConsignmentRouter(svc, nil)

	consignmentID := uuid.New()

	// Expectation
	sqlMock.ExpectQuery(`SELECT \* FROM "consignments" WHERE id = \$1`).
		WithArgs(consignmentID, 1).
		WillReturnRows(sqlmock.NewRows([]string{"id", "state"}).AddRow(consignmentID, "IN_PROGRESS"))

	// Preload WorkflowNodes
	sqlMock.ExpectQuery(`SELECT \* FROM "workflow_nodes" WHERE "workflow_nodes"."consignment_id" = \$1`).
		WithArgs(consignmentID).
		WillReturnRows(sqlmock.NewRows([]string{"id", "consignment_id"}))

	// Request
	req, _ := http.NewRequest("GET", "/api/v1/consignments/"+consignmentID.String(), nil)
	req.SetPathValue("id", consignmentID.String())

	w := httptest.NewRecorder()
	r.HandleGetConsignmentByID(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestConsignmentRouter_HandleGetConsignmentsByTraderID(t *testing.T) {
	db, sqlMock := setupRouterTestDB(t)
	svc := service.NewConsignmentService(db, nil, nil)
	r := NewConsignmentRouter(svc, nil)

	traderID := "trader1"

	// Expectation
	sqlMock.ExpectQuery(`SELECT count\(\*\) FROM "consignments" WHERE trader_id = \$1`).
		WithArgs(traderID).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

	// Allow for offset/limit/order in query
	sqlMock.ExpectQuery(`SELECT \* FROM "consignments" WHERE trader_id = \$1`).
		WithArgs(traderID, 50).
		WillReturnRows(sqlmock.NewRows([]string{"id", "trader_id"}).AddRow(uuid.New(), traderID))

	// Preload WorkflowNodes (Count aggregation)
	// Query: SELECT consignment_id, count(*) as total, count(case when state = $1 then 1 end) as completed FROM "workflow_nodes" WHERE consignment_id IN ($2) GROUP BY "consignment_id"
	sqlMock.ExpectQuery(`SELECT consignment_id, count\(\*\) as total, count\(case when state = \$1 then 1 end\) as completed FROM "workflow_nodes" WHERE consignment_id IN \(\$2\) GROUP BY "consignment_id"`).
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{"consignment_id", "total", "completed"}).AddRow(uuid.New(), 1, 0))

	// Request with Auth
	req, _ := http.NewRequest("GET", "/api/v1/consignments", nil)
	req = req.WithContext(withAuthContext(req.Context(), traderID))

	w := httptest.NewRecorder()
	r.HandleGetConsignmentsByTraderID(w, req)

	if w.Code != http.StatusOK {
		t.Logf("Expected 200, got %d. Body: %s", w.Code, w.Body.String())
	}
	assert.Equal(t, http.StatusOK, w.Code)
}

// --- HSCodeRouter Tests ---

func TestHSCodeRouter_HandleGetAllHSCodes(t *testing.T) {
	db, sqlMock := setupRouterTestDB(t)
	svc := service.NewHSCodeService(db)
	r := NewHSCodeRouter(svc)

	// Expectation
	sqlMock.ExpectQuery(`SELECT count\(\*\) FROM "hs_codes"`).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

	sqlMock.ExpectQuery(`SELECT \* FROM "hs_codes" ORDER BY hs_code ASC LIMIT \$1`).
		WithArgs(50).
		WillReturnRows(sqlmock.NewRows([]string{"id", "hs_code"}).AddRow(uuid.New(), "1234.56"))

	req, _ := http.NewRequest("GET", "/api/v1/hscodes", nil)
	w := httptest.NewRecorder()
	r.HandleGetAllHSCodes(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

// --- PreConsignmentRouter Tests ---

func TestPreConsignmentRouter_HandleGetPreConsignmentByID(t *testing.T) {
	db, sqlMock := setupRouterTestDB(t)
	svc := service.NewPreConsignmentService(db, nil, nil)
	r := NewPreConsignmentRouter(svc)

	id := uuid.New()

	templateID := uuid.New()
	// Expectation
	sqlMock.ExpectQuery(`SELECT \* FROM "pre_consignments" WHERE id = \$1`).
		WithArgs(id, 1).
		WillReturnRows(sqlmock.NewRows([]string{"id", "pre_consignment_template_id"}).AddRow(id, templateID))

	// Preload PreConsignmentTemplate
	sqlMock.ExpectQuery(`SELECT \* FROM "pre_consignment_templates" WHERE "pre_consignment_templates"."id" = \$1`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "name"}).AddRow(templateID, "Template"))

	// Preload WorkflowNodes
	sqlMock.ExpectQuery(`SELECT \* FROM "workflow_nodes" WHERE "workflow_nodes"."pre_consignment_id" = \$1`).
		WithArgs(id).
		WillReturnRows(sqlmock.NewRows([]string{"id", "pre_consignment_id"}))

	req, _ := http.NewRequest("GET", "/api/v1/pre-consignments/"+id.String(), nil)
	req.SetPathValue("preConsignmentId", id.String())

	w := httptest.NewRecorder()
	r.HandleGetPreConsignmentByID(w, req)

	if w.Code != http.StatusOK {
		t.Logf("Expected 200, got %d. Body: %s", w.Code, w.Body.String())
	}
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestPreConsignmentRouter_HandleGetPreConsignmentsByTraderID(t *testing.T) {
	db, sqlMock := setupRouterTestDB(t)
	svc := service.NewPreConsignmentService(db, nil, nil)
	r := NewPreConsignmentRouter(svc)

	traderID := "trader1"

	templateID := uuid.New()
	// Expectation
	sqlMock.ExpectQuery(`SELECT \* FROM "pre_consignments" WHERE trader_id = \$1 AND state != \$2`).
		WithArgs(traderID, "LOCKED").
		WillReturnRows(sqlmock.NewRows([]string{"id", "trader_id", "pre_consignment_template_id"}).
			AddRow(uuid.New(), traderID, templateID))

	// Preload PreConsignmentTemplate
	sqlMock.ExpectQuery(`SELECT \* FROM "pre_consignment_templates" WHERE "pre_consignment_templates"."id" = \$1`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "name"}).AddRow(templateID, "Template"))

	// Preload WorkflowNodes
	sqlMock.ExpectQuery(`SELECT \* FROM "workflow_nodes" WHERE "workflow_nodes"."pre_consignment_id" = \$1`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "pre_consignment_id"}))

	req, _ := http.NewRequest("GET", "/api/v1/pre-consignments", nil)
	req = req.WithContext(withAuthContext(req.Context(), traderID))

	w := httptest.NewRecorder()
	r.HandleGetPreConsignmentsByTraderID(w, req)

	if w.Code != http.StatusOK {
		t.Logf("Expected 200, got %d. Body: %s", w.Code, w.Body.String())
	}
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestPreConsignmentRouter_HandleCreatePreConsignment(t *testing.T) {
	db, _ := setupRouterTestDB(t)
	svc := service.NewPreConsignmentService(db, nil, nil)
	r := NewPreConsignmentRouter(svc)

	req, _ := http.NewRequest("POST", "/api/v1/pre-consignments", nil)
	// No Auth
	w := httptest.NewRecorder()
	r.HandleCreatePreConsignment(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestPreConsignmentRouter_HandleGetTraderPreConsignments(t *testing.T) {
	db, sqlMock := setupRouterTestDB(t)
	svc := service.NewPreConsignmentService(db, nil, nil)
	r := NewPreConsignmentRouter(svc)

	traderID := "trader1"

	// Expectation for GetTraderPreConsignments (Templates)
	sqlMock.ExpectQuery(`SELECT count\(\*\) FROM "pre_consignment_templates"`).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

	sqlMock.ExpectQuery(`SELECT \* FROM "pre_consignment_templates" ORDER BY name ASC LIMIT \$1`).
		WithArgs(50).
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "workflow_template_id"}).
			AddRow(uuid.New(), "Template", uuid.New()))

	// Fetch existing pre-consignments
	sqlMock.ExpectQuery(`SELECT \* FROM "pre_consignments" WHERE trader_id = \$1`).
		WithArgs(traderID).
		WillReturnRows(sqlmock.NewRows([]string{"id", "trader_id"}).AddRow(uuid.New(), traderID))

	req, _ := http.NewRequest("GET", "/api/v1/pre-consignments/templates", nil)
	req = req.WithContext(withAuthContext(req.Context(), traderID))

	w := httptest.NewRecorder()
	r.HandleGetTraderPreConsignments(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestConsignmentRouter_HandleCreateConsignment(t *testing.T) {
	db, _ := setupRouterTestDB(t)
	svc := service.NewConsignmentService(db, nil, nil)
	r := NewConsignmentRouter(svc, nil)

	// Test Invalid Body
	req, _ := http.NewRequest("POST", "/api/v1/consignments", bytes.NewBufferString("invalid json"))
	req = req.WithContext(withAuthContext(req.Context(), "trader1"))
	w := httptest.NewRecorder()
	r.HandleCreateConsignment(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)

	// Test Empty Items (Service Validation Failure)
	payload := `{"items": [], "flow": "IMPORT"}`
	req, _ = http.NewRequest("POST", "/api/v1/consignments", bytes.NewBufferString(payload))
	req = req.WithContext(withAuthContext(req.Context(), "trader1"))
	w = httptest.NewRecorder()
	r.HandleCreateConsignment(w, req)
	// Service validation expects non-empty items.
	// Router calls svc.InitializeConsignment.
	// If svc returns error, Router returns 500.
	assert.NotEqual(t, http.StatusOK, w.Code)
}
