package caldav

import (
	"net/mail"
	"reflect"
	"time"

	"github.com/WF/caldav-go/icalendar/components"
	"github.com/WF/caldav-go/icalendar/properties"
	"github.com/WF/caldav-go/icalendar/values"
	"github.com/WF/go/calendar"
	"github.com/WF/go/convert"
	"github.com/WF/go/enums/importance"
	"github.com/WF/go/enums/rsvp"
	"github.com/WF/go/enums/sensitivity"
	"github.com/Cepreu/Archive/log"
)

func newCalendarItem(event *components.Event, parentCalendar *calendarListEntry) *calendarItem {
	attendees := resolveAttendees(event.Attendees)
	return &calendarItem{
		Event:        event,
		calendar:     parentCalendar,
		responseType: findResponseType(parentCalendar.emailAddress, attendees),
		organizer:    resolveOrganizer(event.Organizer),
		attendees:    attendees,
		sensitivity:  convert.EventAccessClassificationToSensitivity(event.AccessClassification),
	}
}

type calendarItem struct {
	*components.Event
	calendar     *calendarListEntry
	responseType rsvp.MeetingResponseType
	organizer    calendar.EmailAddress
	attendees    []calendar.Attendee
	sensitivity  sensitivity.Sensitivity
}

func (item *calendarItem) UID() string {
	return item.Event.UID
}

func (item *calendarItem) Subject() string {
	return item.Event.Summary
}

func (item *calendarItem) Description() string {
	return item.Event.Description
}

func (item *calendarItem) URL() string {
	return unsafeToString(item.Event.Url)
}

func (item *calendarItem) Start() time.Time {
	return item.Event.DateStart.NativeTime()
}

func (item *calendarItem) End() time.Time {
	return item.Event.DateEnd.NativeTime()
}

func (item *calendarItem) TimeZone() string {
	return item.calendar.timeZone
}

func (item *calendarItem) Location() string {
	return unsafeToString(item.Event.Location)
}

func (item *calendarItem) ResponseType() *rsvp.MeetingResponseType {
	return &item.responseType
}

func (item *calendarItem) Organizer() calendar.EmailAddress {
	return item.organizer
}

func (item *calendarItem) Attendees() []calendar.Attendee {
	return item.attendees
}

func (item *calendarItem) IsRecurring() bool {
	return item.Event.IsRecurrence()
}

func (item *calendarItem) IsAllDay() bool {
	return item.Start().Truncate(time.Millisecond).IsZero() && item.End().Truncate(time.Millisecond).IsZero()
}

func (item *calendarItem) Importance() importance.Importance {
	return importance.Unknown
}

func (item *calendarItem) Sensitivity() sensitivity.Sensitivity {
	return item.sensitivity
}

func (item *calendarItem) CreatedAt() time.Time {
	return item.Event.Created.NativeTime()
}

func (item *calendarItem) LastModifiedAt() time.Time {
	if item.Event.LastModified == nil {
		return item.CreatedAt()
	}
	return item.Event.LastModified.NativeTime()
}

func (item *calendarItem) CalendarID() string {
	return item.calendar.path
}

func (item *calendarItem) CalendarDisplayName() string {
	return item.calendar.displayName
}

func (item *calendarItem) CalendarItemID() string {
	return item.Event.UID
}

func unsafeToString(value properties.CanEncodeValue) string {
	if value == nil || reflect.ValueOf(value).IsNil() {
		return ""
	}

	s, err := value.EncodeICalValue()
	if err != nil {
		log.ErrorObject(err)
		return ""
	}
	return s
}

func resolveOrganizer(organizer *values.OrganizerContact) calendar.EmailAddress {
	if organizer == nil { // can be nil (e.g., an apppointment)
		return nil
	}
	return newEmailAddress(organizer.Entry)
}

func resolveAttendees(eventAttendees []*values.Attendee) []calendar.Attendee {
	attendees := make([]calendar.Attendee, 0, len(eventAttendees))
	for _, a := range eventAttendees {
		responseType := convert.ParticipationStatusToMeetingResponseType(a.ParticipationStatus)
		attendees = append(attendees, &attendee{newEmailAddress(a.Entry), &responseType})
	}
	return attendees
}

func findResponseType(emailAddress string, attendees []calendar.Attendee) rsvp.MeetingResponseType {
	for _, attendee := range attendees {
		if attendee.EmailAddress().Address() == emailAddress {
			return *attendee.ResponseType()
		}
	}
	return rsvp.Unknown
}

type attendee struct {
	*emailAddress
	responseType *rsvp.MeetingResponseType
}

func (a *attendee) EmailAddress() calendar.EmailAddress {
	return a.emailAddress
}

func (a *attendee) ResponseType() *rsvp.MeetingResponseType {
	return a.responseType
}

func newEmailAddress(email mail.Address) *emailAddress {
	return &emailAddress{name: email.Name, address: email.Address}
}

type emailAddress struct {
	name    string
	address string
}

func (email *emailAddress) Name() string {
	return email.name
}

func (email *emailAddress) Address() string {
	return email.address
}
