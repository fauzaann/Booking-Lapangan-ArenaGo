package models

// FieldType adalah jenis olahraga yang didukung lapangan.
type FieldType string

const (
	FieldTypeFutsal     FieldType = "FUTSAL"
	FieldTypeBadminton  FieldType = "BADMINTON"
	FieldTypeBasket     FieldType = "BASKET"
	FieldTypeTennis     FieldType = "TENNIS"
	FieldTypeMiniSoccer FieldType = "MINI_SOCCER"
)

// FieldTypes mengembalikan seluruh jenis lapangan yang valid.
func FieldTypes() []FieldType {
	return []FieldType{
		FieldTypeFutsal,
		FieldTypeBadminton,
		FieldTypeBasket,
		FieldTypeTennis,
		FieldTypeMiniSoccer,
	}
}

// Valid memeriksa apakah jenis lapangan dikenal.
func (f FieldType) Valid() bool {
	for _, t := range FieldTypes() {
		if t == f {
			return true
		}
	}
	return false
}
