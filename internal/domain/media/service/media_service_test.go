package service

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"neo360-api/internal/domain/media"
	"neo360-api/internal/infra/storage"
)

var (
	jpeg  = append([]byte{0xFF, 0xD8, 0xFF, 0xE0, 0, 0x10, 'J', 'F', 'I', 'F', 0}, make([]byte, 64)...)
	errDB = errors.New("db")
)

type fakeRepo struct {
	oldKey, setKey string
	setErr         error
	inUse          bool
	inUseErr       error
	leadID         string
	leadErr        error
	count          int
	countErr       error
	added          []media.Stored
	addErr         error
	photo          media.Stored
	photoErr       error
	photoUser      string
}

func (f *fakeRepo) SetBrokerPhoto(_ context.Context, _, key string) (string, error) {
	f.setKey = key
	return f.oldKey, f.setErr
}
func (f *fakeRepo) BrokerPhotoInUse(context.Context, string) (bool, error) {
	return f.inUse, f.inUseErr
}
func (f *fakeRepo) LeadByToken(_ context.Context, h string) (string, error) {
	if h != hash("tok") {
		return "", media.ErrInvalidToken
	}
	return f.leadID, f.leadErr
}
func (f *fakeRepo) CountLeadPhotos(context.Context, string) (int, error) { return f.count, f.countErr }
func (f *fakeRepo) AddLeadPhoto(_ context.Context, _ string, s media.Stored, size int) (media.Photo, error) {
	f.added = append(f.added, s)
	return media.Photo{ID: "p1", ContentType: s.ContentType, Size: size}, f.addErr
}
func (f *fakeRepo) LeadPhoto(_ context.Context, userID, _, _ string) (media.Stored, error) {
	f.photoUser = userID
	return f.photo, f.photoErr
}

type fakeBucket struct {
	put     map[string]string
	deleted []string
	putErr  error
	delErr  error
	getErr  error
}

func newBucket() *fakeBucket { return &fakeBucket{put: map[string]string{}} }

func (b *fakeBucket) Put(_ context.Context, key, ct string, _ []byte) error {
	if b.putErr != nil {
		return b.putErr
	}
	b.put[key] = ct
	return nil
}
func (b *fakeBucket) Get(_ context.Context, key string) (storage.Object, error) {
	if b.getErr != nil {
		return storage.Object{}, b.getErr
	}
	return storage.Object{Body: io.NopCloser(strings.NewReader(key)), ContentType: "image/jpeg", Size: int64(len(key))}, nil
}
func (b *fakeBucket) Delete(_ context.Context, key string) error {
	b.deleted = append(b.deleted, key)
	return b.delErr
}

type fakeLog struct{ n int }

func (l *fakeLog) Error(string, ...any) { l.n++ }

func TestSetBrokerPhoto(t *testing.T) {
	repo, bucket, log := &fakeRepo{oldKey: "brokers/old.jpg"}, newBucket(), &fakeLog{}
	s := NewMediaService(repo, bucket, log)
	p, err := s.SetBrokerPhoto(context.Background(), "u1", jpeg)
	require.NoError(t, err)
	assert.Regexp(t, `^brokers/[0-9a-f]{32}\.jpg$`, repo.setKey)
	assert.Equal(t, "image/jpeg", bucket.put[repo.setKey])
	assert.Equal(t, "/public/media/"+repo.setKey, p.PhotoPath)
	assert.Equal(t, []string{"brokers/old.jpg"}, bucket.deleted, "a foto anterior sai do bucket")

	bucket.delErr = errDB
	_, err = s.SetBrokerPhoto(context.Background(), "u1", jpeg)
	assert.NoError(t, err, "falha ao apagar a anterior não volta ao usuário")
	assert.Equal(t, 1, log.n)
}

