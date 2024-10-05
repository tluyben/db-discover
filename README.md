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

The service will start on port 8080.

## API Endpoints

- `POST /databases`: Create a new database
- `GET /databases`: List all databases
- `DELETE /databases/{id}`: Destroy a database
- `POST /tables`: Create a new table
- `GET /tables`: List all tables
- `DELETE /tables/{id}`: Destroy a table
- `POST /fields`: Create a new field
- `GET /fields`: List all fields
- `DELETE /fields/{id}`: Destroy a field
- `PUT /fields/{id}`: Update a field
- `POST /data`: Add or update data
- `GET /data`: Query data (SELECT only)

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
