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
	cred     account.Credentials
	credErr  error
	user     account.User
	userErr  error
	hash     string
	hashErr  error
	saved    *string
	reset    *bool
	writeErr error
	list     []account.Account
}

func (f fakeRepo) ListAccounts(context.Context) ([]account.Account, error) { return f.list, f.userErr }
func (f fakeRepo) PasswordHash(context.Context, string) (string, error)    { return f.hash, f.hashErr }
func (f fakeRepo) ResetPassword(_ context.Context, _, hash string) error {
	*f.saved, *f.reset = hash, true
	return f.writeErr
}
func (f fakeRepo) ChangePassword(_ context.Context, _, hash string) error {
	*f.saved, *f.reset = hash, false
	return f.writeErr
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

type fakeComparer struct {
	calls   []string
	hashErr error
}

func (f *fakeComparer) Compare(hash, password string) bool {
	f.calls = append(f.calls, hash)
	return hash == "h:"+password
}
func (f *fakeComparer) Hash(password string) (string, error) { return "h:" + password, f.hashErr }

// ADR-0020: a troca confere a senha atual (ou provisória), exige 8 caracteres e uma senha diferente.
func TestChangePassword(t *testing.T) {
	ctx := context.Background()
	saved, reset := "", true
	repo := fakeRepo{user: account.User{Role: authz.RoleAdmin}, hash: "h:k7Hm-2pQx-9sWd", saved: &saved, reset: &reset}
	s := NewAccountService(repo, &fakeBrokers{}, fakeTokens{}, &fakeComparer{})

	u, err := s.ChangePassword(ctx, "u1", account.PasswordChange{Current: "k7Hm-2pQx-9sWd", New: "minha-senha-nova"})
	require.NoError(t, err)
	assert.Equal(t, "u1", u.ID)
	assert.Equal(t, "h:minha-senha-nova", saved, "grava só o hash")
	assert.False(t, reset, "a troca tira a obrigação")

	_, err = s.ChangePassword(ctx, "u1", account.PasswordChange{Current: "errada", New: "minha-senha-nova"})
	assert.ErrorIs(t, err, account.ErrWrongPassword)
	_, err = s.ChangePassword(ctx, "u1", account.PasswordChange{Current: "k7Hm-2pQx-9sWd", New: "curta"})
	assert.ErrorIs(t, err, account.ErrWeakPassword)
	_, err = s.ChangePassword(ctx, "u1", account.PasswordChange{Current: "k7Hm-2pQx-9sWd", New: "k7Hm-2pQx-9sWd"})
	assert.ErrorIs(t, err, account.ErrSamePassword)

	for name, svc := range map[string]*AccountService{
		"hash":   NewAccountService(fakeRepo{hashErr: errors.New("hash"), saved: &saved, reset: &reset}, &fakeBrokers{}, fakeTokens{}, &fakeComparer{}),
		"gerar":  NewAccountService(repo, &fakeBrokers{}, fakeTokens{}, &fakeComparer{hashErr: errors.New("gerar")}),
		"gravar": NewAccountService(fakeRepo{hash: repo.hash, writeErr: errors.New("gravar"), saved: &saved, reset: &reset}, &fakeBrokers{}, fakeTokens{}, &fakeComparer{}),
	} {
		_, err = svc.ChangePassword(ctx, "u1", account.PasswordChange{Current: "k7Hm-2pQx-9sWd", New: "minha-senha-nova"})
		assert.EqualError(t, err, name)
	}
}

// ADR-0020: o reset gera uma senha provisória, grava só o hash e marca a troca no primeiro acesso.
func TestResetPassword(t *testing.T) {
	ctx := context.Background()
	saved, reset := "", false
	s := NewAccountService(fakeRepo{saved: &saved, reset: &reset}, &fakeBrokers{}, fakeTokens{}, &fakeComparer{})
	temp, err := s.ResetPassword(ctx, "u9")
	require.NoError(t, err)
	assert.Regexp(t, `^[a-hj-km-np-zA-HJ-NP-Z2-9]{4}-[a-hj-km-np-zA-HJ-NP-Z2-9]{4}-[a-hj-km-np-zA-HJ-NP-Z2-9]{4}$`, temp.Password)
	assert.Equal(t, "h:"+temp.Password, saved, "o banco guarda só o hash")
	assert.True(t, reset)
	other, _ := s.ResetPassword(ctx, "u9")
	assert.NotEqual(t, temp.Password, other.Password, "cada reset gera uma senha nova")

	s.newTemp = func() (string, error) { return "", errors.New("rand") }
	_, err = s.ResetPassword(ctx, "u9")
	assert.EqualError(t, err, "rand")
	s = NewAccountService(fakeRepo{saved: &saved, reset: &reset}, &fakeBrokers{}, fakeTokens{}, &fakeComparer{hashErr: errors.New("hash")})
	_, err = s.ResetPassword(ctx, "u9")
	assert.EqualError(t, err, "hash")
	s = NewAccountService(fakeRepo{saved: &saved, reset: &reset, writeErr: account.ErrNotFound}, &fakeBrokers{}, fakeTokens{}, &fakeComparer{})
	_, err = s.ResetPassword(ctx, "nao-existe")
	assert.ErrorIs(t, err, account.ErrNotFound)

	list, err := NewAccountService(fakeRepo{list: []account.Account{{ID: "u1"}}}, &fakeBrokers{}, fakeTokens{}, &fakeComparer{}).ListAccounts(ctx)
	require.NoError(t, err)
	assert.Len(t, list, 1)
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
