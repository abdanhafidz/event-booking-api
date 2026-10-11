// Package integration runs the API in-process against a real PostgreSQL
// database. Set TEST_DATABASE_URL to run it, for example:
//
//	TEST_DATABASE_URL="host=localhost port=5433 user=postgres dbname=booking_test sslmode=disable" go test ./tests/integration/...
package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"sync"
	"testing"
	"time"

	"abdanhafidz.com/event-booking-api/config"
	entity "abdanhafidz.com/event-booking-api/models/entity"
	"abdanhafidz.com/event-booking-api/provider"
	"abdanhafidz.com/event-booking-api/router"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

func TestMain(m *testing.M) {
	gin.SetMode(gin.TestMode)
	slog.SetDefault(slog.New(slog.NewTextHandler(io.Discard, nil)))
	os.Exit(m.Run())
}

type options struct {
	hold           time.Duration
	latency        time.Duration
	sweeper        bool
	sweepInterval  time.Duration
	idempotencyTTL time.Duration
}

type option func(*options)

func withHold(d time.Duration) option           { return func(o *options) { o.hold = d } }
func withLatency(d time.Duration) option        { return func(o *options) { o.latency = d } }
func withIdempotencyTTL(d time.Duration) option { return func(o *options) { o.idempotencyTTL = d } }
func withSweeper(interval time.Duration) option {
	return func(o *options) { o.sweeper = true; o.sweepInterval = interval }
}

type app struct {
	t           *testing.T
	srv         *httptest.Server
	db          *gorm.DB
	provider    provider.AppProvider
	client      *http.Client
	mu          sync.Mutex
	ticketTypes []uuid.UUID
}

// newApp starts the API with its own configuration. After the test it checks
// the inventory invariant (BR-10) for every ticket type the test created.
func newApp(t *testing.T, opts ...option) *app {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not set")
	}
	o := options{hold: 10 * time.Minute, sweepInterval: 5 * time.Second, idempotencyTTL: 24 * time.Hour}
	for _, fn := range opts {
		fn(&o)
	}

	dbConfig := config.NewDatabaseConfigFromDSN(dsn, 100)
	bookingConfig := config.NewBookingConfig(o.hold, o.sweepInterval, o.sweeper, 2*time.Second, o.latency, o.idempotencyTTL)
	p := provider.NewAppProviderWith(provider.NewConfigProviderWith(config.NewEnvConfig(), dbConfig, bookingConfig))
	srv := httptest.NewServer(router.SetupRouter(p))

	a := &app{
		t:        t,
		srv:      srv,
		db:       dbConfig.GetInstance(),
		provider: p,
		client: &http.Client{
			Timeout:   10 * time.Second,
			Transport: &http.Transport{MaxIdleConns: 400, MaxIdleConnsPerHost: 400},
		},
	}

	ctx, cancel := context.WithCancel(context.Background())
	if o.sweeper {
		p.ProvideServices().ProvideExpiryService().Start(ctx, o.sweepInterval)
	}

	t.Cleanup(func() {
		cancel()
		srv.Close()
		if sqlDB, err := a.db.DB(); err == nil {
			sqlDB.Close()
		}
	})
	t.Cleanup(a.checkInvariant)
	return a
}

func (a *app) checkInvariant() {
	a.t.Helper()
	for _, id := range a.ticketTypes {
		var row struct {
			Capacity  int
			Available int
			Held      int
		}
		err := a.db.Raw(`
			SELECT t.capacity, t.available, COALESCE(SUM(b.quantity) FILTER (
			         WHERE b.status IN ('PENDING_PAYMENT', 'CONFIRMED')), 0) AS held
			FROM ticket_types t LEFT JOIN bookings b ON b.ticket_type_id = t.id
			WHERE t.id = ? GROUP BY t.id`, id).Scan(&row).Error
		if err != nil {
			a.t.Errorf("invariant query: %v", err)
			continue
		}
		if row.Capacity != row.Available+row.Held || row.Available < 0 {
			a.t.Errorf("BR-10 violated for ticket type %s: capacity=%d available=%d held=%d",
				id, row.Capacity, row.Available, row.Held)
		}
	}
}

