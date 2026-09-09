package repository

import (
	"context"

	"gorm.io/gorm"
)

// Registry menyediakan seluruh repository dalam satu tempat.
// Saat dipakai di dalam transaksi, seluruh repository yang dikembalikan
// terikat pada *gorm.DB transaksi yang sama.
type Registry interface {
	User() UserRepository
	Field() FieldRepository
	Schedule() ScheduleRepository
}

// UnitOfWork adalah Registry yang mampu menjalankan beberapa operasi
// dalam satu transaksi database (atomic).
type UnitOfWork interface {
	Registry
	Atomic(ctx context.Context, fn func(r Registry) error) error
}

type registry struct {
	db *gorm.DB
}

// NewUnitOfWork membuat UnitOfWork berbasis GORM.
func NewUnitOfWork(db *gorm.DB) UnitOfWork {
	return &registry{db: db}
}

func (r *registry) User() UserRepository         { return NewUserRepository(r.db) }
func (r *registry) Field() FieldRepository       { return NewFieldRepository(r.db) }
func (r *registry) Schedule() ScheduleRepository { return NewScheduleRepository(r.db) }

// Atomic menjalankan fn di dalam satu transaksi. Jika fn mengembalikan error,
// transaksi otomatis di-rollback; jika nil, transaksi di-commit.
func (r *registry) Atomic(ctx context.Context, fn func(Registry) error) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return fn(&registry{db: tx})
	})
}

type ListParams struct {
	Page  int
	Limit int
}

// Normalize memberi nilai default dan batas aman untuk pagination.
func (p ListParams) Normalize() ListParams {
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

// Offset menghitung offset SQL dari halaman saat ini.
func (p ListParams) Offset() int {
	normalized := p.Normalize()
	return (normalized.Page - 1) * normalized.Limit
}
