// Package authz é a matriz de acesso (RN-01.4): cada papel tem uma lista de permissões e toda rota
// protegida exige uma delas. O app recebe a mesma lista em /auth/me e monta o menu por ela.
package authz

import "slices"

const (
	// RoleAdmin é a equipe Neo360: vê tudo e cadastra empreendimentos, empresas e profissionais.
	RoleAdmin = "admin"
	// RoleBroker é o corretor, que entra pelo cadastro aberto (RN-01.1).
	RoleBroker = "corretor"
)

const (
	// Admin é o painel administrativo inteiro.
	Admin = "admin"
	// BrokerArea é a área do corretor: os próprios links e os próprios clientes.
	BrokerArea = "area.corretor"
)

var matrix = map[string][]string{
	RoleAdmin:  {Admin},
	RoleBroker: {BrokerArea},
}

func Can(role, permission string) bool {
	return slices.Contains(matrix[role], permission)
}

// Permissions devolve as permissões do papel; papel desconhecido não tem nenhuma.
func Permissions(role string) []string {
	return append([]string{}, matrix[role]...)
}

// Valid diz se o papel existe.
func Valid(role string) bool {
	_, ok := matrix[role]
	return ok
}
