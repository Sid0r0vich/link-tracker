package domain

import (
	api "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/pkg/api/bot/rest"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/pkg/api/scrapper/rest"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/pkg/api/scrapper/rpc"
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
				Tags: *link.Tags,
				//Filters: *link.Filters,
			},
			URL: *link.Url,
		},
		ID: *link.Id,
	}
}

func LinkResponseSliceToLinkWithID(links []rest.LinkResponse) []LinkWithID {
	result := make([]LinkWithID, len(links))
	for i, link := range links {
		result[i] = *LinkResponseToLinkWithID(&link)
	}
	return result
}

func LinkResponseSliceToLinkWithIDSlice(links []rest.LinkResponse) []LinkWithID {
	newLinks := make([]LinkWithID, len(links))
	for idx, link := range links {
		newLinks[idx] = *LinkResponseToLinkWithID(&link)
	}

	return newLinks
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

func DbLinkToLinkWithID(link *DbLink) *LinkWithID {
	return &LinkWithID{
		Link: Link{
			LinkInfo: LinkInfo{
				Tags:      link.Tags,
				UpdatedAt: link.UpdatedAt,
			},
			URL: link.URL,
		},
		ID: link.ID,
	}
}

func LinkWithIDToRPCLinkResponse(link *LinkWithID) *rpc.LinkResponse {
	return &rpc.LinkResponse{
		Id:      link.ID,
		Url:     link.URL,
		Tags:    link.Tags,
		Filters: link.Filters,
	}
}

func LinkWithIDSliceToRPCLinkResponseSlice(links []LinkWithID) []*rpc.LinkResponse {
	resp := make([]*rpc.LinkResponse, len(links))
	for i, link := range links {
		resp[i] = LinkWithIDToRPCLinkResponse(&link)
	}
	return resp
}

func RPCLinkResponseToLink(link *rpc.AddLinkRequest) *Link {
	return &Link{
		LinkInfo: LinkInfo{
			Tags:    link.Tags,
			Filters: link.Filters,
		},
		URL: link.Url,
	}
}

func ApiEventToEvent(e *api.Event) *Event {
	return &Event{
		Type:        e.Type,
		Title:       e.Title,
		Description: e.Description,
		Username:    e.Username,
		CreatedAt:   e.CreatedAt,
	}
}

func ApiEventSliceToEventSlice(e []api.Event) []Event {
	events := make([]Event, len(e))
	for i, e := range e {
		events[i] = *ApiEventToEvent(&e)
	}
	return events
}

func EventSliceToApiEventSlice(e []Event) []api.Event {
	events := make([]api.Event, len(e))
	for i, event := range e {
		events[i] = api.Event{
			Type:        event.Type,
			Title:       event.Title,
			Description: event.Description,
			Username:    event.Username,
			CreatedAt:   event.CreatedAt,
		}
	}
	return events
}
