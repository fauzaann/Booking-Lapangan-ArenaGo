// Package worker berisi proses latar belakang yang berjalan bersama API.
package worker

import (
	"context"
	"log"
	"time"

	"Booking-Lapangan/repository"
)

// BookingExpirer menjaga konsistensi status booking tanpa menunggu webhook:
// booking yang invoice-nya kedaluwarsa menjadi EXPIRED (slot dilepas), dan
// booking terkonfirmasi yang jamnya sudah lewat menjadi COMPLETED.
type BookingExpirer struct {
	uow      repository.UnitOfWork
	interval time.Duration
}

// NewBookingExpirer membuat worker dengan interval tertentu.
func NewBookingExpirer(uow repository.UnitOfWork, interval time.Duration) *BookingExpirer {
	if interval <= 0 {
		interval = time.Minute
	}
	return &BookingExpirer{uow: uow, interval: interval}
}

// Start menjalankan worker sampai context dibatalkan.
func (w *BookingExpirer) Start(ctx context.Context) {
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	log.Printf("worker: booking expirer started (every %s)", w.interval)

	for {
		select {
		case <-ctx.Done():
			log.Println("worker: booking expirer stopped")
			return
		case <-ticker.C:
			w.runOnce(ctx)
		}
	}
}

func (w *BookingExpirer) runOnce(ctx context.Context) {
	now := time.Now()

	err := w.uow.Atomic(ctx, func(r repository.Registry) error {
		bookingIDs, err := r.Payment().ExpireOverdue(ctx, now)
		if err != nil {
			return err
		}

		expired, err := r.Booking().MarkExpired(ctx, bookingIDs)
		if err != nil {
			return err
		}
		if expired > 0 {
			log.Printf("worker: %d booking(s) marked as EXPIRED", expired)
		}

		completed, err := r.Booking().CompleteFinished(ctx, now)
		if err != nil {
			return err
		}
		if completed > 0 {
			log.Printf("worker: %d booking(s) marked as COMPLETED", completed)
		}
		return nil
	})
	if err != nil {
		log.Printf("worker: booking expirer failed: %v", err)
	}
}
