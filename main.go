package main

import (
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/gorilla/mux"
	_ "github.com/mattn/go-sqlite3"
)

type Database struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	WorkspaceID int    `json:"workspace_id"`
}

type Table struct {
	ID         int    `json:"id"`
	Name       string `json:"name"`
	DatabaseID int    `json:"database_id"`
}

type Field struct {
	ID      int    `json:"id"`
	Name    string `json:"name"`
	Type    string `json:"type"`
	TableID int    `json:"table_id"`
}

var metaDB *sql.DB

var workspaceBasePath string

var metaDBPath string

func destroyDatabase(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	var workspaceID int
	err := metaDB.QueryRow("SELECT workspace_id FROM databases WHERE id = ?", id).Scan(&workspaceID)
	if err != nil {
		log.Fatalf("destroyDatabase: Error querying workspace_id: %v", err)
		return
	}

	_, err = metaDB.Exec("DELETE FROM databases WHERE id = ?", id)
	if err != nil {
		log.Fatalf("destroyDatabase: Error deleting from databases: %v", err)
		return
	}

	// Remove physical SQLite database
	dbPath := filepath.Join(workspaceBasePath, fmt.Sprintf("%d", workspaceID), fmt.Sprintf("%s.sqlite", id))
	os.Remove(dbPath)

	w.WriteHeader(http.StatusNoContent)
}

func getData(w http.ResponseWriter, r *http.Request) {
	databaseID := r.URL.Query().Get("database_id")
	if databaseID == "" {
		http.Error(w, "database_id is required", http.StatusBadRequest)
		return
	}

	query := r.URL.Query().Get("query")
	if query == "" {
		http.Error(w, "query is required", http.StatusBadRequest)
		return
	}

	if !strings.HasPrefix(strings.ToUpper(strings.TrimSpace(query)), "SELECT") {
		http.Error(w, "Only SELECT queries are allowed", http.StatusBadRequest)
		return
	}

	var workspaceID int
	err := metaDB.QueryRow("SELECT workspace_id FROM databases WHERE id = ?", databaseID).Scan(&workspaceID)
	if err != nil {
		log.Fatalf("getData: Error querying workspace_id: %v", err)
		return
	}

	dbPath := filepath.Join(workspaceBasePath, fmt.Sprintf("%d", workspaceID), fmt.Sprintf("%s.sqlite", databaseID))
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		log.Fatalf("getData: Error opening SQLite database: %v", err)
		return
	}
	defer db.Close()

	rows, err := db.Query(query)
	if err != nil {
		log.Fatalf("getData: Error executing query: %v", err)
		return
	}
	defer rows.Close()

	columns, _ := rows.Columns()
	var result []map[string]interface{}

	for rows.Next() {
		values := make([]interface{}, len(columns))
		valuePointers := make([]interface{}, len(columns))
		for i := range columns {
			valuePointers[i] = &values[i]
		}

		err := rows.Scan(valuePointers...)
		if err != nil {
			log.Fatalf("getData: Error scanning row: %v", err)
			return
		}

		row := make(map[string]interface{})
		for i, col := range columns {
			val := values[i]
			b, ok := val.([]byte)
			if ok {
				row[col] = string(b)
			} else {
				row[col] = val
			}
		}

		result = append(result, row)
	}

	json.NewEncoder(w).Encode(result)
}

