package domain

import (
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/api/scrapper/rest"
)

func LinkWithIDToLinkResponse(link *LinkWithID) *rest.LinkResponse {
	return &rest.LinkResponse{
		Id:      &link.ID,
		Url:     &link.URL,
		Tags:    &link.Tags,
		Filters: &link.Filters,
	}
}

func LinkResponseToLinkWithID(link *rest.LinkResponse) *LinkWithID {
	return &LinkWithID{
		Link: Link{
			LinkInfo: LinkInfo{
				Tags:    *link.Tags,
				Filters: *link.Filters,
			},
			URL: *link.Url,
		},
		ID: *link.Id,
	}
}

func LinkInfoWithIDToLinkWithID(link *LinkInfoWithID, url string) *LinkWithID {
	return &LinkWithID{
		Link: Link{
			LinkInfo: LinkInfo{
				Tags:    link.Tags,
				Filters: link.Filters,
			},
			URL: url,
		},
		ID: link.ID,
	}
}
