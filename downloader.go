package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/heroiclabs/nakama-common/runtime"
	"hash/crc32"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

const defaultTypeEnvVarName string = "default_type"
const defaultVersionEnvVarName string = "default_version"
const defaultFilePathEnvVarName string = "default_file_path"

// I decided not to add google.golang.org/grpc to the dependencies list just for 3 status codes.
const invalidArgumentCode = 3
const notFoundCode = 5
const internalErrorCode = 13

var config = make(map[string]string)

type DownloaderRequest struct {
	Type    string  `json:"type"`
	Version string  `json:"version"`
	Hash    *string `json:"hash,omitempty"`
}

type DownloaderResponse struct {
	Type    string  `json:"type"`
	Version string  `json:"version"`
	Hash    *string `json:"hash"`
	Content *string `json:"content"`
}

func RpcFileDownloader(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, payload string) (string, error) {
	req, err := unmarshalRequest(payload, logger)
	if err != nil {
		return "{}", err
	}

	err = validateRequest(req)
	if err != nil {
		return "{}", err
	}

	filePath, err := buildFilePath(req.Type, req.Version)
	if err != nil {
		return "{}", err
	}

	f, err := os.ReadFile(filePath)
	if err != nil {
		return "{}", runtime.NewError(fmt.Sprintf("File not found on path: %s", filePath), notFoundCode)
	}

	crc32Table := crc32.MakeTable(crc32.IEEE)
	fileCrc32 := strconv.FormatUint(uint64(crc32.Checksum(f, crc32Table)), 10)
	var resp DownloaderResponse
	if req.Hash != nil && fileCrc32 != *req.Hash {
		resp = DownloaderResponse{Type: req.Type, Version: req.Version, Hash: req.Hash, Content: nil}
	} else {
		content := string(f)
		resp = DownloaderResponse{Type: req.Type, Version: req.Version, Hash: &fileCrc32, Content: &content}
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
	if strings.TrimSpace(payload) == "" {
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

func buildFilePath(typeName string, version string) (string, error) {
	defaultPath, err := lookupEnvVarOrGetFromCache(defaultFilePathEnvVarName)
	if err != nil {
		return "", err
	}
	return filepath.Join(defaultPath, typeName, version) + ".json", nil
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
		    set download_count = download_statistics.download_count + 1
	`, filePath, resp.Hash, 1)
	if err != nil {
		logger.Error("Failed to save statistics to database: %e", err)
	}
}

func validateRequest(req DownloaderRequest) error {
	/*
		It's necessary to be prepared for the situation when, for example, `type` will contain following value
		`/../../core`. If such value will be passed to filepath.Join, then the RPC will provide the ability to
		get any file outside the desired folder. It'll be a serious vulnerability.

		Linux does not allow only 2 characters in folder names: / and /0. It looks like that
		checking for absence of `/` is enough in the current case.
	*/

	if strings.Contains(req.Type, "/") {
		return runtime.NewError("`type` field must not contain /", invalidArgumentCode)
	}

	if strings.Contains(req.Version, "/") {
		return runtime.NewError("`version` field must not contain /", invalidArgumentCode)
	}

	return nil
}
