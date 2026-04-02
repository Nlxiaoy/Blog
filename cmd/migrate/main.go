package main

import (
	"flag"
	"fmt"
	"log"
	"server/config"

	_ "github.com/go-sql-driver/mysql"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/mysql"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

func buildDatabaseURL(cfg *config.Config) string {
	dsn := fmt.Sprintf("mysql://%s:%s@tcp(%s:%d)/%s?%s",
		cfg.Mysql.User,
		cfg.Mysql.Password,
		cfg.Mysql.Host,
		cfg.Mysql.Port,
		cfg.Mysql.DBName,
		cfg.Mysql.Config,
	)
	return dsn
}

func main() {
	dir := flag.String("dir", "migrations", "")
	action := flag.String("action", "up", "")
	steps := flag.Int("steps", 0, "")
	to := flag.Int("to", -1, "")
	flag.Parse()

	cfg, err := config.NewConfig()
	if err != nil {
		log.Fatalf("Migrate: config error: %s", err)
	}

	dbURL := buildDatabaseURL(cfg)
	m, err := migrate.New("file://"+*dir, dbURL)
	if err != nil {
		log.Fatalf("Migrate: init error: %s", err)
	}
	defer m.Close()

	switch *action {
	case "up":
		err = m.Up()
	case "down":
		if *steps > 0 {
			err = m.Steps(-*steps)
		} else {
			err = m.Down()
		}
	case "steps":
		if *steps == 0 {
			log.Fatalf("Migrate: steps must be non-zero")
		}
		err = m.Steps(*steps)
	case "force":
		if *to < 0 {
			log.Fatalf("Migrate: to must be set")
		}
		err = m.Force(*to)
	case "version":
		v, dirty, verr := m.Version()
		if verr != nil {
			log.Fatalf("Migrate: version error: %s", verr)
		}
		log.Printf("Migrate: version=%d dirty=%t", v, dirty)
		return
	case "drop":
		err = m.Drop()
	default:
		log.Fatalf("Migrate: unknown action: %s", *action)
	}

	if err != nil {
		if err == migrate.ErrNoChange {
			log.Printf("Migrate: no change")
			return
		}
		log.Fatalf("Migrate: error: %s", err)
	}

	log.Printf("Migrate: %s success", *action)
}