func updateField(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	var field Field
	json.NewDecoder(r.Body).Decode(&field)

	_, err := metaDB.Exec("UPDATE fields SET name = ?, type = ? WHERE id = ?",
		field.Name, field.Type, id)
	if err != nil {
		log.Fatalf("updateField: Error updating field: %v", err)
		return
	}

	// Update column in physical SQLite database
	var tableID, databaseID, workspaceID int
	var oldFieldName, tableName string
	err = metaDB.QueryRow("SELECT name, table_id FROM fields WHERE id = ?", id).Scan(&oldFieldName, &tableID)
	if err != nil {
		log.Fatalf("updateField: Error querying old field name and table_id: %v", err)
		return
	}

	err = metaDB.QueryRow("SELECT database_id, name FROM tables WHERE id = ?", tableID).Scan(&databaseID, &tableName)
	if err != nil {
		log.Fatalf("updateField: Error querying database_id and table name: %v", err)
		return
	}

	err = metaDB.QueryRow("SELECT workspace_id FROM databases WHERE id = ?", databaseID).Scan(&workspaceID)
	if err != nil {
		log.Fatalf("updateField: Error querying workspace_id: %v", err)
		return
	}

	dbPath := filepath.Join(workspaceBasePath, fmt.Sprintf("%d", workspaceID), fmt.Sprintf("%d.sqlite", databaseID))
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		log.Fatalf("updateField: Error opening SQLite database: %v", err)
		return
	}
	defer db.Close()

	// SQLite doesn't support changing column types directly, so we need to recreate the table
	_, err = db.Exec(fmt.Sprintf(`
		CREATE TABLE %s_new (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			%s %s,
			%s
		);
		INSERT INTO %s_new SELECT * FROM %s;
		DROP TABLE %s;
		ALTER TABLE %s_new RENAME TO %s;
	`, tableName, field.Name, field.Type,
		getOtherColumnsSQL(tableID, id),
		tableName, tableName, tableName, tableName, tableName))
	if err != nil {
		log.Fatalf("updateField: Error recreating table: %v", err)
		return
	}

	json.NewEncoder(w).Encode(field)
}

func createField(w http.ResponseWriter, r *http.Request) {
	var field Field
	json.NewDecoder(r.Body).Decode(&field)

	result, err := metaDB.Exec("INSERT INTO fields (name, type, table_id) VALUES (?, ?, ?)",
		field.Name, field.Type, field.TableID)
	if err != nil {
		log.Fatalf("createField: Error inserting into fields: %v", err)
		return
	}

	id, _ := result.LastInsertId()
	field.ID = int(id)

	// Add column to physical SQLite database
	var databaseID, workspaceID int
	var tableName string
	err = metaDB.QueryRow("SELECT database_id, name FROM tables WHERE id = ?", field.TableID).Scan(&databaseID, &tableName)
	if err != nil {
		log.Fatalf("createField: Error querying database_id and table name: %v", err)
		return
	}

	err = metaDB.QueryRow("SELECT workspace_id FROM databases WHERE id = ?", databaseID).Scan(&workspaceID)
	if err != nil {
		log.Fatalf("createField: Error querying workspace_id: %v", err)
		return
	}

	dbPath := filepath.Join(workspaceBasePath, fmt.Sprintf("%d", workspaceID), fmt.Sprintf("%d.sqlite", databaseID))
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		log.Fatalf("createField: Error opening SQLite database: %v", err)
		return
	}
	defer db.Close()

	_, err = db.Exec(fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s %s", tableName, field.Name, field.Type))
	if err != nil {
		log.Fatalf("createField: Error altering table: %v", err)
		return
	}

	json.NewEncoder(w).Encode(field)
}

func addUpdateData(w http.ResponseWriter, r *http.Request) {
	var data map[string]interface{}
	json.NewDecoder(r.Body).Decode(&data)

	databaseID, ok := data["database_id"].(float64)
	if !ok {
		http.Error(w, "database_id is required", http.StatusBadRequest)
		return
	}

	tableName, ok := data["table_name"].(string)
	if !ok {
		http.Error(w, "table_name is required", http.StatusBadRequest)
		return
	}

	delete(data, "database_id")
	delete(data, "table_name")

	var workspaceID int
	err := metaDB.QueryRow("SELECT workspace_id FROM databases WHERE id = ?", int(databaseID)).Scan(&workspaceID)
	if err != nil {
		log.Fatalf("addUpdateData: Error querying workspace_id: %v", err)
		return
	}

	dbPath := filepath.Join(workspaceBasePath, fmt.Sprintf("%d", workspaceID), fmt.Sprintf("%d.sqlite", int(databaseID)))
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		log.Fatalf("addUpdateData: Error opening database: %v", err)
		return
	}
	defer db.Close()

	columns := make([]string, 0, len(data))
	values := make([]interface{}, 0, len(data))
	placeholders := make([]string, 0, len(data))

	for k, v := range data {
		columns = append(columns, k)
		values = append(values, v)
		placeholders = append(placeholders, "?")
	}

	query := fmt.Sprintf("INSERT OR REPLACE INTO %s (%s) VALUES (%s)",
		tableName, strings.Join(columns, ", "), strings.Join(placeholders, ", "))

	_, err = db.Exec(query, values...)
	if err != nil {
		log.Fatalf("addUpdateData: Error executing query: %v", err)
		return
	}

	w.WriteHeader(http.StatusCreated)
}

