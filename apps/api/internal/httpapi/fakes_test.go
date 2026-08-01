package httpapi

import (
	"context"
	"time"

	"github.com/google/uuid"

	"egger/api/internal/store"
)

// fakes_test.go contém implementações em memória das interfaces de repositório,
// usadas só em teste — permite testar handlers sem PostgreSQL real (Seção 21).

type fakeOrgs struct {
	items []store.Organization
}

func (f *fakeOrgs) List(ctx context.Context, page store.Page) ([]store.Organization, int, error) {
	return f.items, len(f.items), nil
}

type fakeSites struct {
	items []store.Site
}

func (f *fakeSites) ListByOrganization(ctx context.Context, orgID uuid.UUID, page store.Page) ([]store.Site, int, error) {
	var out []store.Site
	for _, s := range f.items {
		if s.OrganizationID == orgID {
			out = append(out, s)
		}
	}
	return out, len(out), nil
}

func (f *fakeSites) Get(ctx context.Context, id uuid.UUID) (store.Site, error) {
	for _, s := range f.items {
		if s.ID == id {
			return s, nil
		}
	}
	return store.Site{}, store.ErrNotFound
}

type fakeUsers struct {
	byEmail map[string]store.User
	byID    map[uuid.UUID]store.User
}

func newFakeUsers(users ...store.User) *fakeUsers {
	f := &fakeUsers{byEmail: map[string]store.User{}, byID: map[uuid.UUID]store.User{}}
	for _, u := range users {
		f.byEmail[u.Email] = u
		f.byID[u.ID] = u
	}
	return f
}

func (f *fakeUsers) GetByEmail(ctx context.Context, email string) (store.User, error) {
	u, ok := f.byEmail[email]
	if !ok {
		return store.User{}, store.ErrNotFound
	}
	return u, nil
}

func (f *fakeUsers) Get(ctx context.Context, id uuid.UUID) (store.User, error) {
	u, ok := f.byID[id]
	if !ok {
		return store.User{}, store.ErrNotFound
	}
	return u, nil
}

type fakeSessions struct {
	byHash map[string]store.Session
}

func newFakeSessions() *fakeSessions {
	return &fakeSessions{byHash: map[string]store.Session{}}
}

func (f *fakeSessions) Create(ctx context.Context, s store.Session) error {
	f.byHash[s.TokenHash] = s
	return nil
}

func (f *fakeSessions) GetByTokenHash(ctx context.Context, tokenHash string) (store.Session, error) {
	s, ok := f.byHash[tokenHash]
	if !ok {
		return store.Session{}, store.ErrNotFound
	}
	return s, nil
}

func (f *fakeSessions) Revoke(ctx context.Context, id uuid.UUID) error {
	for hash, s := range f.byHash {
		if s.ID == id {
			now := s.CreatedAt
			s.RevokedAt = &now
			f.byHash[hash] = s
		}
	}
	return nil
}

type fakeAudit struct {
	entries []store.AuditEntry
}

func (f *fakeAudit) Record(ctx context.Context, e store.AuditEntry) error {
	f.entries = append(f.entries, e)
	return nil
}

type fakePinger struct {
	err error
}

func (f *fakePinger) Ping(ctx context.Context) error {
	return f.err
}

type fakeAgents struct {
	byID map[uuid.UUID]store.Agent
}

func newFakeAgents() *fakeAgents {
	return &fakeAgents{byID: map[uuid.UUID]store.Agent{}}
}

func (f *fakeAgents) Create(ctx context.Context, a store.Agent) error {
	f.byID[a.ID] = a
	return nil
}

func (f *fakeAgents) Get(ctx context.Context, id uuid.UUID) (store.Agent, error) {
	a, ok := f.byID[id]
	if !ok {
		return store.Agent{}, store.ErrNotFound
	}
	return a, nil
}

func (f *fakeAgents) ListBySite(ctx context.Context, siteID uuid.UUID, page store.Page) ([]store.Agent, int, error) {
	var out []store.Agent
	for _, a := range f.byID {
		if a.SiteID == siteID {
			out = append(out, a)
		}
	}
	return out, len(out), nil
}

func (f *fakeAgents) UpdateLastSeen(ctx context.Context, id uuid.UUID, at time.Time) error {
	a, ok := f.byID[id]
	if !ok {
		return store.ErrNotFound
	}
	a.LastSeenAt = &at
	f.byID[id] = a
	return nil
}

type fakeAgentEnrollmentTokens struct {
	byHash map[string]store.AgentEnrollmentToken
}

func newFakeAgentEnrollmentTokens() *fakeAgentEnrollmentTokens {
	return &fakeAgentEnrollmentTokens{byHash: map[string]store.AgentEnrollmentToken{}}
}

func (f *fakeAgentEnrollmentTokens) Create(ctx context.Context, t store.AgentEnrollmentToken) error {
	f.byHash[t.TokenHash] = t
	return nil
}

func (f *fakeAgentEnrollmentTokens) GetValidByTokenHash(ctx context.Context, tokenHash string, now time.Time) (store.AgentEnrollmentToken, error) {
	t, ok := f.byHash[tokenHash]
	if !ok || t.UsedAt != nil || !t.ExpiresAt.After(now) {
		return store.AgentEnrollmentToken{}, store.ErrTokenExpiredOrUsed
	}
	return t, nil
}

func (f *fakeAgentEnrollmentTokens) MarkUsed(ctx context.Context, id uuid.UUID, agentID uuid.UUID, at time.Time) error {
	for hash, t := range f.byHash {
		if t.ID == id {
			t.UsedAt = &at
			t.UsedByAgentID = &agentID
			f.byHash[hash] = t
		}
	}
	return nil
}

type fakeAgentHeartbeats struct {
	entries []store.AgentHeartbeat
}

func (f *fakeAgentHeartbeats) Record(ctx context.Context, h store.AgentHeartbeat) error {
	f.entries = append(f.entries, h)
	return nil
}

type fakePingTests struct {
	byKey map[string]store.PingTest
}

func newFakePingTests() *fakePingTests {
	return &fakePingTests{byKey: map[string]store.PingTest{}}
}

func (f *fakePingTests) InsertBatch(ctx context.Context, tests []store.PingTest) error {
	for _, t := range tests {
		key := t.AgentID.String() + "|" + t.IdempotencyKey
		if _, exists := f.byKey[key]; exists {
			continue
		}
		f.byKey[key] = t
	}
	return nil
}

func (f *fakePingTests) ListBySite(ctx context.Context, siteID uuid.UUID, page store.Page) ([]store.PingTest, int, error) {
	var out []store.PingTest
	for _, t := range f.byKey {
		out = append(out, t)
	}
	return out, len(out), nil
}

type fakeSpeedTests struct {
	byKey map[string]store.SpeedTest
}

func newFakeSpeedTests() *fakeSpeedTests {
	return &fakeSpeedTests{byKey: map[string]store.SpeedTest{}}
}

func (f *fakeSpeedTests) InsertBatch(ctx context.Context, tests []store.SpeedTest) error {
	for _, t := range tests {
		key := t.AgentID.String() + "|" + t.IdempotencyKey
		if _, exists := f.byKey[key]; exists {
			continue
		}
		f.byKey[key] = t
	}
	return nil
}

func (f *fakeSpeedTests) ListBySite(ctx context.Context, siteID uuid.UUID, page store.Page) ([]store.SpeedTest, int, error) {
	var out []store.SpeedTest
	for _, t := range f.byKey {
		out = append(out, t)
	}
	return out, len(out), nil
}
