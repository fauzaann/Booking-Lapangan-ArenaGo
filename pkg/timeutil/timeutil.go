// Package timeutil berisi helper parsing dan perhitungan waktu booking.
// Jam disimpan sebagai string "HH:MM" 24 jam agar perbandingan leksikografis
// di database identik dengan perbandingan kronologis.
package timeutil

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

const (
	ClockLayout = "15:04"
	DateLayout  = "2006-01-02"
)

// ParseDate memparsing "YYYY-MM-DD" menjadi tanggal (tanpa jam) di UTC.
func ParseDate(value string) (time.Time, error) {
	t, err := time.Parse(DateLayout, strings.TrimSpace(value))
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid date format, expected YYYY-MM-DD")
	}
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC), nil
}

// NormalizeDate memotong komponen jam dari sebuah time.Time.
func NormalizeDate(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
}

// Today mengembalikan tanggal hari ini (lokal server) dalam bentuk ternormalisasi.
func Today() time.Time {
	return NormalizeDate(time.Now())
}

// ValidClock memeriksa apakah string berformat "HH:MM" yang valid.
func ValidClock(value string) bool {
	_, err := Minutes(value)
	return err == nil
}

// Minutes mengubah "HH:MM" menjadi jumlah menit sejak tengah malam.
func Minutes(value string) (int, error) {
	parts := strings.Split(strings.TrimSpace(value), ":")
	if len(parts) != 2 || len(parts[0]) != 2 || len(parts[1]) != 2 {
		return 0, fmt.Errorf("invalid time format, expected HH:MM")
	}
	hour, err := strconv.Atoi(parts[0])
	if err != nil || hour < 0 || hour > 23 {
		return 0, fmt.Errorf("invalid hour value")
	}
	minute, err := strconv.Atoi(parts[1])
	if err != nil || minute < 0 || minute > 59 {
		return 0, fmt.Errorf("invalid minute value")
	}
	return hour*60 + minute, nil
}

// FormatMinutes mengubah menit sejak tengah malam menjadi "HH:MM".
func FormatMinutes(total int) string {
	return fmt.Sprintf("%02d:%02d", total/60, total%60)
}

// DurationHours menghitung durasi dalam jam penuh antara dua jam.
// Booking dibatasi pada kelipatan 1 jam agar cocok dengan model slot per jam.
func DurationHours(start, end string) (int, error) {
	startMin, err := Minutes(start)
	if err != nil {
		return 0, fmt.Errorf("start_time: %w", err)
	}
	endMin, err := Minutes(end)
	if err != nil {
		return 0, fmt.Errorf("end_time: %w", err)
	}
	if endMin <= startMin {
		return 0, fmt.Errorf("end_time must be greater than start_time")
	}
	diff := endMin - startMin
	if diff%60 != 0 {
		return 0, fmt.Errorf("booking duration must be in full hours")
	}
	return diff / 60, nil
}

// HourlySlots memecah rentang waktu menjadi slot per jam.
func HourlySlots(start, end string) [][2]string {
	startMin, err := Minutes(start)
	if err != nil {
		return nil
	}
	endMin, err := Minutes(end)
	if err != nil || endMin <= startMin {
		return nil
	}
	slots := make([][2]string, 0, (endMin-startMin)/60)
	for cursor := startMin; cursor+60 <= endMin; cursor += 60 {
		slots = append(slots, [2]string{FormatMinutes(cursor), FormatMinutes(cursor + 60)})
	}
	return slots
}

// Overlap memeriksa dua rentang waktu saling tumpang tindih.
// Rentang dianggap half-open [start, end) sehingga 10:00-11:00 dan
// 11:00-12:00 TIDAK dianggap overlap.
func Overlap(aStart, aEnd, bStart, bEnd string) bool {
	as, err1 := Minutes(aStart)
	ae, err2 := Minutes(aEnd)
	bs, err3 := Minutes(bStart)
	be, err4 := Minutes(bEnd)
	if err1 != nil || err2 != nil || err3 != nil || err4 != nil {
		return false
	}
	return as < be && ae > bs
}

// Within memeriksa rentang [start,end) berada di dalam [outerStart,outerEnd).
func Within(start, end, outerStart, outerEnd string) bool {
	s, err1 := Minutes(start)
	e, err2 := Minutes(end)
	os, err3 := Minutes(outerStart)
	oe, err4 := Minutes(outerEnd)
	if err1 != nil || err2 != nil || err3 != nil || err4 != nil {
		return false
	}
	return s >= os && e <= oe
}

// CombineDateTime menggabungkan tanggal booking dan jam "HH:MM" menjadi
// satu time.Time pada zona waktu lokal server.
func CombineDateTime(date time.Time, clock string) (time.Time, error) {
	minutes, err := Minutes(clock)
	if err != nil {
		return time.Time{}, err
	}
	return time.Date(date.Year(), date.Month(), date.Day(), minutes/60, minutes%60, 0, 0, time.Local), nil
}
