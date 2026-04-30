package main

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type database struct {
	dbConn *pgxpool.Pool
}

type KV interface {
	Set(ctx context.Context, key string, value any, ttl int64) (bool, error)
	Get(ctx context.Context, key string) (value any, err error)
	CleanUp(ctx context.Context) (bool, error)
}

func NewKv(dbConn *pgxpool.Pool) KV {
	return &database{dbConn: dbConn}
}

// pointer because it create new object for each db connection so pointer will optmize it
func (d *database) Set(ctx context.Context, key string, value any, ttl int64) (bool, error) {
	tx, err := d.dbConn.Begin(ctx)
	if err != nil {
		fmt.Printf("Unable To Start Transaction : %v", err)
		return false, err
	}
	defer tx.Rollback(ctx)

	if ttl <= 0 {
		return false, fmt.Errorf("ttl must be greater than 0")
	}

	ttl = time.Now().Unix() + ttl

	_, err = tx.Exec(ctx, `
			INSERT INTO keyvalue 
			values ($1,$2,$3) 
			ON CONFLICT (key) 
			DO UPDATE SET value=$2 , ttl =$3 ;
	`, key, value, ttl)

	if err != nil {
		fmt.Printf("Unable To Insert or Update : %v", err)
		return false, err
	}

	err = tx.Commit(ctx)
	if err != nil {
		fmt.Printf("Unable To Commit : %v", err)
		return false, err
	}
	return true, nil

}

// pointer because it create new object for each db connection so pointer will optmize it
func (d *database) Get(ctx context.Context, key string) (value any, err error) {
	tx, err := d.dbConn.Begin(ctx)
	if err != nil {
		fmt.Printf("Unable To Start Transaction : %v", err)
		return nil, err
	}
	defer tx.Rollback(ctx)

	err = tx.QueryRow(ctx, `
		SELECT value FROM keyvalue
		Where key =$1 and ttl > EXTRACT (EPOCH FROM now())
	`, key).Scan(&value)

	if err != nil {
		fmt.Printf("Unable To Insert or Update : %v", err)
		return nil, err
	}

	err = tx.Commit(ctx)
	if err != nil {
		fmt.Printf("Unable To Commit : %v", err)
		return nil, err
	}
	return value, nil

}

// this delete is background task which delete the key
func (d *database) CleanUp(ctx context.Context) (bool, error) {
	tx, err := d.dbConn.Begin(ctx)
	if err != nil {
		fmt.Printf("Unable To Start Transaction : %v", err)
		return false, err
	}
	defer tx.Rollback(ctx)

	rowAffected, err := tx.Exec(ctx, `DELETE FROM keyvalue where ttl <= EXTRACT (EPOCH FROM now())`)

	if err != nil {
		fmt.Printf("Unable To Insert or Update : %v", err)
		return false, err
	}

	fmt.Printf("Total Row Deleted :%v", rowAffected.RowsAffected())
	err = tx.Commit(ctx)
	if err != nil {
		fmt.Printf("Unable To Commit : %v", err)
		return false, err
	}
	return true, nil

}
