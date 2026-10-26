package database

import (
	"database/sql"
	"fmt"
	"os"

	_ "github.com/microsoft/go-mssqldb/azuread"
)

func Open() (*sql.DB, error) {
	server, name := os.Getenv("SQL_SERVER_FQDN"), os.Getenv("SQL_DATABASE")
	if server == "" || name == "" {
		return nil, fmt.Errorf("set SQL_SERVER_FQDN and SQL_DATABASE from azd environment outputs")
	}
	return sql.Open("azuresql", fmt.Sprintf(
		"server=%s;database=%s;fedauth=ActiveDirectoryDefault;encrypt=true;TrustServerCertificate=false;connection timeout=30",
		server, name,
	))
}
