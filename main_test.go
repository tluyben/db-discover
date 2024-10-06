package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestMain(m *testing.M) {
	// Setup
	setupTestEnvironment()

	// Run tests
	code := m.Run()

	// Teardown
	cleanupTestEnvironment()

	// Exit with the test result code
	os.Exit(code)
}

func TestDestroyField(t *testing.T) {
	req, _ := http.NewRequest("DELETE", "/fields/1", nil)
	rr := httptest.NewRecorder()

	handler := http.HandlerFunc(destroyField)
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusNoContent {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusNoContent)
	}
}

func TestDestroyTable(t *testing.T) {
	req, _ := http.NewRequest("DELETE", "/tables/1", nil)
	rr := httptest.NewRecorder()

	handler := http.HandlerFunc(destroyTable)
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusNoContent {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusNoContent)
	}
}

func TestGetData(t *testing.T) {
	req, _ := http.NewRequest("GET", "/data?database_id=1&query=SELECT * FROM TestTable", nil)
	rr := httptest.NewRecorder()

	handler := http.HandlerFunc(getData)
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	var result []map[string]interface{}
	json.Unmarshal(rr.Body.Bytes(), &result)

	if len(result) == 0 {
		t.Errorf("handler returned no data")
	}
}

func TestUpdateField(t *testing.T) {
	field := Field{
		Name: "UpdatedField",
		Type: "INTEGER",
	}

	body, _ := json.Marshal(field)
	req, _ := http.NewRequest("PUT", "/fields/1", bytes.NewBuffer(body))
	rr := httptest.NewRecorder()

	handler := http.HandlerFunc(updateField)
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	var updatedField Field
	json.Unmarshal(rr.Body.Bytes(), &updatedField)

	if updatedField.Name != field.Name || updatedField.Type != field.Type {
		t.Errorf("handler returned unexpected body: got %v want %v", updatedField, field)
	}
}

func TestCreateField(t *testing.T) {
	field := Field{
		Name:    "TestField",
		Type:    "TEXT",
		TableID: 1,
	}

	body, _ := json.Marshal(field)
	req, _ := http.NewRequest("POST", "/fields", bytes.NewBuffer(body))
	rr := httptest.NewRecorder()

	handler := http.HandlerFunc(createField)
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	var createdField Field
	json.Unmarshal(rr.Body.Bytes(), &createdField)

	if createdField.Name != field.Name {
		t.Errorf("handler returned unexpected body: got %v want %v", createdField.Name, field.Name)
	}
}

func cleanupTestEnvironment() {
	metaDB.Close()
	os.Remove(metaDBPath)
}

func setupTestEnvironment() {
	workspaceBasePath = os.TempDir()
	metaDBPath = filepath.Join(workspaceBasePath, "metadb.db")

	// Remove existing test database if it exists
	os.Remove(metaDBPath)

	initMetaDB("test_metadb.db")

	// Create a test database
	_, err := metaDB.Exec("INSERT INTO databases (name, description, workspace_id) VALUES (?, ?, ?)", "TestDB", "Test Database", 1)
	if err != nil {
		panic(err)
	}

	// Create a test table
	_, err = metaDB.Exec("INSERT INTO tables (name, database_id) VALUES (?, ?)", "TestTable", 1)
	if err != nil {
		panic(err)
	}

	// Create a test field
	_, err = metaDB.Exec("INSERT INTO fields (name, type, table_id) VALUES (?, ?, ?)", "TestField", "TEXT", 1)
	if err != nil {
		panic(err)
	}
}

func TestListFields(t *testing.T) {
	req, _ := http.NewRequest("GET", "/fields?table_id=1", nil)
	rr := httptest.NewRecorder()

	handler := http.HandlerFunc(listFields)
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	var fields []Field
	json.Unmarshal(rr.Body.Bytes(), &fields)

	if len(fields) == 0 {
		t.Errorf("handler returned no fields")
	}
}

func TestDestroyDatabase(t *testing.T) {
	req, _ := http.NewRequest("DELETE", "/databases/1", nil)
	rr := httptest.NewRecorder()

	handler := http.HandlerFunc(destroyDatabase)
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusNoContent {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusNoContent)
	}
}

func TestCreateTable(t *testing.T) {
	table := Table{
		Name:       "TestTable",
		DatabaseID: 1,
	}

	body, _ := json.Marshal(table)
	req, _ := http.NewRequest("POST", "/tables", bytes.NewBuffer(body))
	rr := httptest.NewRecorder()

	handler := http.HandlerFunc(createTable)
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	var createdTable Table
	json.Unmarshal(rr.Body.Bytes(), &createdTable)

	if createdTable.Name != table.Name {
		t.Errorf("handler returned unexpected body: got %v want %v", createdTable.Name, table.Name)
	}
}

func TestListDatabases(t *testing.T) {
	req, _ := http.NewRequest("GET", "/databases?workspace_id=1", nil)
	rr := httptest.NewRecorder()

	handler := http.HandlerFunc(listDatabases)
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	var databases []Database
	json.Unmarshal(rr.Body.Bytes(), &databases)

	if len(databases) == 0 {
		t.Errorf("handler returned no databases")
	}
}

func TestAddUpdateData(t *testing.T) {
	data := map[string]interface{}{
		"database_id": 1,
		"table_name":  "TestTable",
		"column1":     "value1",
		"column2":     42,
	}

	body, _ := json.Marshal(data)
	req, _ := http.NewRequest("POST", "/data", bytes.NewBuffer(body))
	rr := httptest.NewRecorder()

	handler := http.HandlerFunc(addUpdateData)
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusCreated {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusCreated)
	}
}

func TestListTables(t *testing.T) {
	req, _ := http.NewRequest("GET", "/tables?database_id=1", nil)
	rr := httptest.NewRecorder()

	handler := http.HandlerFunc(listTables)
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	var tables []Table
	json.Unmarshal(rr.Body.Bytes(), &tables)

	if len(tables) == 0 {
		t.Errorf("handler returned no tables")
	}
}
