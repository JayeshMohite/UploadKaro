package postgresql

import (
	"database/sql"
	"log"
	"sync"

	_ "github.com/lib/pq"
)

var (
	PostgresConnection *sql.DB
	once               sync.Once
)

func PostgresDbConnect() {

	// Build the DSN (Data Source Name) string
	connStr := "a"

	// Initialize the database connection once using sync.Once
	once.Do(func() {
		var err error
		PostgresConnection, err = sql.Open("postgres", connStr)
		if err != nil {
			log.Fatal(err)
		}

		// Ensure the database is reachable
		err = PostgresConnection.Ping()
		if err != nil {
			log.Fatal(err)
		}
		log.Println("PostgreSQL connection established successfully")
	})
}
