package caldav

import (
	"net/http"
	"net/url"
	"time"

	"github.com/WorkFit/caldav-go/caldav"
	"github.com/WorkFit/caldav-go/webdav"
	"github.com/WorkFit/caldav-go/webdav/entities"
	common "github.com/WorkFit/commongo/log"
	"github.com/WorkFit/commongo/web"
	"github.com/WorkFit/go/calendar"
	"github.com/WorkFit/go/errors"
	"github.com/WorkFit/go/log"
)

const (
	reportMethod  = "REPORT"
	depth         = "Depth"
	prefer        = "Prefer"
	returnMinimal = "return-minimal"
)

var (
	paths = []string{"", "/caldav", "/caldav/st", "/.well-known/caldav"}
	// Adds custom headers and logging to all CalDAV requests
	transport = &customHeadersRoundTripper{innerRoundTripper: loggingTransport, depth: "1", prefer: returnMinimal}
	// Adds a leveled logging with a CalDav: prefix to all CalDAV requests
	loggingTransport = web.NewLeveledLoggerRoundTripper(
		http.DefaultTransport,
		common.NewPrefixedLeveledLogger(log.CurrentLogger(), "CalDAV:"))
	findCurrentUserPrincipalRequestBody = &entities.Propfind{
		Props: []*entities.Prop{
			{CurrentUserPrincipal: &entities.CurrentUserPrincipal{}},
		},
	}
	findCalendarHomeSetRequestBody = &entities.Propfind{
		Props: []*entities.Prop{{CalendarHomeSet: &entities.CalendarHomeSet{}}},
	}
)

// NewClient creates a new authenticated CalDAV client.
func NewClient(host string, username string, password string) (calendar.Client, error) {
	httpClient := &http.Client{
		Timeout:   time.Minute,
		Transport: web.NewBasicAuthRoundTripper(transport, username, password),
	}

	calendarClient, calendarHomeSet, err := discoverServer(host, httpClient)
	if err != nil {
		return nil, err
	}

	path, err := url.QueryUnescape(calendarHomeSet.Href)
	if err != nil {
		return nil, err
	}

	return &client{
		path:           path,
		emailAddress:   username,
		calendarClient: calendarClient,
		httpClient:     httpClient,
	}, nil
}

type client struct {
	path           string
	emailAddress   string
	calendarClient *caldav.Client
	httpClient     *http.Client `test-hook:"verify-unexported"`
}

func discoverServer(host string, client *http.Client) (*caldav.Client, *entities.CalendarHomeSet, error) {
	// See https://tools.ietf.org/html/rfc6764 for thorough discovery methods.
	errs := []error{}
	for _, path := range paths {
		candidate, err := caldav.NewServer("https://" + host)
		client := caldav.NewClient(candidate, client)
		calendarHomeSet, err := findCalendarHomeSet(client, path)
		if err != nil {
			errs = append(errs, err)
		} else {
			return client, calendarHomeSet, nil
		}
	}
	return nil, nil, errors.WF11301(errs...)
}

func findCalendarHomeSet(client *caldav.Client, path string) (*entities.CalendarHomeSet, error) {
	multistatus, err := client.WebDAV().Propfind(path, webdav.Depth0, findCurrentUserPrincipalRequestBody)
	if err != nil {
		return nil, err
	}

	path, err = url.QueryUnescape(multistatus.Responses[0].PropStats[0].Prop.CurrentUserPrincipal.Href)
	if err != nil {
		return nil, err
	}

	multistatus, err = client.WebDAV().Propfind(path, webdav.Depth0, findCalendarHomeSetRequestBody)
	if err != nil {
		return nil, err
	}
	return multistatus.Responses[0].PropStats[0].Prop.CalendarHomeSet, nil
}

type customHeadersRoundTripper struct {
	innerRoundTripper http.RoundTripper
	depth             string
	prefer            string
}

func (transport *customHeadersRoundTripper) RoundTrip(request *http.Request) (*http.Response, error) {
	if request.Method == reportMethod {
		request.Header.Add(depth, transport.depth)
	}
	request.Header.Add(prefer, transport.prefer)
	return transport.innerRoundTripper.RoundTrip(request)
}