// ---------- fixtures ----------

func (a *app) user() uuid.UUID {
	a.t.Helper()
	u := entity.User{Id: uuid.New(), Name: "it-user"}
	if err := a.db.Create(&u).Error; err != nil {
		a.t.Fatalf("create user: %v", err)
	}
	return u.Id
}

func (a *app) users(n int) []uuid.UUID {
	a.t.Helper()
	list := make([]entity.User, n)
	ids := make([]uuid.UUID, n)
	for i := range list {
		ids[i] = uuid.New()
		list[i] = entity.User{Id: ids[i], Name: "it-user"}
	}
	if err := a.db.CreateInBatches(&list, 500).Error; err != nil {
		a.t.Fatalf("create users: %v", err)
	}
	return ids
}

// event creates an event starting after startsIn, with one ticket type per
// capacity, priced at 150,000 IDR.
func (a *app) event(startsIn time.Duration, capacities ...int) (uuid.UUID, []uuid.UUID) {
	a.t.Helper()
	ev := entity.Event{Name: "it-" + a.t.Name(), StartsAt: time.Now().Add(startsIn)}
	for i, c := range capacities {
		ev.TicketTypes = append(ev.TicketTypes, entity.TicketType{
			Name: "Tier " + string(rune('A'+i)), Price: 150000, Capacity: c, Available: c,
		})
	}
	if err := a.db.Create(&ev).Error; err != nil {
		a.t.Fatalf("create event: %v", err)
	}
	ids := make([]uuid.UUID, len(ev.TicketTypes))
	for i, tt := range ev.TicketTypes {
		ids[i] = tt.Id
	}
	a.mu.Lock()
	a.ticketTypes = append(a.ticketTypes, ids...)
	a.mu.Unlock()
	return ev.Id, ids
}

// ticketType is the shared setup: one ticket type R with the given capacity on
// an event 7 days away.
func (a *app) ticketType(capacity int) uuid.UUID {
	_, ids := a.event(7*24*time.Hour, capacity)
	return ids[0]
}

func (a *app) available(tt uuid.UUID) int {
	a.t.Helper()
	var n int
	if err := a.db.Raw(`SELECT available FROM ticket_types WHERE id = ?`, tt).Scan(&n).Error; err != nil {
		a.t.Fatalf("available: %v", err)
	}
	return n
}

func (a *app) status(booking uuid.UUID) string {
	a.t.Helper()
	var s string
	if err := a.db.Raw(`SELECT status FROM bookings WHERE id = ?`, booking).Scan(&s).Error; err != nil {
		a.t.Fatalf("status: %v", err)
	}
	return s
}

func (a *app) payments(booking uuid.UUID, status string) int {
	a.t.Helper()
	var n int
	if err := a.db.Raw(`SELECT count(*) FROM payments WHERE booking_id = ? AND status = ?`, booking, status).Scan(&n).Error; err != nil {
		a.t.Fatalf("payments: %v", err)
	}
	return n
}

func (a *app) bookingsFor(tt uuid.UUID) int {
	a.t.Helper()
	var n int
	if err := a.db.Raw(`SELECT count(*) FROM bookings WHERE ticket_type_id = ?`, tt).Scan(&n).Error; err != nil {
		a.t.Fatalf("bookings: %v", err)
	}
	return n
}

func (a *app) exec(sql string, args ...any) {
	a.t.Helper()
	if err := a.db.Exec(sql, args...).Error; err != nil {
		a.t.Fatalf("exec: %v", err)
	}
}

func (a *app) sweep() {
	a.t.Helper()
	if _, err := a.provider.ProvideServices().ProvideExpiryService().SweepOnce(context.Background()); err != nil {
		a.t.Fatalf("sweep: %v", err)
	}
}

// ---------- HTTP ----------