func createDatabase(w http.ResponseWriter, r *http.Request) {
	var db Database
	json.NewDecoder(r.Body).Decode(&db)

	result, err := metaDB.Exec("INSERT INTO databases (name, description, workspace_id) VALUES (?, ?, ?)",
		db.Name, db.Description, db.WorkspaceID)
	if err != nil {
		log.Fatalf("createDatabase: Error inserting into databases: %v", err)
		return
	}

	id, _ := result.LastInsertId()
	db.ID = int(id)

	// Create physical SQLite database
	dbPath := filepath.Join(workspaceBasePath, fmt.Sprintf("%d", db.WorkspaceID), fmt.Sprintf("%d.sqlite", db.ID))
	os.MkdirAll(filepath.Dir(dbPath), os.ModePerm)
	_, err = sql.Open("sqlite3", dbPath)
	if err != nil {
		log.Fatalf("createDatabase: Error creating SQLite database: %v", err)
		return
	}

	json.NewEncoder(w).Encode(db)
}

func createTable(w http.ResponseWriter, r *http.Request) {
	var table Table
	json.NewDecoder(r.Body).Decode(&table)

	result, err := metaDB.Exec("INSERT INTO tables (name, database_id) VALUES (?, ?)",
		table.Name, table.DatabaseID)
	if err != nil {
		log.Fatalf("createTable: Error inserting into tables: %v", err)
		return
	}

	id, _ := result.LastInsertId()
	table.ID = int(id)

	// Create table in physical SQLite database
	var workspaceID int
	err = metaDB.QueryRow("SELECT workspace_id FROM databases WHERE id = ?", table.DatabaseID).Scan(&workspaceID)
	if err != nil {
		log.Fatalf("createTable: Error querying workspace_id: %v", err)
		return
	}

	dbPath := filepath.Join(workspaceBasePath, fmt.Sprintf("%d", workspaceID), fmt.Sprintf("%d.sqlite", table.DatabaseID))
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		log.Fatalf("createTable: Error opening SQLite database: %v", err)
		return
	}
	defer db.Close()

	_, err = db.Exec(fmt.Sprintf("CREATE TABLE %s (id INTEGER PRIMARY KEY AUTOINCREMENT)", table.Name))
	if err != nil {
		log.Fatalf("createTable: Error creating table: %v", err)
		return
	}

	json.NewEncoder(w).Encode(table)
}

func listTables(w http.ResponseWriter, r *http.Request) {
	databaseID := r.URL.Query().Get("database_id")
	if databaseID == "" {
		http.Error(w, "database_id is required", http.StatusBadRequest)
		return
	}

	rows, err := metaDB.Query("SELECT id, name FROM tables WHERE database_id = ?", databaseID)
	if err != nil {
		log.Fatalf("listTables: Error querying tables: %v", err)
		return
	}
	defer rows.Close()

	var tables []Table
	for rows.Next() {
		var table Table
		err := rows.Scan(&table.ID, &table.Name)
		if err != nil {
			log.Fatalf("listTables: Error scanning row: %v", err)
			return
		}
		table.DatabaseID, _ = strconv.Atoi(databaseID)
		tables = append(tables, table)
	}

	json.NewEncoder(w).Encode(tables)
}