func TestSetBrokerPhoto_Erros(t *testing.T) {
	ctx := context.Background()
	_, err := NewMediaService(&fakeRepo{}, nil, &fakeLog{}).SetBrokerPhoto(ctx, "u1", jpeg)
	assert.ErrorIs(t, err, media.ErrUnavailable)

	s := NewMediaService(&fakeRepo{}, newBucket(), &fakeLog{})
	_, err = s.SetBrokerPhoto(ctx, "u1", []byte("<svg></svg>"))
	assert.ErrorIs(t, err, media.ErrNotImage)
	_, err = s.SetBrokerPhoto(ctx, "u1", nil)
	assert.ErrorIs(t, err, media.ErrNotImage)
	_, err = s.SetBrokerPhoto(ctx, "u1", make([]byte, media.MaxBrokerPhotoBytes+1))
	assert.ErrorIs(t, err, media.ErrTooLarge)

	bucket := newBucket()
	bucket.putErr = errDB
	_, err = NewMediaService(&fakeRepo{}, bucket, &fakeLog{}).SetBrokerPhoto(ctx, "u1", jpeg)
	assert.ErrorIs(t, err, errDB)

	repo, bucket := &fakeRepo{setErr: media.ErrBrokerNotFound}, newBucket()
	_, err = NewMediaService(repo, bucket, &fakeLog{}).SetBrokerPhoto(ctx, "u1", jpeg)
	assert.ErrorIs(t, err, media.ErrBrokerNotFound)
	assert.Equal(t, []string{repo.setKey}, bucket.deleted, "a foto nova sai do bucket se o banco falhou")
}

func TestRemoveBrokerPhoto(t *testing.T) {
	repo, bucket := &fakeRepo{oldKey: "brokers/old.jpg"}, newBucket()
	require.NoError(t, NewMediaService(repo, bucket, &fakeLog{}).RemoveBrokerPhoto(context.Background(), "u1"))
	assert.Equal(t, "", repo.setKey)
	assert.Equal(t, []string{"brokers/old.jpg"}, bucket.deleted)

	assert.NoError(t, NewMediaService(&fakeRepo{oldKey: "x"}, nil, &fakeLog{}).RemoveBrokerPhoto(context.Background(), "u1"))
	assert.ErrorIs(t, NewMediaService(&fakeRepo{setErr: errDB}, newBucket(), &fakeLog{}).RemoveBrokerPhoto(context.Background(), "u1"), errDB)
}

func TestBrokerPhoto(t *testing.T) {
	ctx := context.Background()
	file := "0123456789abcdef0123456789abcdef.jpg"
	obj, err := NewMediaService(&fakeRepo{inUse: true}, newBucket(), &fakeLog{}).BrokerPhoto(ctx, file)
	require.NoError(t, err)
	body, _ := io.ReadAll(obj.Body)
	assert.Equal(t, "brokers/"+file, string(body))

	for name, s := range map[string]*MediaService{
		"sem uso":    NewMediaService(&fakeRepo{}, newBucket(), &fakeLog{}),
		"sem bucket": NewMediaService(&fakeRepo{inUse: true}, nil, &fakeLog{}),
	} {
		_, err := s.BrokerPhoto(ctx, file)
		assert.ErrorIs(t, err, media.ErrNotFound, name)
	}
	_, err = NewMediaService(&fakeRepo{inUse: true}, newBucket(), &fakeLog{}).BrokerPhoto(ctx, "../leads/x.jpg")
	assert.ErrorIs(t, err, media.ErrNotFound)
	_, err = NewMediaService(&fakeRepo{inUseErr: errDB}, newBucket(), &fakeLog{}).BrokerPhoto(ctx, file)
	assert.ErrorIs(t, err, errDB)

	bucket := newBucket()
	bucket.getErr = storage.ErrNotFound
	_, err = NewMediaService(&fakeRepo{inUse: true}, bucket, &fakeLog{}).BrokerPhoto(ctx, file)
	assert.ErrorIs(t, err, media.ErrNotFound)
}

