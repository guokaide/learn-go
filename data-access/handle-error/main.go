package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	"github.com/go-sql-driver/mysql"
	"github.com/pkg/errors"
)

// database handle
var db *sql.DB

type Album struct {
	ID     int64
	Title  string
	Artist string
	Price  float32
}

// AlbumNotFound Handling Errors: Error Types (Custom Error Types)
type AlbumNotFound struct {
	ID int64
}

func (e *AlbumNotFound) Error() string {
	return fmt.Sprintf("Album with ID %d not found", e.ID)
}

// Handling Errors: Opaque errors
type albumNotFound interface {
	AlbumNotFound() (bool, int64)
}

func IsErrAlbumNotFound(err error) (bool, int64) {
	if e, ok := errors.Cause(err).(albumNotFound); ok {
		return e.AlbumNotFound()
	}
	return false, 0
}

type errAlbumNotFound struct {
	id int64
}

func (e *errAlbumNotFound) Error() string {
	return fmt.Sprintf("Album with Id %d not found", e.id)
}

func (e *errAlbumNotFound) AlbumNotFound() (bool, int64) {
	return true, e.id
}

func main() {
	// Capture connection properties.
	cfg := mysql.Config{
		User:                 os.Getenv("DBUSER"),
		Passwd:               os.Getenv("DBPASS"),
		Net:                  "tcp",
		Addr:                 "127.0.0.1:3306",
		DBName:               "recordings",
		AllowNativePasswords: true,
	}

	// Get a database handle.
	var err error
	db, err = sql.Open("mysql", cfg.FormatDSN())
	if err != nil {
		log.Fatal(err)
	}
	// Handling Errors: Opaque errors
	_, err = albumByID(10)
	if ok, _ := IsErrAlbumNotFound(err); ok {
		fmt.Printf("original error:\n%T %v\n", errors.Cause(err), errors.Cause(err))
		fmt.Printf("stack trace:\n%+v\n", err)
	}

	if err != nil {
		// ...
	}
	// Handling Errors: Error Types (Custom Error Types)
	//switch err := err.(type) {
	//case nil:
	//	fmt.Printf("Album found: %v\n", alb)
	//case *AlbumNotFound:
	//	fmt.Printf("original error:\n%T %v\n", errors.Cause(err), errors.Cause(err))
	//	fmt.Printf("stack trace:\n%+v\n", err)
	//default:
	//	fmt.Printf("original error:\n%T %v\n", errors.Cause(err), errors.Cause(err))
	//	fmt.Printf("stack trace:\n%+v\n", err)
	//}
}

// queries for the album with the specified ID
func albumByID(id int64) (Album, error) {
	var alb Album

	// It returns an sql.Row. To simplify the calling code (your code!),
	row := db.QueryRow("SELECT * FROM album WHERE id = ?", id)
	// QueryRow doesn’t return an error. Instead, it arranges to return any query error
	// (such as sql.ErrNoRows) from Rows.Scan later.
	if err := row.Scan(&alb.ID, &alb.Title, &alb.Artist, &alb.Price); err != nil {
		// The special error sql.ErrNoRows indicates that the query returned no rows.
		// Typically that error is worth replacing with more specific text, such as “no such album” here.
		if err == sql.ErrNoRows {
			return alb, errors.Wrapf(&errAlbumNotFound{id}, fmt.Sprintf("albumById %d: no such album", id))
			// Handling Errors: Error Types (Custom Error Types)
			//return alb, errors.Wrapf(&AlbumNotFound{id}, fmt.Sprintf("albumById %d: no such album", id))
			//return alb, errors.Wrapf(err, fmt.Sprintf("albumById %d: no such album", id))
		}
		return alb, errors.Wrapf(err, fmt.Sprintf("albumById %d: db query row error", id))
	}
	return alb, nil
}