func main() {
	workspaceBasePath = os.Getenv("WORKSPACE_BASE_PATH")
	if workspaceBasePath == "" {
		workspaceBasePath = "/home/workspaces"
	}

	// Create workspaceBasePath if it doesn't exist
	err := os.MkdirAll(workspaceBasePath, os.ModePerm)
	if err != nil {
		log.Fatalf("Failed to create workspaceBasePath: %v", err)
	}

	initMetaDB("metadb.db")
	defer metaDB.Close()

	// Define the port flag
	port := flag.Int("port", 8080, "Port to run the server on")
	flag.Parse()

	r := mux.NewRouter()
	r.HandleFunc("/databases", createDatabase).Methods("POST")
	r.HandleFunc("/databases", listDatabases).Methods("GET")
	r.HandleFunc("/databases/{id}", destroyDatabase).Methods("DELETE")
	r.HandleFunc("/tables", createTable).Methods("POST")
	r.HandleFunc("/tables", listTables).Methods("GET")
	r.HandleFunc("/tables/{id}", destroyTable).Methods("DELETE")
	r.HandleFunc("/fields", createField).Methods("POST")
	r.HandleFunc("/fields", listFields).Methods("GET")
	r.HandleFunc("/fields/{id}", destroyField).Methods("DELETE")
	r.HandleFunc("/fields/{id}", updateField).Methods("PUT")
	r.HandleFunc("/data", addUpdateData).Methods("POST")
	r.HandleFunc("/data", getData).Methods("GET")

	log.Printf("Server starting on :%d", *port)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", *port), r))
}

func initMetaDB(db string) {
	var err error
	metaDBPath = filepath.Join(workspaceBasePath, db)
	metaDB, err = sql.Open("sqlite3", metaDBPath)
	if err != nil {
		log.Fatalf("initMetaDB: Error opening metaDB: %v", err)
	}

	_, err = metaDB.Exec(`
		CREATE TABLE IF NOT EXISTS databases (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			description TEXT,
			workspace_id INTEGER NOT NULL
		);
		CREATE TABLE IF NOT EXISTS tables (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			database_id INTEGER NOT NULL,
			FOREIGN KEY (database_id) REFERENCES databases (id)
		);
		CREATE TABLE IF NOT EXISTS fields (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			type TEXT NOT NULL,
			table_id INTEGER NOT NULL,
			FOREIGN KEY (table_id) REFERENCES tables (id)
		);
	`)
	if err != nil {
		log.Fatalf("initMetaDB: Error creating tables: %v", err)
	}
}

