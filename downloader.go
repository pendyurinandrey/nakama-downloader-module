package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"github.com/heroiclabs/nakama-common/runtime"
	"hash/crc32"
	"os"
	"path/filepath"
	"strconv"
)

const defaultTypeEnvVarName string = "default_type"
const defaultVersionEnvVarName string = "default_version"
const defaultFilePathEnvVarName string = "default_file_path"

// I decided not to add google.golang.org/grpc to the dependencies list just for 2 status codes.
const notFoundCode = 5
const internalErrorCode = 13

var config = make(map[string]string)

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
	req, err := unmarshalRequest(payload, logger)
	if err != nil {
		return "{}", err
	}

	filePath, err := buildFilePath(req)
	if err != nil {
		return "{}", err
	}

	f, err := os.ReadFile(filePath)
	if err != nil {
		return "{}", runtime.NewError("File not found", notFoundCode)
	}

	crc32Table := crc32.MakeTable(crc32.IEEE)
	fileCrc32 := strconv.FormatUint(uint64(crc32.Checksum(f, crc32Table)), 10)
	var resp DownloaderResponse
	if req.Hash != "" && fileCrc32 != req.Hash {
		resp = DownloaderResponse{Type: req.Type, Version: req.Version, Hash: req.Hash, Content: nil}
	} else {
		resp = DownloaderResponse{Type: req.Type, Version: req.Version, Hash: req.Hash, Content: f}
	}
	writeStatistics(resp, filePath, db, logger)
	respStr, err := json.Marshal(resp)
	if err != nil {
		return "{}", err
	}
	return string(respStr[:]), nil
}

func unmarshalRequest(payload string, logger runtime.Logger) (DownloaderRequest, error) {
	req, err := buildDefaultRequest()
	if err != nil {
		return req, nil
	}
	err = json.Unmarshal([]byte(payload), &req)
	if err != nil {
		/*
			Since it is more likely a client's error, it's better to log it at a lower logging level than 'error'
			to avoid excessive logs. In a production environment, the logging level can be adjusted
			to show this type of error when necessary.
		*/
		logger.Info("Unable to deserialize request %e", err)
		/*
			It might not be a good idea to return the request object with default values in this case.
			Using pointers here is a bit questionable because the object will be copied to the heap
			by the end of the method and then garbage collected when the pointer becomes unusable.
			Since the request object is small and created frequently, it is better to leave it on the stack.
		*/
		return req, err
	}
	return req, nil
}

func buildDefaultRequest() (DownloaderRequest, error) {
	defaultType, err := lookupEnvVarOrGetFromCache(defaultTypeEnvVarName)
	if err != nil {
		return DownloaderRequest{}, err
	}
	defaultVersion, err := lookupEnvVarOrGetFromCache(defaultVersionEnvVarName)
	if err != nil {
		return DownloaderRequest{}, err
	}

	return DownloaderRequest{Type: defaultType, Version: defaultVersion}, nil
}

func lookupEnvVarOrGetFromCache(key string) (string, error) {
	value, ok := config[key]
	if !ok {
		value, exists := os.LookupEnv(key)
		if !exists {
			return "", runtime.NewError("Wrong service configuration", internalErrorCode)
		}
		config[key] = value
		return value, nil
	}

	return value, nil
}

func buildFilePath(req DownloaderRequest) (string, error) {
	defaultPath, err := lookupEnvVarOrGetFromCache(defaultFilePathEnvVarName)
	if err != nil {
		return "", err
	}
	return filepath.Join(defaultPath, req.Type, req.Version), nil
}

func writeStatistics(resp DownloaderResponse, filePath string, db *sql.DB, logger runtime.Logger) {
	if resp.Content == nil {
		// Right now the method only stores statistics for existing files with matched hash.
		return
	}
	_, err := db.Exec(`
		insert into download_statistics(file_name, file_hash, download_count)
		values($1, $2, $3)
		on conflict(file_name, file_hash) do update
		    set download_count = download_count + 1
	`, filePath, resp.Hash, 1)
	if err != nil {
		logger.Error("Failed to save statistics to database: %e", err)
	}
}
