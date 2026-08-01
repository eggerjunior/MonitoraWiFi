// Comando de migração de banco (Seção 3: "migrações versionadas"). Uso:
//
//	migrate -database "$DATABASE_URL" -path "$MIGRATIONS_DIR" up
//	migrate -database "$DATABASE_URL" -path "$MIGRATIONS_DIR" down 1
package main

import (
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"strconv"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

func main() {
	databaseURL := flag.String("database", os.Getenv("DATABASE_URL"), "URL de conexão com o PostgreSQL")
	path := flag.String("path", "infrastructure/database/migrations", "diretório com os arquivos de migração")
	flag.Parse()

	args := flag.Args()
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "uso: migrate -database <url> -path <dir> <up|down [n]|version>")
		os.Exit(2)
	}

	if *databaseURL == "" {
		fmt.Fprintln(os.Stderr, "DATABASE_URL não definido")
		os.Exit(2)
	}

	m, err := migrate.New("file://"+*path, *databaseURL)
	if err != nil {
		slog.Error("erro ao inicializar migrate", slog.Any("error", err))
		os.Exit(1)
	}

	switch args[0] {
	case "up":
		err = m.Up()
	case "down":
		steps := 1
		if len(args) > 1 {
			steps, _ = strconv.Atoi(args[1])
		}
		err = m.Steps(-steps)
	case "version":
		v, dirty, verr := m.Version()
		if verr != nil {
			err = verr
		} else {
			fmt.Printf("versão=%d dirty=%v\n", v, dirty)
		}
	default:
		fmt.Fprintf(os.Stderr, "comando desconhecido: %s\n", args[0])
		os.Exit(2)
	}

	if err != nil && !errors.Is(err, migrate.ErrNoChange) {
		slog.Error("erro ao migrar", slog.Any("error", err))
		os.Exit(1)
	}

	fmt.Println("ok")
}
