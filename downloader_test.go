package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
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
	mockLogger := buildLoggerMock()
	mockNakamaModule := mocks.NewNakamaModuleMock(t)

	res, err := RpcFileDownloader(context.Background(), mockLogger, db, mockNakamaModule, "")
	assert.NoError(t, err)
	response := unmarshalResponse(res)
	assert.Equal(t, "core", response.Type)
	assert.Equal(t, "1.0.0", response.Version)
	assert.Equal(t, "2358080557", *response.Hash)
	assert.Equal(t, "{\"core\": \"1.0.0\"}", *response.Content)
}

func TestThatDownloaderWillReturnDataOfCustomTypeWith5_0_0Version(t *testing.T) {
	db, _ := createDbMock()
	mockLogger := buildLoggerMock()
	mockNakamaModule := mocks.NewNakamaModuleMock(t)
	payload := buildPayload("custom", "5.0.0", nil)

	res, err := RpcFileDownloader(context.Background(), mockLogger, db, mockNakamaModule, payload)
	assert.NoError(t, err)
	response := unmarshalResponse(res)
	assert.Equal(t, "custom", response.Type)
	assert.Equal(t, "5.0.0", response.Version)
	assert.Equal(t, "3181399843", *response.Hash)
	assert.Equal(t, "{\"custom\": \"5.0.0\"}", *response.Content)
}

func TestThatStatisticsWillBeStoredToDatabase(t *testing.T) {
	db, dbMock := createDbMock()
	mockLogger := buildLoggerMock()
	mockNakamaModule := mocks.NewNakamaModuleMock(t)
	payload := buildPayload("custom", "5.0.0", nil)
	expectedPath, err := buildFilePath("custom", "5.0.0")
	if err != nil {
		panic(err)
	}
	dbMock.
		ExpectExec("insert into download_statistics").
		WithArgs(expectedPath, "3181399843", 1).
		WillReturnResult(sqlmock.NewResult(1, 1))
	_, err = RpcFileDownloader(context.Background(), mockLogger, db, mockNakamaModule, payload)
	assert.NoError(t, err)
	err = dbMock.ExpectationsWereMet()
	assert.NoError(t, err)
}

func TestThatContentWillBeEmptyIfHashCodesDoNotMatch(t *testing.T) {
	db, _ := createDbMock()
	mockLogger := buildLoggerMock()
	mockNakamaModule := mocks.NewNakamaModuleMock(t)
	hash := "notcrc32"
	payload := buildPayload("custom", "5.0.0", &hash)

	res, err := RpcFileDownloader(context.Background(), mockLogger, db, mockNakamaModule, payload)
	assert.NoError(t, err)
	response := unmarshalResponse(res)
	assert.Equal(t, "custom", response.Type)
	assert.Equal(t, "5.0.0", response.Version)
	assert.Equal(t, "notcrc32", *response.Hash)
	assert.Nil(t, response.Content)
}

func TestThatErrorWillBeRaisedIfFileIsNotFound(t *testing.T) {
	db, _ := createDbMock()
	mockLogger := buildLoggerMock()
	mockNakamaModule := mocks.NewNakamaModuleMock(t)
	payload := buildPayload("non_existing_type", "5.0.0", nil)

	res, rpcErr := RpcFileDownloader(context.Background(), mockLogger, db, mockNakamaModule, payload)
	expectedFilePath, err := buildFilePath("non_existing_type", "5.0.0")
	if err != nil {
		panic(err)
	}
	assert.EqualError(t, rpcErr, fmt.Sprintf("File not found on path: %s", expectedFilePath))
	assert.Equal(t, "{}", res)
}

func TestThatErrorWillBeRaisedIfTypeContainsBackslash(t *testing.T) {
	db, _ := createDbMock()
	mockLogger := buildLoggerMock()
	mockNakamaModule := mocks.NewNakamaModuleMock(t)
	payload := buildPayload("../../core", "5.0.0", nil)

	res, rpcErr := RpcFileDownloader(context.Background(), mockLogger, db, mockNakamaModule, payload)
	assert.EqualError(t, rpcErr, "`type` field must not contain /")
	assert.Equal(t, "{}", res)
}

func TestThatErrorWillBeRaisedIfVersionContainsBackslash(t *testing.T) {
	db, _ := createDbMock()
	mockLogger := buildLoggerMock()
	mockNakamaModule := mocks.NewNakamaModuleMock(t)
	payload := buildPayload("core", "../5.0.0", nil)

	res, rpcErr := RpcFileDownloader(context.Background(), mockLogger, db, mockNakamaModule, payload)
	assert.EqualError(t, rpcErr, "`version` field must not contain /")
	assert.Equal(t, "{}", res)
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

func buildLoggerMock() *mocks.LoggerMock {
	mockLogger := mocks.LoggerMock{}
	mockLogger.On("Info", mock.Anything, mock.Anything).Return(nil)
	mockLogger.On("Error", mock.Anything, mock.Anything).Return(nil)
	return &mockLogger
}

func unmarshalResponse(res string) DownloaderResponse {
	response := DownloaderResponse{}
	err := json.Unmarshal([]byte(res), &response)
	if err != nil {
		panic(err)
	}
	return response
}

func buildPayload(typeName string, version string, hash *string) string {
	req := DownloaderRequest{Type: typeName, Version: version, Hash: hash}
	payload, err := json.Marshal(req)
	if err != nil {
		panic(err)
	}
	return string(payload[:])
}