func destroyField(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	var tableID, databaseID, workspaceID int
	var fieldName, tableName string
	err := metaDB.QueryRow("SELECT name, table_id FROM fields WHERE id = ?", id).Scan(&fieldName, &tableID)
	if err != nil {
		log.Fatalf("destroyField: Error querying field name and table_id: %v", err)
		return
	}

	err = metaDB.QueryRow("SELECT database_id, name FROM tables WHERE id = ?", tableID).Scan(&databaseID, &tableName)
	if err != nil {
		log.Fatalf("destroyField: Error querying database_id and table name: %v", err)
		return
	}

	err = metaDB.QueryRow("SELECT workspace_id FROM databases WHERE id = ?", databaseID).Scan(&workspaceID)
	if err != nil {
		log.Fatalf("destroyField: Error querying workspace_id: %v", err)
		return
	}

	_, err = metaDB.Exec("DELETE FROM fields WHERE id = ?", id)
	if err != nil {
		log.Fatalf("destroyField: Error deleting from fields: %v", err)
		return
	}

	// Remove column from physical SQLite database
	dbPath := filepath.Join(workspaceBasePath, fmt.Sprintf("%d", workspaceID), fmt.Sprintf("%d.sqlite", databaseID))
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		log.Fatalf("destroyField: Error opening SQLite database: %v", err)
		return
	}
	defer db.Close()

	// SQLite doesn't support dropping columns directly, so we need to recreate the table
	_, err = db.Exec(fmt.Sprintf(`
		CREATE TABLE %s_new AS SELECT * FROM %s;
		DROP TABLE %s;
		ALTER TABLE %s_new RENAME TO %s;
	`, tableName, tableName, tableName, tableName, tableName))
	if err != nil {
		log.Fatalf("destroyField: Error recreating table: %v", err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func getOtherColumnsSQL(tableID int, excludeFieldID string) string {
	rows, err := metaDB.Query("SELECT name, type FROM fields WHERE table_id = ? AND id != ?", tableID, excludeFieldID)
	if err != nil {
		log.Fatalf("getOtherColumnsSQL: Error querying fields: %v", err)
		return ""
	}
	defer rows.Close()

	var columns []string
	for rows.Next() {
		var name, fieldType string
		err := rows.Scan(&name, &fieldType)
		if err != nil {
			log.Fatalf("getOtherColumnsSQL: Error scanning row: %v", err)
			return ""
		}
		columns = append(columns, fmt.Sprintf("%s %s", name, fieldType))
	}

	return strings.Join(columns, ", ")
}

func listDatabases(w http.ResponseWriter, r *http.Request) {
	workspaceID := r.URL.Query().Get("workspace_id")
	if workspaceID == "" {
		http.Error(w, "workspace_id is required", http.StatusBadRequest)
		return
	}

	rows, err := metaDB.Query("SELECT id, name, description FROM databases WHERE workspace_id = ?", workspaceID)
	if err != nil {
		log.Fatalf("listDatabases: Error querying databases: %v", err)
		return
	}
	defer rows.Close()

	var databases []Database
	for rows.Next() {
		var db Database
		err := rows.Scan(&db.ID, &db.Name, &db.Description)
		if err != nil {
			log.Fatalf("listDatabases: Error scanning row: %v", err)
			return
		}
		db.WorkspaceID, _ = strconv.Atoi(workspaceID)
		databases = append(databases, db)
	}

	json.NewEncoder(w).Encode(databases)
}

func destroyTable(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	var databaseID, workspaceID int
	var tableName string
	err := metaDB.QueryRow("SELECT database_id, name FROM tables WHERE id = ?", id).Scan(&databaseID, &tableName)
	if err != nil {
		log.Fatalf("destroyTable: Error querying database_id and table name: %v", err)
		return
	}

	err = metaDB.QueryRow("SELECT workspace_id FROM databases WHERE id = ?", databaseID).Scan(&workspaceID)
	if err != nil {
		log.Fatalf("destroyTable: Error querying workspace_id: %v", err)
		return
	}

	_, err = metaDB.Exec("DELETE FROM tables WHERE id = ?", id)
	if err != nil {
		log.Fatalf("destroyTable: Error deleting from tables: %v", err)
		return
	}

	// Drop table in physical SQLite database
	dbPath := filepath.Join(workspaceBasePath, fmt.Sprintf("%d", workspaceID), fmt.Sprintf("%d.sqlite", databaseID))
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		log.Fatalf("destroyTable: Error opening SQLite database: %v", err)
		return
	}
	defer db.Close()

	_, err = db.Exec(fmt.Sprintf("DROP TABLE IF EXISTS %s", tableName))
	if err != nil {
		log.Fatalf("destroyTable: Error dropping table: %v", err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func listFields(w http.ResponseWriter, r *http.Request) {
	tableID := r.URL.Query().Get("table_id")
	if tableID == "" {
		http.Error(w, "table_id is required", http.StatusBadRequest)
		return
	}

	rows, err := metaDB.Query("SELECT id, name, type FROM fields WHERE table_id = ?", tableID)
	if err != nil {
		log.Fatalf("listFields: Error querying fields: %v", err)
		return
	}
	defer rows.Close()

	var fields []Field
	for rows.Next() {
		var field Field
		err := rows.Scan(&field.ID, &field.Name, &field.Type)
		if err != nil {
			log.Fatalf("listFields: Error scanning row: %v", err)
			return
		}
		field.TableID, _ = strconv.Atoi(tableID)
		fields = append(fields, field)
	}

	json.NewEncoder(w).Encode(fields)
}
