# DB Discovery Service

This Go-based service manages SQLite databases within workspaces, providing REST APIs for database, table, field, and data management.

## Features

- Database management (create, list, destroy)
- Table management (create, list, destroy)
- Field management (create, list, destroy, update)
- Data management (add/update, query)
- Workspace-based SQLite file management

## Prerequisites

- Go 1.16 or higher
- SQLite3

## Installation

1. Clone the repository:

   ```
   git clone https://github.com/yourusername/db-discover.git
   cd db-discover
   ```

2. Install dependencies:
   ```
   go get github.com/gorilla/mux
   go get github.com/mattn/go-sqlite3
   ```

## Usage

1. Build the service:

   ```
   make build
   ```

2. Run the service:
   ```
   make run
   ```

The service will start on port 8080 by default. You can specify a different port using the `-port` flag:

```
./db-discover -port 9000
```

## API Endpoints

- `POST /databases`: Create a new database
  ```
  curl -X POST http://localhost:8080/databases -H "Content-Type: application/json" -d '{"name": "mydb", "description": "My new database", "workspace_id": 1}'
  ```

- `GET /databases`: List all databases
  ```
  curl "http://localhost:8080/databases?workspace_id=1"
  ```

- `DELETE /databases/{id}`: Destroy a database
  ```
  curl -X DELETE http://localhost:8080/databases/1
  ```

- `POST /tables`: Create a new table
  ```
  curl -X POST http://localhost:8080/tables -H "Content-Type: application/json" -d '{"name": "users", "database_id": 1}'
  ```

- `GET /tables`: List all tables
  ```
  curl "http://localhost:8080/tables?database_id=1"
  ```

- `DELETE /tables/{id}`: Destroy a table
  ```
  curl -X DELETE http://localhost:8080/tables/1
  ```

- `POST /fields`: Create a new field
  ```
  curl -X POST http://localhost:8080/fields -H "Content-Type: application/json" -d '{"name": "username", "type": "TEXT", "table_id": 1}'
  ```

- `GET /fields`: List all fields
  ```
  curl "http://localhost:8080/fields?table_id=1"
  ```

- `DELETE /fields/{id}`: Destroy a field
  ```
  curl -X DELETE http://localhost:8080/fields/1
  ```

- `PUT /fields/{id}`: Update a field
  ```
  curl -X PUT http://localhost:8080/fields/1 -H "Content-Type: application/json" -d '{"name": "email", "type": "TEXT"}'
  ```

- `POST /data`: Add or update data
  ```
  curl -X POST http://localhost:8080/data -H "Content-Type: application/json" -d '{"database_id": 1, "table_name": "users", "username": "john_doe", "email": "john@example.com"}'
  ```

- `GET /data`: Query data (SELECT only)
  ```
  curl "http://localhost:8080/data?database_id=1&query=SELECT%20*%20FROM%20users%20WHERE%20username%3D'john_doe'"
  ```

## Development

To run the service in development mode with automatic reloading:

```
make dev
```

## Testing

To run the tests:

```
make test
```

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.
