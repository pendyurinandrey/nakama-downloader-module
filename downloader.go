package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/heroiclabs/nakama-common/runtime"
	"hash/crc32"
	"os"
	"strconv"
)

const defaultType string = "core"
const defaultVersion string = "1.0.0"

const notFoundErrorCode = 5

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
	f, err := os.ReadFile(fmt.Sprintf("/%s/%s.json", req.Type, req.Version))
	if err != nil {
		return "{}", runtime.NewError("File not found", notFoundErrorCode)
	}

	crc32Table := crc32.MakeTable(crc32.IEEE)
	fileCrc32 := strconv.FormatUint(uint64(crc32.Checksum(f, crc32Table)), 10)
	var resp DownloaderResponse
	if req.Hash != "" && fileCrc32 != req.Hash {
		resp = DownloaderResponse{Type: req.Type, Version: req.Version, Hash: req.Hash, Content: nil}
	} else {
		resp = DownloaderResponse{Type: req.Type, Version: req.Version, Hash: req.Hash, Content: f}
	}
	respStr, err := json.Marshal(resp)
	// TODO: Better error handling
	return string(respStr[:]), err
}
