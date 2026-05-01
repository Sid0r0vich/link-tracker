package rest_test

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/cache"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/domain"
	uerrors "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/errors"
	basehandlers "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/handlers"
	server "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/handlers/rest"
	linkmocks "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/repository/link/mocks"
	api "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/pkg/api/scrapper/rest"
	"go.uber.org/mock/gomock"
)

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func decodeJSON[T any](t *testing.T, rr *httptest.ResponseRecorder, dst *T) {
	t.Helper()
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), dst))
}

func TestScrapperRestServer_GetLinks_Success(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	chatID := int64(101)
	expected := []domain.LinkWithID{
		{
			ID: 1,
			Link: domain.Link{
				URL: "https://example.com/1",
				LinkInfo: domain.LinkInfo{
					Tags: []string{"go"},
				},
			},
		},
		{
			ID: 2,
			Link: domain.Link{
				URL: "https://example.com/2",
				LinkInfo: domain.LinkInfo{
					Tags: []string{},
				},
			},
		},
	}

	service := linkmocks.NewMockLinkRepository(ctrl)
	service.EXPECT().GetLinks(chatID).Return(expected, nil)

	server := server.NewScrapperRestServer(service, testLogger(), cache.NewNoCache())
	request := httptest.NewRequest(http.MethodGet, "/links", nil)
	response := httptest.NewRecorder()

	server.GetLinks(response, request, api.GetLinksParams{TgChatId: chatID})

	require.Equal(t, http.StatusOK, response.Code)
	assert.Equal(t, "application/json", response.Header().Get("Content-Type"))

	var got api.ListLinksResponse
	decodeJSON(t, response, &got)

	require.NotNil(t, got.Links)
	require.NotNil(t, got.Size)
	assert.Equal(t, int32(len(expected)), *got.Size)
	gotLinks := domain.LinkResponseSliceToLinkWithID(*got.Links)
	assert.Equal(t, expected, gotLinks)
}

func TestScrapperRestServer_GetLinks_Error(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	service := linkmocks.NewMockLinkRepository(ctrl)
	service.EXPECT().GetLinks(int64(77)).Return(nil, uerrors.ErrChatNotExists)

	server := server.NewScrapperRestServer(service, testLogger(), cache.NewNoCache())
	request := httptest.NewRequest(http.MethodGet, "/links", nil)
	response := httptest.NewRecorder()

	server.GetLinks(response, request, api.GetLinksParams{TgChatId: 77})

	require.Equal(t, http.StatusNotFound, response.Code)

	var got api.ApiErrorResponse
	decodeJSON(t, response, &got)
	assert.Equal(t, basehandlers.ChatNotExists, *got.Description)
	assert.Equal(t, "", *got.Code)
	assert.Equal(t, uerrors.ErrChatNotExists.Error(), *got.ExceptionMessage)
}

func TestScrapperRestServer_ChatHandlers(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	chatID := int64(123)
	service := linkmocks.NewMockLinkRepository(ctrl)
	service.EXPECT().AddChat(chatID).Return(nil)
	service.EXPECT().DeleteChat(chatID).Return(nil)

	server := server.NewScrapperRestServer(service, testLogger(), cache.NewNoCache())

	postRequest := httptest.NewRequest(http.MethodPost, "/tg-chat", nil)
	postResponse := httptest.NewRecorder()
	server.PostTgChatId(postResponse, postRequest, chatID)
	require.Equal(t, http.StatusOK, postResponse.Code)

	deleteRequest := httptest.NewRequest(http.MethodDelete, "/tg-chat", nil)
	deleteResponse := httptest.NewRecorder()
	server.DeleteTgChatId(deleteResponse, deleteRequest, chatID)
	require.Equal(t, http.StatusOK, deleteResponse.Code)
}

func TestScrapperRestServer_AddLink_Success(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	chatID := int64(55)
	linkID := int64(1234)
	url := "https://stackoverflow.com/questions/1"
	tags := []string{"tag1", "tag2"}
	filters := []string{"f1"}

	service := linkmocks.NewMockLinkRepository(ctrl)
	service.EXPECT().AddLink(chatID, domain.Link{
		LinkInfo: domain.LinkInfo{
			Tags:    tags,
			Filters: filters,
		},
		URL: url,
	}).Return(linkID, nil)

	server := server.NewScrapperRestServer(service, testLogger(), cache.NewNoCache())
	requestBody, err := json.Marshal(domain.Link{
		LinkInfo: domain.LinkInfo{
			Tags:    tags,
			Filters: filters,
		},
		URL: url,
	})
	require.NoError(t, err)

	request := httptest.NewRequest(http.MethodPost, "/links", bytes.NewReader(requestBody))
	response := httptest.NewRecorder()

	server.PostLinks(response, request, api.PostLinksParams{TgChatId: chatID})

	require.Equal(t, http.StatusOK, response.Code)
	assert.Equal(t, "application/json", response.Header().Get("Content-Type"))

	var got api.LinkResponse
	decodeJSON(t, response, &got)

	expected := api.LinkResponse{
		Id:      &linkID,
		Url:     &url,
		Tags:    &tags,
		Filters: &filters,
	}
	assert.Equal(t, expected, got)
}

func TestScrapperRestServer_RemoveLink_Error(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	chatID := int64(1)
	url := "https://example.com"
	service := linkmocks.NewMockLinkRepository(ctrl)
	service.EXPECT().DeleteLink(chatID, url).Return(nil, uerrors.ErrLinkNotFound)

	server := server.NewScrapperRestServer(service, testLogger(), cache.NewNoCache())
	body, err := json.Marshal(api.RemoveLinkRequest{Link: &url})
	require.NoError(t, err)

	request := httptest.NewRequest(http.MethodDelete, "/links", bytes.NewReader(body))
	response := httptest.NewRecorder()

	server.DeleteLinks(response, request, api.DeleteLinksParams{TgChatId: chatID})

	require.Equal(t, http.StatusNotFound, response.Code)

	var got api.ApiErrorResponse
	decodeJSON(t, response, &got)
	assert.Equal(t, basehandlers.LinkNotFound, *got.Description)
	assert.Equal(t, "", *got.Code)
	assert.Equal(t, uerrors.ErrLinkNotFound.Error(), *got.ExceptionMessage)
}

func TestScrapperRestServer_RemoveLink_Success(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	chatID := int64(10)
	expected := &domain.LinkWithID{
		ID: 88,
		Link: domain.Link{
			URL: "https://example.com",
			LinkInfo: domain.LinkInfo{
				Tags:    []string{"tag"},
				Filters: []string{"filter"},
			},
		},
	}

	service := linkmocks.NewMockLinkRepository(ctrl)
	service.EXPECT().DeleteLink(chatID, expected.URL).Return(expected, nil)

	server := server.NewScrapperRestServer(service, testLogger(), cache.NewNoCache())
	body, err := json.Marshal(api.RemoveLinkRequest{Link: &expected.URL})
	require.NoError(t, err)

	request := httptest.NewRequest(http.MethodDelete, "/links", bytes.NewReader(body))
	response := httptest.NewRecorder()

	server.DeleteLinks(response, request, api.DeleteLinksParams{TgChatId: chatID})

	require.Equal(t, http.StatusOK, response.Code)
	assert.Equal(t, "application/json", response.Header().Get("Content-Type"))

	var got api.LinkResponse
	decodeJSON(t, response, &got)
	assert.Equal(t, domain.LinkWithIDToLinkResponse(expected), &got)
}
