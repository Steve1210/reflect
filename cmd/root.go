package cmd

import (
	"context"
	"log"
	"os"
	"reflections/internal"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/spf13/cobra"
)

type storeInterface interface {
	FetchAllMetadataWithFilters(ctx context.Context, filters internal.FilterOptions) ([]internal.ReflectionHeader, error)
	FetchReflectionByID(ctx context.Context, id int64) (internal.Reflection, error)
	InsertReflection(ctx context.Context, r internal.Reflection) (int64, error)
	UpdateReflection(ctx context.Context, id int64, r internal.Reflection) error
	DeleteReflection(ctx context.Context, id int64) error
}

var (
	dbPool *pgxpool.Pool
	store  storeInterface
	apiURL string
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
	rootCmd.PersistentFlags().StringVar(&apiURL, "api-url", "", "Base URL of the reflections API (e.g. http://192.168.0.8:30080)")
}

func initDB() {
	if apiURL == "" {
		apiURL = os.Getenv("REFLECT_API_URL")
	}
	if apiURL != "" {
		store = newHTTPStore(apiURL)
		return
	}

	ctx := context.Background()

	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		databaseURL = "postgres://steve@localhost:5432/steve"
	}

	var err error
	dbPool, err = pgxpool.New(ctx, databaseURL)
	if err != nil {
		log.Fatal(err)
	}

	store = internal.NewStore(dbPool)
}
