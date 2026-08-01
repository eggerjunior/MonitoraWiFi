package httpapi

import (
	"context"

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
