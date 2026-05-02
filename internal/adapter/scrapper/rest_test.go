package scrapper_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/adapter/scrapper"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/domain"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/pkg/api/scrapper/rest"
	clientMocks "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/pkg/api/scrapper/rest/mocks"
	"go.uber.org/mock/gomock"
)

func TestAddChat_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	chatID := int64(123)
	mockClient := clientMocks.NewMockClientWithResponsesInterface(ctrl)

	mockClient.EXPECT().
		PostTgChatIdWithResponse(gomock.Any(), chatID).
		Return(&rest.PostTgChatIdResponse{
			HTTPResponse: &http.Response{
				StatusCode: http.StatusOK,
			},
		}, nil)

	adapter := &scrapper.ScrapperRestAdapter{Client: mockClient}

	err := adapter.AddChat(chatID)
	assert.NoError(t, err)
}

func TestAddChat_Failure(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	chatID := int64(123)
	mockClient := clientMocks.NewMockClientWithResponsesInterface(ctrl)

	mockClient.EXPECT().
		PostTgChatIdWithResponse(gomock.Any(), chatID).
		Return(&rest.PostTgChatIdResponse{
			HTTPResponse: &http.Response{
				StatusCode: http.StatusInternalServerError,
			},
		}, nil)

	adapter := &scrapper.ScrapperRestAdapter{Client: mockClient}

	err := adapter.AddChat(chatID)
	assert.Error(t, err)
}

func TestGetLinks_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	chatID := int64(123)
	id := int64(1)
	url := "http://example.com"
	tags := []string{}
	filters := []string{}
	mockClient := clientMocks.NewMockClientWithResponsesInterface(ctrl)

	links := []rest.LinkResponse{
		{
			Id:      &id,
			Url:     &url,
			Tags:    &tags,
			Filters: &filters,
		},
	}
	n := int32(len(links))

	mockClient.EXPECT().
		GetLinksWithResponse(gomock.Any(), gomock.Any()).
		Return(&rest.GetLinksResponse{
			HTTPResponse: &http.Response{
				StatusCode: http.StatusOK,
			},
			JSON200: &rest.ListLinksResponse{
				Links: &links,
				Size:  &n,
			},
		}, nil)

	adapter := &scrapper.ScrapperRestAdapter{Client: mockClient}
	result, err := adapter.GetLinks(chatID)
	assert.NoError(t, err)

	linksWithID := domain.LinkResponseSliceToLinkWithIDSlice(links)
	assert.Equal(t, linksWithID, result)
}
