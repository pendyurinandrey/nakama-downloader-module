package main

import (
	"context"
	"database/sql"
	"github.com/heroiclabs/nakama-common/runtime"
)

func InitModule(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, initializer runtime.Initializer) error {
	err := createScheme(ctx, db)
	if err != nil {
		logger.Error("Failed to create DB scheme: %e", err)
		return err
	}
	err = initializer.RegisterRpc("FileDownloader", RpcFileDownloader)
	if err != nil {
		logger.Error("Failed to register the downloader rpc: %e", err)
		return err
	}

	return nil
}

func createScheme(ctx context.Context, db *sql.DB) error {
	_, err := db.ExecContext(ctx, createTableQuery)
	return err
}

const createTableQuery = `
	CREATE TABLE IF NOT EXISTS download_statistics (
	    file_name varchar(256) not null,
	    file_hash varchar(256) not null,
	    download_count bigint default 0,
	    primary key(file_name, file_hash)
	)`
