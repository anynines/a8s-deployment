package postgresql

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"

	"github.com/jackc/pgx/v4"
)

const (
	protocol = "postgres"

	hostname = "localhost"

	DbAdminUsernameKey = "username"
	DbAdminPasswordKey = "password"
	DatabaseKey        = "database"
)

func NewClient(credentials map[string]string, port string) Client {
	return Client{
		credentials: credentials,
		port:        port,
	}
}

// Client is responsible for opening up connections to the DSI in order to test the dataservice's
// behavior. Connections are opened for each method call to ensure a new and untainted connection to the
// database. This is important since leaving a connection open would break restore functionality.
type Client struct {
	// credentials is servicebinding data needed for connecting to a DSI
	credentials map[string]string
	// port is a dynmically determined port on localhost that has been portforwarded for opening connection to the DSI
	port string
}

func (c Client) Write(ctx context.Context, tableName, data string) error {
	dbConn, err := connectToDB(ctx, c.credentials, c.port)
	if err != nil {
		return err
	}
	defer func() { closeConnection(ctx, dbConn) }()

	if err := createTableIfNotExists(ctx, dbConn, tableName); err != nil {
		return err
	}

	if err := insertData(ctx, dbConn, tableName, data); err != nil {
		return fmt.Errorf("failed to insert data: %w", err)
	}
	return nil
}

func (c Client) Read(ctx context.Context, tableName string) (string, error) {
	dbConn, err := connectToDB(ctx, c.credentials, c.port)
	if err != nil {
		return "", err
	}
	defer func() { closeConnection(ctx, dbConn) }()

	query := fmt.Sprintf("SELECT * FROM %s;", tableName)
	rows, err := dbConn.Query(ctx, query)
	if err != nil {
		return "", fmt.Errorf(
			"failed to query database for rows with query %s: %w", query, err)
	}
	defer func() { rows.Close() }()

	var table []string
	for rows.Next() {
		var row string
		if err := rows.Scan(&row); err != nil {
			return "", fmt.Errorf("failed to scan row: %w", err)
		}
		table = append(table, row)
	}
	return strings.Join(table, "\n"), nil
}

func (c Client) UserExists(ctx context.Context, username string) (bool, error) {
	dbConn, err := connectToDB(ctx, c.credentials, c.port)
	if err != nil {
		return false, fmt.Errorf("failed to connect to the database: %w", err)
	}

	var success int
	err = dbConn.QueryRow(ctx, "SELECT 1 FROM pg_roles WHERE rolname=$1", username).
		Scan(&success)
	if err != nil {
		return false, fmt.Errorf("failed to query users from database: %w", err)
	}

	return success == 1, nil
}

func (c Client) CollectionExists(ctx context.Context, collection string) bool {
	dbConn, err := connectToDB(ctx, c.credentials, c.port)
	if err != nil {
		return false
	}

	var success int
	err = dbConn.QueryRow(ctx, "SELECT 1 FROM pg_database WHERE datname=$1", collection).
		Scan(&success)
	if err != nil {
		return false
	}

	return success == 1
}

func (c Client) Delete(ctx context.Context, entity, data string) error {
	return errors.New("not implemented")
}

func insertData(ctx context.Context, dbConn *pgx.Conn, tableName, input string) error {
	tx, err := dbConn.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin a transaction: %w", err)
	}
	defer func() { err = endTransaction(ctx, tx, err) }()

	query := fmt.Sprintf("INSERT INTO %s(input) VALUES ($1);", tableName)
	_, err = tx.Exec(ctx, query, input)
	if err != nil {
		return fmt.Errorf(
			"failed transaction for query %s with input %s: %v", query, input, err)
	}
	return nil
}

func createTableIfNotExists(ctx context.Context, dbConn *pgx.Conn, tableName string) error {
	createSqlTable := fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s(input text);", tableName)
	if _, err := dbConn.Exec(ctx, createSqlTable); err != nil {
		return fmt.Errorf("failed to create table with %s: %w", createSqlTable, err)
	}
	return nil
}

func connectToDB(ctx context.Context,
	credentials map[string]string,
	port string) (*pgx.Conn, error) {

	dbURL := dbURL(credentials, port)
	conn, err := pgx.Connect(ctx, dbURL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database with %s: %w", dbURL, err)
	}
	return conn, nil
}

func closeConnection(ctx context.Context, conn *pgx.Conn) {
	if err := conn.Close(ctx); err != nil {
		log.Println(err, "failed to close connection")
	}
}

func endTransaction(ctx context.Context, tx pgx.Tx, err error) error {
	if err != nil {
		if err1 := tx.Rollback(ctx); err1 != nil {
			log.Println(err1, "failed to rollback transaction")
		}
		return err
	}

	if err = tx.Commit(ctx); err != nil {
		err = fmt.Errorf("failed to commit transaction: %w", err)
	}
	return err
}

func dbURL(credentials map[string]string, port string) string {
	user := credentials[DbAdminUsernameKey]
	password := credentials[DbAdminPasswordKey]
	database := credentials[DatabaseKey]
	return protocol + "://" + user + ":" + password + "@" + hostname + ":" + port + "/" + database
}
