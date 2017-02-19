package caldav

import (
	"net/url"
	"time"

	"github.com/WF/caldav-go/caldav/entities"
	"github.com/WF/caldav-go/icalendar/components"
	"github.com/WF/caldav-go/webdav"
	props "github.com/WF/caldav-go/webdav/entities"
	"github.com/WF/go/calendar"
)

const (
	calendarType = "VEVENT"
	httpOK       = "HTTP/1.1 200 OK"
)

var (
	findCalendarsRequestBody = &props.Propfind{
		Props: []*props.Prop{
			&props.Prop{
				DisplayName:                   " ", // non-empty so that it's not omitted
				CalendarTimezone:              &components.TimeZone{},
				SupportedCalendarComponentSet: &props.SupportedCalendarComponentSet{},
			},
		},
	}
)

// CalendarEvents gets events from the user's calendars in the specified time
// window.
func (client *client) CalendarEvents(startUTC time.Time, endUTC time.Time) ([]calendar.Event, error) {
	query, err := entities.NewEventRangeQuery(startUTC, endUTC)
	if err != nil {
		return nil, err
	}

	calendars, err := client.findCalendars()
	if err != nil {
		return nil, err
	}

	calendarItems := []calendar.Event{}
	for _, calendar := range calendars {
		events, err := client.calendarClient.QueryEvents(calendar.path, query)
		if err != nil {
			return nil, err
		}

		for _, event := range events {
			calendarItems = append(calendarItems, newCalendarItem(event, calendar))
		}
	}

	return calendarItems, nil
}

func (client *client) findCalendars() ([]*calendarListEntry, error) {
	multistatus, err := client.calendarClient.WebDAV().Propfind(client.path, webdav.Depth1, findCalendarsRequestBody)
	if err != nil {
		return nil, err
	}

	calendars := make([]*calendarListEntry, 0, len(multistatus.Responses))
	for _, response := range multistatus.Responses {
		propertyStatus := response.PropStats[0]
		if propertyStatus.Status == httpOK && propertyStatus.Prop.SupportedCalendarComponentSet != nil {
			for _, component := range propertyStatus.Prop.SupportedCalendarComponentSet.Components {
				if component.Name == calendarType {
					path, err := url.QueryUnescape(response.Href)
					if err != nil {
						return nil, err
					}

					cal := &calendarListEntry{
						path:         path,
						emailAddress: client.emailAddress,
						displayName:  response.PropStats[0].Prop.DisplayName,
						timeZone:     extractTimeZoneID(response.PropStats[0].Prop.CalendarTimezone),
					}
					calendars = append(calendars, cal)
					break
				}
			}
		}
	}

	return calendars, nil
}

func extractTimeZoneID(timeZone *components.TimeZone) string {
	if timeZone == nil {
		return ""
	}
	return timeZone.Id
}

type calendarListEntry struct {
	path         string
	emailAddress string
	displayName  string
	timeZone     string
}
