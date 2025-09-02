package domain

import (
	"net/http"
)

type RouterService interface {
	Run(addr string) error
	Handler() http.Handler
}
