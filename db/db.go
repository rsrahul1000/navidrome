package db

import (
	"database/sql"
	"fmt"
	"os"

	_ "github.com/mattn/go-sqlite3"
	"github.com/navidrome/navidrome/conf"
	_ "github.com/navidrome/navidrome/db/migration"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/utils/singleton"
	"github.com/pressly/goose"
)

var (
	Driver = "sqlite3"
	Path   string
)

func Db() *sql.DB {
	instance := singleton.Get(&sql.DB{}, func() interface{} {
		Path = conf.Server.DbPath
		if Path == ":memory:" {
			Path = "file::memory:?cache=shared&_foreign_keys=on"
			conf.Server.DbPath = Path
		}
		log.Debug("Opening DataBase", "dbPath", Path, "driver", Driver)
		instance, err := sql.Open(Driver, Path)
		if err != nil {
			panic(err)
		}
		return instance
	})
	return instance.(*sql.DB)
}

func EnsureLatestVersion() {
	db := Db()

	// Disable foreign_keys to allow re-creating tables in migrations
	_, err := db.Exec("PRAGMA foreign_keys=off")
	defer func() {
		_, err := db.Exec("PRAGMA foreign_keys=on")
		if err != nil {
			log.Error("Error re-enabling foreign_keys", err)
		}
	}()
	if err != nil {
		log.Error("Error disabling foreign_keys", err)
	}

	gooseLogger := &logAdapter{silent: isSchemaEmpty(db)}
	goose.SetLogger(gooseLogger)

	err = goose.SetDialect(Driver)
	if err != nil {
		log.Error("Invalid DB driver", "driver", Driver, err)
		os.Exit(1)
	}
	err = goose.Run("up", db, "./")
	if err != nil {
		log.Error("Failed to apply new migrations", err)
		os.Exit(1)
	}
}

func isSchemaEmpty(db *sql.DB) bool { // nolint:interfacer
	rows, err := db.Query("SELECT name FROM sqlite_master WHERE type='table' AND name='goose_db_version';") // nolint:rowserrcheck
	if err != nil {
		log.Error("Database could not be opened!", err)
		os.Exit(1)
	}
	defer rows.Close()
	return !rows.Next()
}

type logAdapter struct {
	silent bool
}

func (l *logAdapter) Fatal(v ...interface{}) {
	log.Error(fmt.Sprint(v...))
	os.Exit(-1)
}

func (l *logAdapter) Fatalf(format string, v ...interface{}) {
	log.Error(fmt.Sprintf(format, v...))
	os.Exit(-1)
}

func (l *logAdapter) Print(v ...interface{}) {
	if !l.silent {
		log.Info(fmt.Sprint(v...))
	}
}

func (l *logAdapter) Println(v ...interface{}) {
	if !l.silent {
		log.Info(fmt.Sprintln(v...))
	}
}

func (l *logAdapter) Printf(format string, v ...interface{}) {
	if !l.silent {
		log.Info(fmt.Sprintf(format, v...))
	}
}
