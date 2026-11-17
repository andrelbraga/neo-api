package service

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"neo360-api/internal/domain/broker"
)

type fakeRepo struct {
	codes     []string
	taken     int
	err       error
	hash      string
	updatedID string
}

func (f *fakeRepo) Create(_ context.Context, r broker.CreateRequest, hash, code string) (broker.Broker, error) {
	f.codes = append(f.codes, code)
	f.hash = hash
	if len(f.codes) <= f.taken {
		return broker.Broker{}, broker.ErrCodeTaken
	}
	return broker.Broker{ID: "b1", Code: code, Email: r.Email}, f.err
}
func (f *fakeRepo) List(context.Context) ([]broker.Broker, error) {
	return []broker.Broker{{ID: "b1"}}, f.err
}
func (f *fakeRepo) Get(_ context.Context, id string) (broker.Broker, error) {
	return broker.Broker{ID: id}, f.err
}
func (f *fakeRepo) GetByUser(_ context.Context, userID string) (broker.Broker, error) {
	return broker.Broker{UserID: userID}, f.err
}
func (f *fakeRepo) Update(_ context.Context, id string, r broker.UpdateRequest) (broker.Broker, error) {
	f.updatedID = id
	return broker.Broker{ID: id, Phone: r.Phone}, f.err
}
func (f *fakeRepo) UpdateProfile(_ context.Context, userID string, p broker.Profile) (broker.Broker, error) {
	return broker.Broker{UserID: userID, Phone: p.Phone}, f.err
}

type fakeHasher struct{ err error }

func (f fakeHasher) Hash(p string) (string, error) { return "hash:" + p, f.err }

func newService(repo *fakeRepo, h fakeHasher) *BrokerService {
	s := NewBrokerService(repo, h)
	n := 0
	s.random = func() int { n++; return n }
	return s
}

var valid = broker.CreateRequest{
	Profile:  broker.Profile{FirstName: "Fabiano", LastName: "Souza", Phone: "54999990000"},
	Email:    "fab@exemplo.com",
	Password: "senha-longa",
}

func TestCreate(t *testing.T) {
	repo := &fakeRepo{}
	b, err := newService(repo, fakeHasher{}).Create(context.Background(), valid)
	require.NoError(t, err)
	assert.Equal(t, "FAB-0001", b.Code)
	assert.Equal(t, "hash:senha-longa", repo.hash, "a senha nunca vai em claro para o banco")

	repo = &fakeRepo{taken: 2}
	b, err = newService(repo, fakeHasher{}).Create(context.Background(), valid)
	require.NoError(t, err)
	assert.Equal(t, []string{"FAB-0001", "FAB-0002", "FAB-0003"}, repo.codes, "código em uso: sorteia outro")
	assert.Equal(t, "FAB-0003", b.Code)

	repo = &fakeRepo{taken: 99}
	_, err = newService(repo, fakeHasher{}).Create(context.Background(), valid)
	assert.ErrorIs(t, err, broker.ErrCodeTaken)
	assert.Len(t, repo.codes, codeAttempts)

	_, err = newService(&fakeRepo{}, fakeHasher{}).Create(context.Background(), broker.CreateRequest{})
	assert.ErrorIs(t, err, broker.ErrNameRequired)

	_, err = newService(&fakeRepo{}, fakeHasher{err: errors.New("bcrypt")}).Create(context.Background(), valid)
	assert.EqualError(t, err, "bcrypt")

	_, err = newService(&fakeRepo{err: broker.ErrEmailTaken}, fakeHasher{}).Create(context.Background(), valid)
	assert.ErrorIs(t, err, broker.ErrEmailTaken)
}

func TestReadAndUpdate(t *testing.T) {
	repo := &fakeRepo{}
	s := newService(repo, fakeHasher{})
	ctx := context.Background()

	list, err := s.List(ctx)
	require.NoError(t, err)
	assert.Len(t, list, 1)
	b, err := s.Get(ctx, "b9")
	require.NoError(t, err)
	assert.Equal(t, "b9", b.ID)
	b, err = s.GetByUser(ctx, "u9")
	require.NoError(t, err)
	assert.Equal(t, "u9", b.UserID)

	b, err = s.Update(ctx, "b1", broker.UpdateRequest{Profile: broker.Profile{FirstName: "A", LastName: "B", Phone: "(54) 99999-0000"}})
	require.NoError(t, err)
	assert.Equal(t, "54999990000", b.Phone)
	_, err = s.Update(ctx, "b1", broker.UpdateRequest{})
	assert.ErrorIs(t, err, broker.ErrNameRequired)

	b, err = s.UpdateProfile(ctx, "u1", broker.Profile{FirstName: "A", LastName: "B", Phone: "54 99999 0000"})
	require.NoError(t, err)
	assert.Equal(t, "54999990000", b.Phone)
	_, err = s.UpdateProfile(ctx, "u1", broker.Profile{FirstName: "A", LastName: "B", Phone: "1"})
	assert.ErrorIs(t, err, broker.ErrInvalidPhone)
}
