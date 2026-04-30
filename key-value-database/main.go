package main

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

const STR = "postgresql://neondb_owner:npg_yiqg4xJFNH2G@ep-dawn-sun-anjw31aq.c-6.us-east-1.aws.neon.tech/neondb?sslmode=require"

const CREATETABLE = `
 CREATE TABLE IF NOT EXISTS keyvalue  (
    key VARCHAR(16) PRIMARY KEY,
	value JSONB NOT NULL,
	ttl BIGINT NOT NULL
);

  CREATE INDEX IF NOT EXISTS idx_ttl on keyvalue(ttl);
`

func init() {
	ctx := context.Background()

	conn, err := pgxpool.New(ctx, STR)
	if err != nil {
		fmt.Printf("Unable To Connect TO Database %v", err)
		panic(err)
	}
	defer conn.Close()

	_, err = conn.Exec(ctx, CREATETABLE)
	if err != nil {
		fmt.Printf("Unable To Create Table %v", err)
		panic(err)
	}
}

func main() {
	ctx := context.Background()

	conn, err := pgxpool.New(ctx, STR)
	if err != nil {
		fmt.Printf("Unable To Connect TO Database %v", err)
		panic(err)
	}
	defer conn.Close()

}
