//go:build integration

package repository

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"neo360-api/internal/domain/media"
	"neo360-api/internal/testutil"
)

func TestMediaRepository_Integration(t *testing.T) {
	database := testutil.MigratedDB(t, "../../../../sql/migrations")
	repo := NewMediaRepository(database)
	ctx := context.Background()

	var devID, userID, brokerID, otherUser, leadID, mergedID string
	require.NoError(t, database.QueryRowContext(ctx, `SELECT id FROM developments WHERE slug = 'boa-vista-residencial'`).Scan(&devID))
	newBroker := func(email, code string, user, broker *string) {
		require.NoError(t, database.QueryRowContext(ctx,
			`INSERT INTO users (name, email, password_hash, role) VALUES ('X', $1, 'h', 'corretor') RETURNING id`, email).Scan(user))
		require.NoError(t, database.QueryRowContext(ctx,
			`INSERT INTO brokers (user_id, first_name, last_name, phone, code) VALUES ($1, 'Fabiano', 'Souza', '54999990000', $2) RETURNING id`,
			*user, code).Scan(broker))
	}
	var otherBroker string
	newBroker("fab@x.com", "FAB-0001", &userID, &brokerID)
	newBroker("ana@x.com", "ANA-0001", &otherUser, &otherBroker)

	// Foto do corretor: troca devolvendo a anterior; só a atual de corretor ativo está em uso.
	old, err := repo.SetBrokerPhoto(ctx, userID, "brokers/a.jpg")
	require.NoError(t, err)
	assert.Empty(t, old)
	old, err = repo.SetBrokerPhoto(ctx, userID, "brokers/b.jpg")
	require.NoError(t, err)
	assert.Equal(t, "brokers/a.jpg", old)
	inUse, err := repo.BrokerPhotoInUse(ctx, "brokers/b.jpg")
	require.NoError(t, err)
	assert.True(t, inUse)
	inUse, _ = repo.BrokerPhotoInUse(ctx, "brokers/a.jpg")
	assert.False(t, inUse, "a foto trocada não sai mais")
	_, err = database.ExecContext(ctx, `UPDATE users SET active = false WHERE id = $1`, userID)
	require.NoError(t, err)
	inUse, _ = repo.BrokerPhotoInUse(ctx, "brokers/b.jpg")
	assert.False(t, inUse, "corretor desativado não mostra a foto")
	_, err = database.ExecContext(ctx, `UPDATE users SET active = true WHERE id = $1`, userID)
	require.NoError(t, err)
	for _, u := range []string{"00000000-0000-0000-0000-000000000000", "nao-e-uuid"} {
		_, err = repo.SetBrokerPhoto(ctx, u, "brokers/c.jpg")
		assert.ErrorIs(t, err, media.ErrBrokerNotFound, u)
	}

	// Cliente pelo token; o token de um cadastro juntado vale pelo que ficou (ADR-0016).
	require.NoError(t, database.QueryRowContext(ctx,
		`INSERT INTO leads (development_id, broker_id, token_hash) VALUES ($1, $2, 'h-lead') RETURNING id`, devID, brokerID).Scan(&leadID))
	require.NoError(t, database.QueryRowContext(ctx,
		`INSERT INTO leads (development_id, broker_id, token_hash, merged_into) VALUES ($1, $2, 'h-merged', $3) RETURNING id`, devID, brokerID, leadID).Scan(&mergedID))
	got, err := repo.LeadByToken(ctx, "h-lead")
	require.NoError(t, err)
	assert.Equal(t, leadID, got)
	got, err = repo.LeadByToken(ctx, "h-merged")
	require.NoError(t, err)
	assert.Equal(t, leadID, got)
	_, err = repo.LeadByToken(ctx, "h-nao-existe")
	assert.ErrorIs(t, err, media.ErrInvalidToken)

	// Fotos de veículo: contam por cliente e só aparecem ligadas a uma proposta.
	p, err := repo.AddLeadPhoto(ctx, leadID, media.Stored{Key: "leads/l/a.jpg", ContentType: "image/jpeg"}, 1234)
	require.NoError(t, err)
	assert.Equal(t, media.Photo{ID: p.ID, ContentType: "image/jpeg", Size: 1234}, p)
	n, err := repo.CountLeadPhotos(ctx, leadID)
	require.NoError(t, err)
	assert.Equal(t, 1, n)
	_, err = repo.LeadPhoto(ctx, userID, leadID, p.ID)
	assert.ErrorIs(t, err, media.ErrNotFound, "foto ainda sem proposta não aparece")

	var proposalID string
	require.NoError(t, database.QueryRowContext(ctx,
		`INSERT INTO proposals (lead_id, author, condition) VALUES ($1, 'cliente', '{}') RETURNING id`, leadID).Scan(&proposalID))
	_, err = database.ExecContext(ctx, `UPDATE lead_photos SET proposal_id = $1 WHERE id = $2`, proposalID, p.ID)
	require.NoError(t, err)
	s, err := repo.LeadPhoto(ctx, userID, leadID, p.ID)
	require.NoError(t, err)
	assert.Equal(t, media.Stored{Key: "leads/l/a.jpg", ContentType: "image/jpeg"}, s)
	_, err = repo.LeadPhoto(ctx, "", leadID, p.ID)
	assert.NoError(t, err, "o admin vê")
	_, err = repo.LeadPhoto(ctx, otherUser, leadID, p.ID)
	assert.ErrorIs(t, err, media.ErrNotFound, "outro corretor não vê")
	_, err = repo.LeadPhoto(ctx, userID, mergedID, p.ID)
	assert.ErrorIs(t, err, media.ErrNotFound, "o id do cliente precisa ser o do cadastro que ficou")
	// Foto enviada por um cadastro juntado ao cliente aparece no cliente que ficou (ADR-0016).
	pm, err := repo.AddLeadPhoto(ctx, mergedID, media.Stored{Key: "leads/m/b.jpg", ContentType: "image/jpeg"}, 10)
	require.NoError(t, err)
	_, err = database.ExecContext(ctx, `UPDATE lead_photos SET proposal_id = $1 WHERE id = $2`, proposalID, pm.ID)
	require.NoError(t, err)
	s, err = repo.LeadPhoto(ctx, userID, leadID, pm.ID)
	require.NoError(t, err)
	assert.Equal(t, "leads/m/b.jpg", s.Key)
	_, err = repo.LeadPhoto(ctx, userID, leadID, "nao-e-uuid")
	assert.ErrorIs(t, err, media.ErrNotFound)
}
