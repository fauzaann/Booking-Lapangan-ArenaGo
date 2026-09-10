package dto

// PaginationQuery adalah query untuk pagination
type PaginationQuery struct {
	Page  int `form:"page"`
	Limit int `form:"limit"`
}

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