type response struct {
	Code     int
	Header   http.Header
	Status   string          `json:"status"`
	ErrCode  string          `json:"code"`
	Data     json.RawMessage `json:"data"`
	Errors   map[string]any  `json:"errors"`
	RawBody  []byte
	Duration time.Duration
}

type booking struct {
	Id          uuid.UUID `json:"id"`
	Quantity    int       `json:"quantity"`
	UnitPrice   int64     `json:"unit_price"`
	TotalAmount int64     `json:"total_amount"`
	Status      string    `json:"status"`
	ExpiresAt   time.Time `json:"expires_at"`
	CreatedAt   time.Time `json:"created_at"`
}

func (r response) booking(t *testing.T) booking {
	t.Helper()
	var b booking
	if err := json.Unmarshal(r.Data, &b); err != nil {
		t.Fatalf("decode booking: %v (body %s)", err, r.RawBody)
	}
	return b
}

// request sends a request; user and key are omitted when zero/empty.
func (a *app) request(method, path string, user uuid.UUID, key string, body any) response {
	a.t.Helper()
	var reader io.Reader
	if body != nil {
		switch b := body.(type) {
		case string:
			reader = bytes.NewBufferString(b)
		default:
			raw, _ := json.Marshal(b)
			reader = bytes.NewBuffer(raw)
		}
	}
	req, _ := http.NewRequest(method, a.srv.URL+path, reader)
	req.Header.Set("Content-Type", "application/json")
	if user != uuid.Nil {
		req.Header.Set("X-User-Id", user.String())
	}
	if key != "" {
		req.Header.Set("Idempotency-Key", key)
	}
	start := time.Now()
	res, err := a.client.Do(req)
	if err != nil {
		a.t.Errorf("%s %s: %v", method, path, err)
		return response{}
	}
	defer res.Body.Close()
	raw, _ := io.ReadAll(res.Body)
	out := response{Code: res.StatusCode, Header: res.Header, RawBody: raw, Duration: time.Since(start)}
	_ = json.Unmarshal(raw, &out)
	return out
}

func key() string { return uuid.NewString() }

func (a *app) createBooking(user, tt uuid.UUID, quantity int, idemKey string) response {
	return a.request("POST", "/v1/bookings", user, idemKey, map[string]any{"ticket_type_id": tt, "quantity": quantity})
}

func (a *app) pay(user, b uuid.UUID, token, idemKey string) response {
	return a.request("POST", "/v1/bookings/"+b.String()+"/pay", user, idemKey, map[string]any{"payment_token": token})
}

func (a *app) cancel(user, b uuid.UUID) response {
	return a.request("POST", "/v1/bookings/"+b.String()+"/cancel", user, "", nil)
}

// book creates a booking and fails the test unless it is 201.
func (a *app) book(user, tt uuid.UUID, quantity int) booking {
	a.t.Helper()
	res := a.createBooking(user, tt, quantity, key())
	expectCode(a.t, res, 201, "")
	return res.booking(a.t)
}

func expectCode(t *testing.T, res response, httpCode int, errCode string) {
	t.Helper()
	if res.Code != httpCode || res.ErrCode != errCode {
		t.Fatalf("want %d %q, got %d %q: %s", httpCode, errCode, res.Code, res.ErrCode, res.RawBody)
	}
}

func expectEqual[T comparable](t *testing.T, what string, want, got T) {
	t.Helper()
	if want != got {
		t.Fatalf("%s: want %v, got %v", what, want, got)
	}
}

// parallel runs n calls at once, released together, and returns their results.
func parallel(n int, fn func(i int) response) []response {
	results := make([]response, n)
	start := make(chan struct{})
	var wg sync.WaitGroup
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			<-start
			results[i] = fn(i)
		}(i)
	}
	close(start)
	wg.Wait()
	return results
}

func countCodes(results []response) map[string]int {
	counts := map[string]int{}
	for _, r := range results {
		k := http.StatusText(r.Code)
		if r.ErrCode != "" {
			k = r.ErrCode
		}
		counts[k]++
	}
	return counts
}
