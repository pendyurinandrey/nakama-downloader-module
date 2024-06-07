package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"github.com/heroiclabs/nakama-common/runtime"
)

const defaultType string = "core"
const defaultVersion string = "1.0.0"

type DownloaderRequest struct {
	Type    string `json:"type"`
	Version string `json:"version"`
	Hash    string `json:"hash,omitempty"`
}

type DownloaderResponse struct {
	Type    string `json:"type"`
	Version string `json:"version"`
	Hash    string `json:"hash"`
	Content []byte `json:"content"`
}

func RpcFileDownloader(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, payload string) (string, error) {
	logger.Info("Payload: %s", payload)
	req := DownloaderRequest{Type: defaultType, Version: defaultVersion}
	err := json.Unmarshal([]byte(payload), &req)
	if err != nil {
		/*
			Since it's more likely a client's error, it's better to log it at a lower logging level than 'error'
			to avoid excessive logs. In a production environment, it will be possible to decrease the logging level
			to show this type of error when necessary.
		*/
		logger.Info("Unable to deserialize request %e", err)
		return "", err
	}
	return "", nil
}
