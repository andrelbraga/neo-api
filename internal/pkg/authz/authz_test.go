package authz

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMatrix(t *testing.T) {
	assert.True(t, Can(RoleAdmin, Admin))
	assert.False(t, Can(RoleAdmin, BrokerArea), "admin não tem área de corretor")
	assert.True(t, Can(RoleBroker, BrokerArea))
	assert.False(t, Can(RoleBroker, Admin))
	assert.False(t, Can("desconhecido", Admin))

	assert.Equal(t, []string{Admin}, Permissions(RoleAdmin))
	assert.Empty(t, Permissions("desconhecido"))

	p := Permissions(RoleBroker)
	p[0] = "alterado"
	assert.Equal(t, []string{BrokerArea}, Permissions(RoleBroker), "a cópia não altera a matriz")

	assert.True(t, Valid(RoleBroker))
	assert.False(t, Valid("cliente"))
}
