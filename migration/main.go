package main

import (
	"embed_tern/migrations"
	"log"
	"os"

	"github.com/joho/godotenv"
)

func InitConfig() {
	err := godotenv.Load(".env")
	if err != nil {
		log.Fatal("Error loading .env file")
	}
}

func main() {
	InitConfig()
	// create a migrator, connecting to the postgresql
	// database defined by the environment variable
	migrator, err := migrations.NewMigrator(os.Getenv("POSTGRES_STRING"))
	if err != nil {
		panic(err)
	}

	// get the current migration status
	now, exp, info, err := migrator.Info()
	if err != nil {
		panic(err)
	}
	if now < exp {
		// migration is required, dump out the current state
		// and perform the migration
		println("migration needed, current state:")
		println(info)

		err = migrator.Migrate()
		if err != nil {
			panic(err)
		}
		println("migration successful!")
	} else {
		println("no database migration needed")
	}

}
