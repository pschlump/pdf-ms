package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/pschlump/godebug"
)

// LogFile sets the output log file to an open file.  This will turn on logging of SQL statments.
func LogFile(f *os.File) {
	logFilePtr = f
}

// LogQueries is called with all statments to log them to a file.
func logQueries(stmt string, err error, data []interface{}, elapsed time.Duration) {
	if logFilePtr != nil {
		if err != nil {
			fmt.Fprintf(logFilePtr, "Error: %s stmt: %s data: %v elapsed: %s called from: %s\n", err, stmt, data, elapsed, godebug.LF(3))
		} else {
			fmt.Fprintf(logFilePtr, "stmt: %s data: %v elapsed: %s\n", stmt, data, elapsed)
		}
	}
}

// SQLQueryRow queries a single row and returns that data.
func SQLQueryRow(stmt string, data ...interface{}) (aRow *sql.Row) {
	start := time.Now()
	aRow = DB.QueryRow(stmt, data...)
	elapsed := time.Since(start)
	logQueries(stmt, nil, data, elapsed)
	return
}

// SQLExec will run command that returns a resultSet (think insert).
func SQLExec(stmt string, data ...interface{}) (resultSet sql.Result, err error) {
	start := time.Now()
	resultSet, err = DB.Exec(stmt, data...)
	elapsed := time.Since(start)
	logQueries(stmt, err, data, elapsed)
	return
}

// SQLQuery runs stmt and returns rows.
func SQLQuery(stmt string, data ...interface{}) (resultSet *sql.Rows, err error) {
	start := time.Now()
	resultSet, err = DB.Query(stmt, data...)
	elapsed := time.Since(start)
	logQueries(stmt, err, data, elapsed)
	return
}

// SQLUpdate can run insert/update statements that do not return data.
func SQLUpdate(stmt string, data ...interface{}) (err error) {
	start := time.Now()
	var resultSet *sql.Rows
	resultSet, err = DB.Query(stmt, data...)
	elapsed := time.Since(start)
	logQueries(stmt, err, data, elapsed)
	if err == nil && resultSet != nil {
		resultSet.Close()
	}
	return
}

// ConnectToPG Connects to the postgresDB.  This can be called multiple times.
func ConnectToPG() {
	// Check if DB has been set yet
	if DB == nil {
		psqlInfo := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
			gCfg.DBHost, gCfg.DBPort, gCfg.DBUser, gCfg.DBPassword, gCfg.DBName, gCfg.DBSSLMode)
		// fmt.Printf("Connect to PG with: ->%s<-\n", psqlInfo)
		var err error
		DB, err = sql.Open("postgres", psqlInfo)
		if err != nil {
			log.Fatal(err)
		}
		if err = DB.Ping(); err != nil {
			log.Fatal(err)
		}
	}
}

// Ping database to verify that we are actually connected.
func Ping() string {
	DB.Ping()
	return "Connected!"
}
