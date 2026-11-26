package media

import "errors"

var (
	ErrUnavailable    = errors.New("o envio de fotos ainda não está disponível")
	ErrNotImage       = errors.New("envie uma foto em JPG, PNG ou WebP")
	ErrTooLarge       = errors.New("a foto é grande demais")
	ErrTooMany        = errors.New("limite de fotos atingido")
	ErrInvalidToken   = errors.New("visita não encontrada; recarregue a página")
	ErrNotFound       = errors.New("foto não encontrada")
	ErrBrokerNotFound = errors.New("esta conta não é de corretor")
)