func TestUploadLeadPhoto(t *testing.T) {
	ctx := context.Background()
	repo, bucket := &fakeRepo{leadID: "l1"}, newBucket()
	p, err := NewMediaService(repo, bucket, &fakeLog{}).UploadLeadPhoto(ctx, " tok ", jpeg)
	require.NoError(t, err)
	assert.Equal(t, "p1", p.ID)
	require.Len(t, repo.added, 1)
	assert.Regexp(t, `^leads/l1/[0-9a-f]{32}\.jpg$`, repo.added[0].Key)
	assert.Contains(t, bucket.put, repo.added[0].Key)

	cases := map[string]struct {
		s     *MediaService
		token string
		data  []byte
		want  error
	}{
		"sem token":      {NewMediaService(&fakeRepo{}, newBucket(), &fakeLog{}), "", jpeg, media.ErrInvalidToken},
		"sem bucket":     {NewMediaService(&fakeRepo{}, nil, &fakeLog{}), "tok", jpeg, media.ErrUnavailable},
		"token errado":   {NewMediaService(&fakeRepo{}, newBucket(), &fakeLog{}), "outro", jpeg, media.ErrInvalidToken},
		"limite":         {NewMediaService(&fakeRepo{leadID: "l1", count: media.MaxPhotosPerLead}, newBucket(), &fakeLog{}), "tok", jpeg, media.ErrTooMany},
		"contagem falha": {NewMediaService(&fakeRepo{leadID: "l1", countErr: errDB}, newBucket(), &fakeLog{}), "tok", jpeg, errDB},
		"não é imagem":   {NewMediaService(&fakeRepo{leadID: "l1"}, newBucket(), &fakeLog{}), "tok", []byte("%PDF-1.4"), media.ErrNotImage},
		"grande demais":  {NewMediaService(&fakeRepo{leadID: "l1"}, newBucket(), &fakeLog{}), "tok", make([]byte, media.MaxVehiclePhotoBytes+1), media.ErrTooLarge},
	}
	for name, tc := range cases {
		_, err := tc.s.UploadLeadPhoto(ctx, tc.token, tc.data)
		assert.ErrorIs(t, err, tc.want, name)
	}

	bucket = newBucket()
	bucket.putErr = errDB
	_, err = NewMediaService(&fakeRepo{leadID: "l1"}, bucket, &fakeLog{}).UploadLeadPhoto(ctx, "tok", jpeg)
	assert.ErrorIs(t, err, errDB)

	repo, bucket = &fakeRepo{leadID: "l1", addErr: errDB}, newBucket()
	_, err = NewMediaService(repo, bucket, &fakeLog{}).UploadLeadPhoto(ctx, "tok", jpeg)
	assert.ErrorIs(t, err, errDB)
	assert.Equal(t, []string{repo.added[0].Key}, bucket.deleted, "o arquivo sai do bucket se o banco falhou")
}

func TestLeadPhoto(t *testing.T) {
	ctx := context.Background()
	repo := &fakeRepo{photo: media.Stored{Key: "leads/l1/a.jpg", ContentType: "image/jpeg"}}
	obj, err := NewMediaService(repo, newBucket(), &fakeLog{}).LeadPhoto(ctx, "u1", "l1", "p1")
	require.NoError(t, err)
	assert.Equal(t, "u1", repo.photoUser)
	body, _ := io.ReadAll(obj.Body)
	assert.Equal(t, "leads/l1/a.jpg", string(body))

	_, err = NewMediaService(&fakeRepo{}, nil, &fakeLog{}).LeadPhoto(ctx, "u1", "l1", "p1")
	assert.ErrorIs(t, err, media.ErrNotFound)
	_, err = NewMediaService(&fakeRepo{photoErr: media.ErrNotFound}, newBucket(), &fakeLog{}).LeadPhoto(ctx, "u1", "l1", "p1")
	assert.ErrorIs(t, err, media.ErrNotFound)
}
