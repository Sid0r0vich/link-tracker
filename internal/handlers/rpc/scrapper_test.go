package rpc_test

import (
	"context"
	"io"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/domain"
	uerrors "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/errors"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/handlers"
	server "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/handlers/rpc"
	linkmocks "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/repository/link/mocks"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/pkg/api/scrapper/rpc"
	"go.uber.org/mock/gomock"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func TestScrapperRPCServer_GetLinks_Success(t *testing.T) {
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
			},
		},
	}

	service := linkmocks.NewMockLinkRepository(ctrl)
	service.EXPECT().GetLinks(chatID).Return(expected, nil)

	server := server.NewScrapperRPCServer(service, testLogger())

	resp, err := server.GetLinks(context.Background(), &rpc.GetLinksRequest{TgChatId: chatID})
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.Len(t, resp.Links, len(expected))

	assert.Equal(t, int32(len(expected)), resp.Size)
	expectedFormatted := domain.LinkWithIDSliceToRPCLinkResponseSlice(expected)
	assert.Equal(t, expectedFormatted, resp.Links)
}

func TestScrapperRPCServer_GetLinks_Error(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	service := linkmocks.NewMockLinkRepository(ctrl)
	service.EXPECT().GetLinks(int64(77)).Return(nil, uerrors.ErrChatNotExists)

	server := server.NewScrapperRPCServer(service, testLogger())

	resp, err := server.GetLinks(context.Background(), &rpc.GetLinksRequest{TgChatId: 77})
	assert.Nil(t, resp)
	require.Error(t, err)

	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.NotFound, st.Code())
	assert.Equal(t, handlers.ChatNotExists, st.Message())
}

func TestScrapperRPCServer_RegisterDeleteChat(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	chatID := int64(123)
	service := linkmocks.NewMockLinkRepository(ctrl)
	service.EXPECT().AddChat(chatID).Return(nil)
	service.EXPECT().DeleteChat(chatID).Return(nil)

	server := server.NewScrapperRPCServer(service, testLogger())

	registerResp, registerErr := server.RegisterChat(context.Background(), &rpc.RegisterChatRequest{Id: chatID})
	deleteResp, deleteErr := server.DeleteChat(context.Background(), &rpc.DeleteChatRequest{Id: chatID})

	require.NoError(t, registerErr)
	require.NoError(t, deleteErr)
	require.NotNil(t, registerResp)
	require.NotNil(t, deleteResp)
}

func TestScrapperRPCServer_AddLink_Success(t *testing.T) {
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

	server := server.NewScrapperRPCServer(service, testLogger())

	resp, err := server.AddLink(context.Background(), &rpc.AddLinkRequest{
		TgChatId: chatID,
		Url:      url,
		Tags:     tags,
		Filters:  filters,
	})

	require.NoError(t, err)
	require.NotNil(t, resp)

	expected := &rpc.LinkResponse{
		Id:      linkID,
		Url:     url,
		Tags:    tags,
		Filters: filters,
	}
	assert.Equal(t, expected, resp)
}

func TestScrapperRPCServer_RemoveLink_Error(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	url := "https://example.com"
	chatID := int64(1)
	service := linkmocks.NewMockLinkRepository(ctrl)
	service.EXPECT().DeleteLink(chatID, url).Return(nil, uerrors.ErrLinkNotFound)

	server := server.NewScrapperRPCServer(service, testLogger())

	resp, err := server.RemoveLink(context.Background(), &rpc.RemoveLinkRequest{TgChatId: chatID, Link: url})
	assert.Nil(t, resp)
	require.Error(t, err)

	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.NotFound, st.Code())
	assert.Equal(t, handlers.LinkNotFound, st.Message())
}

func TestScrapperRPCServer_RemoveLink_Success(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

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
	chatID := int64(10)

	service := linkmocks.NewMockLinkRepository(ctrl)
	service.EXPECT().DeleteLink(chatID, expected.URL).Return(expected, nil)

	server := server.NewScrapperRPCServer(service, testLogger())

	resp, err := server.RemoveLink(context.Background(), &rpc.RemoveLinkRequest{TgChatId: chatID, Link: expected.URL})
	require.NoError(t, err)
	require.NotNil(t, resp)

	assert.Equal(t, domain.LinkWithIDToRPCLinkResponse(expected), resp)
}
