// Package dto berisi kontrak request/response API. Model database tidak
// pernah dikirim langsung ke client agar field sensitif tidak bocor.
package dto

// PaginationQuery adalah query string pagination standar.
type PaginationQuery struct {
	Page  int `form:"page"`
	Limit int `form:"limit"`
}

// Normalize memberi nilai default pada pagination.
func (p PaginationQuery) Normalize() PaginationQuery {
	if p.Page < 1 {
		p.Page = 1
	}
	if p.Limit < 1 {
		p.Limit = 10
	}
	if p.Limit > 100 {
		p.Limit = 100
	}
	return p
}
