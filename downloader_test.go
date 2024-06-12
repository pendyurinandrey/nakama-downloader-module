package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"os"
	mocks "pendyurinandrey.com/nakama-downloader-module/mocks/github.com/heroiclabs/nakama-common/runtime"
	"testing"
)
import "github.com/DATA-DOG/go-sqlmock"

func init() {
	setEnvVars()
}

func TestThatBlankPayloadWillBeParsedAsDefaultRequest(t *testing.T) {
	db, _ := createDbMock()
	mockLogger := buildLoggerMock(t)
	mockNakamaModule := mocks.NewNakamaModuleMock(t)

	res, err := RpcFileDownloader(context.Background(), mockLogger, db, mockNakamaModule, "")
	assert.NoError(t, err)
	response := unmarshalResponse(res)
	assert.Equal(t, "core", response.Type)
	assert.Equal(t, "1.0.0", response.Version)
	assert.Equal(t, "2358080557", response.Hash)
	assert.Equal(t, "{\"core\": \"1.0.0\"}", response.Content)
}

func TestThatDownloaderWillReturnDataOfCustomTypeWith5_0_0Version(t *testing.T) {
	db, _ := createDbMock()
	mockLogger := buildLoggerMock(t)
	mockNakamaModule := mocks.NewNakamaModuleMock(t)
	payload := buildPayload("custom", "5.0.0", "")

	res, err := RpcFileDownloader(context.Background(), mockLogger, db, mockNakamaModule, payload)
	assert.NoError(t, err)
	response := unmarshalResponse(res)
	assert.Equal(t, "custom", response.Type)
	assert.Equal(t, "5.0.0", response.Version)
	assert.Equal(t, "3181399843", response.Hash)
	assert.Equal(t, "{\"custom\": \"5.0.0\"}", response.Content)

}

func createDbMock() (*sql.DB, sqlmock.Sqlmock) {
	db, mock, err := sqlmock.New()
	if err != nil {
		panic(err)
	}
	return db, mock
}

func setEnvVars() {
	os.Setenv(defaultTypeEnvVarName, "core")
	os.Setenv(defaultVersionEnvVarName, "1.0.0")
	os.Setenv(defaultFilePathEnvVarName, "./test_data")
}

func buildLoggerMock(t *testing.T) *mocks.LoggerMock {
	mockLogger := mocks.NewLoggerMock(t)
	mockLogger.On("Info", mock.Anything, mock.Anything).Return(nil)
	mockLogger.On("Error", mock.Anything, mock.Anything).Return(nil)
	return mockLogger
}

func unmarshalResponse(res string) DownloaderResponse {
	response := DownloaderResponse{}
	err := json.Unmarshal([]byte(res), &response)
	if err != nil {
		panic(err)
	}
	return response
}

func buildPayload(typeName string, version string, hash string) string {
	req := DownloaderRequest{Type: typeName, Version: version, Hash: hash}
	payload, err := json.Marshal(req)
	if err != nil {
		panic(err)
	}
	return string(payload[:])
}
