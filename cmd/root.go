package cmd

import (
	"context"
	"log"
	"reflections/internal"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/spf13/cobra"
)

var (
	dbPool *pgxpool.Pool
	store  *internal.Store
)

var rootCmd = &cobra.Command{
	Use:   "reflections",
	Short: "A tool to help you reflect",
	Long:  `A tool to help you reflect`, // TODO: - add more
}

func Execute() {
	cobra.CheckErr(rootCmd.Execute())
}

func init() {
	cobra.OnInitialize(initDB)
}

func initDB() {
	ctx := context.Background()

	databaseURL := "postgres://steve@localhost:5432/steve"

	var err error
	dbPool, err = pgxpool.New(ctx, databaseURL)
	if err != nil {
		log.Fatal(err)
	}

	store = internal.NewStore(dbPool)

}
