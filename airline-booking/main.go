package main

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const STR = "postgresql://neondb_owner:npg_yiqg4xJFNH2G@ep-dawn-sun-anjw31aq.c-6.us-east-1.aws.neon.tech/neondb?sslmode=require"

const (
	QUERYWITHOUTLOCK = `SELECT id, name, trip_id, user_id 
		FROM seat 
		WHERE trip_id = 1 AND user_id IS NULL 
		ORDER BY id 
		LIMIT 1 
	`
	QUERYWITHUPDATELOCK = `SELECT id, name, trip_id, user_id 
		FROM seat 
		WHERE trip_id = 1 AND user_id IS NULL 
		ORDER BY id 
		LIMIT 1 FOR UPDATE
	`
	QUERYWITHSKIPLOCKED = `SELECT id, name, trip_id, user_id 
		FROM seat 
		WHERE trip_id = 1 AND user_id IS NULL 
		ORDER BY id 
		LIMIT 1 FOR UPDATE SKIP LOCKED
	`
)

func BookSeat(db *pgxpool.Pool, ctx context.Context, userID int, query string) (int, error) {
	tx, err := db.Begin(ctx)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback(ctx)

	var id, tripId int
	var name string
	var userId *int

	// Step 1: SELECT Withiut lock
	err = tx.QueryRow(ctx, query).Scan(&id, &name, &tripId, &userId)

	if err != nil {
		if err == pgx.ErrNoRows {
			fmt.Printf("⚠️ User %d → No seat available\n", userID)
			return 0, nil
		}
		return 0, err
	}

	// Step 2: UPDATE same row
	_, err = tx.Exec(ctx,
		"UPDATE seat SET user_id=$1 WHERE id=$2",
		userID, id,
	)
	if err != nil {
		return 0, err
	}

	// Step 3: COMMIT
	if err := tx.Commit(ctx); err != nil {
		return 0, err
	}

	// Step 4: PRINT (after commit)
	fmt.Printf("✅ User %d booked Seat %d (%s)\n", userID, id, name)

	return id, nil
}

func main() {
	ctx := context.Background()
	db, err := pgxpool.New(ctx, STR)
	if err != nil {
		panic(err)
	}
	defer db.Close()

	user_ids := []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}
	start := time.Now()

	var wg sync.WaitGroup
	for _, value := range user_ids {
		wg.Add(1)
		go func(uid int) {
			defer wg.Done()

			_, err := BookSeat(db, ctx, uid, QUERYWITHOUTLOCK)
			if err != nil {
				fmt.Printf("❌ User %d error: %v\n", uid, err)
			}

		}(value)
	}
	wg.Wait()
	elapsed := time.Since(start)
	fmt.Printf("Time passed: %s\n", elapsed)
}
