package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"neo360-api/internal/domain/account"
	"neo360-api/internal/domain/broker"
	"neo360-api/internal/pkg/authz"
)

type fakeRepo struct {
	cred    account.Credentials
	credErr error
	user    account.User
	userErr error
}

func (f fakeRepo) Credentials(context.Context, string) (account.Credentials, error) {
	return f.cred, f.credErr
}
func (f fakeRepo) User(_ context.Context, id string) (account.User, error) {
	u := f.user
	u.ID = id
	return u, f.userErr
}

type fakeBrokers struct {
	created    broker.CreateRequest
	createErr  error
	getErr     error
	profileErr error
}

func (f *fakeBrokers) Create(_ context.Context, r broker.CreateRequest) (broker.Broker, error) {
	f.created = r
	return broker.Broker{ID: "b1", UserID: "u1"}, f.createErr
}
func (f *fakeBrokers) GetByUser(_ context.Context, userID string) (broker.Broker, error) {
	return broker.Broker{ID: "b1", UserID: userID, Code: "FAB-0001"}, f.getErr
}
func (f *fakeBrokers) UpdateProfile(_ context.Context, userID string, _ broker.Profile) (broker.Broker, error) {
	return broker.Broker{UserID: userID}, f.profileErr
}

type fakeTokens struct{ err error }

func (f fakeTokens) Generate(userID, role string) (string, time.Time, error) {
	return "tok:" + userID + ":" + role, time.Unix(100, 0), f.err
}

type fakeComparer struct{ calls []string }

func (f *fakeComparer) Compare(hash, password string) bool {
	f.calls = append(f.calls, hash)
	return hash == "h:"+password
}

func TestLogin(t *testing.T) {
	ctx := context.Background()
	active := account.Credentials{UserID: "u1", Role: authz.RoleAdmin, PasswordHash: "h:senha", Active: true}
	cmp := &fakeComparer{}
	s := NewAccountService(fakeRepo{cred: active, user: account.User{Role: authz.RoleAdmin}}, &fakeBrokers{}, fakeTokens{}, cmp)

	sess, err := s.Login(ctx, account.LoginRequest{Email: " Admin@Neo360.test ", Password: "senha"})
	require.NoError(t, err)
	assert.Equal(t, "tok:u1:admin", sess.Token)
	assert.Equal(t, []string{authz.Admin}, sess.User.Permissions)
	assert.Nil(t, sess.User.Broker, "admin não tem cadastro de corretor")

	_, err = s.Login(ctx, account.LoginRequest{Email: "admin@neo360.test", Password: "errada"})
	assert.ErrorIs(t, err, account.ErrInvalidCredentials)

	inactive := active
	inactive.Active = false
	s = NewAccountService(fakeRepo{cred: inactive}, &fakeBrokers{}, fakeTokens{}, cmp)
	_, err = s.Login(ctx, account.LoginRequest{Password: "senha"})
	assert.ErrorIs(t, err, account.ErrInvalidCredentials, "conta desativada não entra")

	cmp = &fakeComparer{}
	s = NewAccountService(fakeRepo{credErr: account.ErrNotFound}, &fakeBrokers{}, fakeTokens{}, cmp)
	_, err = s.Login(ctx, account.LoginRequest{Password: "x"})
	assert.ErrorIs(t, err, account.ErrInvalidCredentials)
	assert.Equal(t, []string{dummyHash}, cmp.calls, "e-mail inexistente também compara uma senha")

	s = NewAccountService(fakeRepo{credErr: errors.New("db")}, &fakeBrokers{}, fakeTokens{}, cmp)
	_, err = s.Login(ctx, account.LoginRequest{})
	assert.EqualError(t, err, "db")

	s = NewAccountService(fakeRepo{cred: active}, &fakeBrokers{}, fakeTokens{err: errors.New("jwt")}, &fakeComparer{})
	_, err = s.Login(ctx, account.LoginRequest{Password: "senha"})
	assert.EqualError(t, err, "jwt")
}

func TestSignup(t *testing.T) {
	ctx := context.Background()
	brokers := &fakeBrokers{}
	s := NewAccountService(fakeRepo{user: account.User{Role: authz.RoleBroker}}, brokers, fakeTokens{}, &fakeComparer{})
	agency := "a1"

	sess, err := s.Signup(ctx, broker.CreateRequest{Email: "fab@x.com", AgencyID: &agency})
	require.NoError(t, err)
	assert.Equal(t, "tok:u1:corretor", sess.Token, "a conta nasce como corretor")
	assert.Nil(t, brokers.created.AgencyID, "o cadastro aberto não escolhe imobiliária")
	require.NotNil(t, sess.User.Broker)
	assert.Equal(t, "FAB-0001", sess.User.Broker.Code)
	assert.Equal(t, []string{authz.BrokerArea}, sess.User.Permissions)

	s = NewAccountService(fakeRepo{}, &fakeBrokers{createErr: broker.ErrEmailTaken}, fakeTokens{}, &fakeComparer{})
	_, err = s.Signup(ctx, broker.CreateRequest{})
	assert.ErrorIs(t, err, broker.ErrEmailTaken)
}

func TestMeAndProfile(t *testing.T) {
	ctx := context.Background()
	s := NewAccountService(fakeRepo{userErr: account.ErrNotFound}, &fakeBrokers{}, fakeTokens{}, &fakeComparer{})
	_, err := s.Me(ctx, "u1")
	assert.ErrorIs(t, err, account.ErrNotFound)

	s = NewAccountService(fakeRepo{user: account.User{Role: authz.RoleBroker}}, &fakeBrokers{getErr: broker.ErrNotFound}, fakeTokens{}, &fakeComparer{})
	_, err = s.Me(ctx, "u1")
	assert.ErrorIs(t, err, broker.ErrNotFound)

	s = NewAccountService(fakeRepo{user: account.User{Role: authz.RoleBroker}}, &fakeBrokers{}, fakeTokens{}, &fakeComparer{})
	u, err := s.UpdateProfile(ctx, "u1", broker.Profile{})
	require.NoError(t, err)
	assert.Equal(t, "u1", u.Broker.UserID)

	s = NewAccountService(fakeRepo{}, &fakeBrokers{profileErr: broker.ErrInvalidPhone}, fakeTokens{}, &fakeComparer{})
	_, err = s.UpdateProfile(ctx, "u1", broker.Profile{})
	assert.ErrorIs(t, err, broker.ErrInvalidPhone)
}
