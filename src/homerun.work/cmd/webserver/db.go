package main

import (
	"context"
	"database/sql"
	"fmt"
	"runtime/debug"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/pkg/errors"
)

//connection properties
const (
	connectionMaxLifetime = 5 * time.Minute
	processTimeout        = 10 * time.Second
)

//ScanFn : function for scanning a row
type ScanFn func(desc ...interface{}) error

//OpenDB : open a db connection
func OpenDB(ctx context.Context, address string, user string, pwd Secret, database string) (*DB, error) {
	//create the data source
	dsn := fmt.Sprintf("%s:%s@tcp(%s)/%s?parseTime=true", user, string(pwd), address, database)

	//set-up the db connection
	ctx, logger := GetLogger(ctx)
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, errors.Wrap(err, "db open")
	}
	logger.Infow("db open", "address", address, "user", user, "db", database)

	err = db.Ping()
	if err != nil {
		return nil, errors.Wrap(err, "db ping")
	}
	logger.Debugw("db ping", "address", address, "user", user, "db", database)

	//set the pool size
	db.SetMaxIdleConns(GetDBMaxIdleConnections())
	db.SetMaxOpenConns(GetDBMaxOpenConnections())
	db.SetConnMaxLifetime(connectionMaxLifetime)

	//create the wrapper
	wrappedDB := &DB{
		db: db,
	}
	return wrappedDB, nil
}

//DB : wrapper for the database
type DB struct {
	db *sql.DB
	tx *sql.Tx
}

//Close : close the database connection
func (db *DB) Close() error {
	return db.db.Close()
}

//QueryRow : query a row from the database
func (db *DB) QueryRow(ctx context.Context, stmt string, args ...interface{}) (context.Context, *sql.Row, error) {
	ctx, logger := GetLogger(ctx)

	//query
	start := time.Now()
	defer func() {
		logger.Debugw("db query row", "sql", stmt, "args", args, "elapsedMS", FormatElapsedMS(start))
		AddCtxStatsAPI(ctx, ServerStatDB, "db query row", time.Since(start))
	}()

	var row *sql.Row
	if db.tx != nil {
		row = db.tx.QueryRowContext(ctx, stmt, args...)
	} else if db.db != nil {
		row = db.db.QueryRowContext(ctx, stmt, args...)
	} else {
		return ctx, nil, fmt.Errorf("no valid db for query row")
	}
	return ctx, row, nil
}

//Query : query the database
func (db *DB) Query(ctx context.Context, stmt string, args ...interface{}) (context.Context, *sql.Rows, error) {
	ctx, logger := GetLogger(ctx)

	//query
	start := time.Now()
	defer func() {
		logger.Debugw("db query", "sql", stmt, "args", args, "elapsedMS", FormatElapsedMS(start))
		AddCtxStatsAPI(ctx, ServerStatDB, "db query", time.Since(start))
	}()

	var err error
	var rows *sql.Rows
	if db.tx != nil {
		rows, err = db.tx.QueryContext(ctx, stmt, args...)
	} else if db.db != nil {
		rows, err = db.db.QueryContext(ctx, stmt, args...)
	} else {
		return ctx, nil, fmt.Errorf("no valid db for query")
	}
	return ctx, rows, err
}

//Exec : execute a statement against the database
func (db *DB) Exec(ctx context.Context, stmt string, args ...interface{}) (context.Context, sql.Result, error) {
	ctx, logger := GetLogger(ctx)

	//execute
	start := time.Now()
	defer func() {
		logger.Debugw("db exec", "sql", stmt, "args", args, "elapsedMS", FormatElapsedMS(start))
		AddCtxStatsAPI(ctx, ServerStatDB, "db exec", time.Since(start))
	}()

	var err error
	var result sql.Result
	if db.tx != nil {
		result, err = db.tx.ExecContext(ctx, stmt, args...)
	} else if db.db != nil {
		result, err = db.db.ExecContext(ctx, stmt, args...)
	} else {
		return ctx, nil, fmt.Errorf("no valid db for query")
	}
	return ctx, result, err
}

//BeginTx : begin a transaction
func (db *DB) BeginTx() error {
	tx, err := db.db.Begin()
	if err != nil {
		return errors.Wrap(err, "begin transaction")
	}
	db.tx = tx
	return nil
}

//RollbackTx : rollback a transaction
func (db *DB) RollbackTx() error {
	defer func() {
		db.tx = nil
	}()
	err := db.tx.Rollback()
	if err != nil {
		return errors.Wrap(err, "rollback transaction")
	}
	return nil
}

//CommitTx : commit a transaction
func (db *DB) CommitTx() error {
	defer func() {
		db.tx = nil
	}()
	err := db.tx.Commit()
	if err != nil {
		return errors.Wrap(err, "commit transaction")
	}
	return nil
}

//definition of the function to execute in a transaction
type txFunc func(ctx context.Context, db *DB) (context.Context, error)

//ProcessTx : process a transaction
func (db *DB) ProcessTx(ctx context.Context, op string, fn txFunc) (returnCtx context.Context, returnErr error) {
	ctx, logger := GetLogger(ctx)

	//process
	start := time.Now()
	defer func() {
		logger.Debugw("transaction", "operation", op, "elapsedMS", FormatElapsedMS(start))
		AddCtxStatsAPI(ctx, ServerStatDB, "transaction", time.Since(start))
	}()

	//check for an existing transaction
	if db.tx != nil {
		//execute the function
		ctx, err := fn(ctx, db)
		return ctx, err
	}

	//start a transaction
	err := db.BeginTx()
	if err != nil {
		return ctx, errors.Wrap(err, "begin transaction")
	}
	defer func() {
		//check for a panic as a safety net to ensure the transaction is rolled-back
		p := recover()
		if p != nil {
			logger.Warnw("transaction panic", "panic", p, "stack", string(debug.Stack()))
			err := db.RollbackTx()
			if err != nil {
				logger.Warnw("rollback transaction panic", "error", err)
			}
			panic(p)
		}

		//check for an error
		if returnErr != nil {
			err := db.RollbackTx()
			if err != nil {
				logger.Warnw("rollback transaction", "error", err)
			}
			return
		}

		//commit the transaction
		err := db.CommitTx()
		if err != nil {
			logger.Warnw("commit transaction", "error", err)
		}
	}()

	//execute the function
	ctx, err = fn(ctx, db)
	return ctx, err
}
