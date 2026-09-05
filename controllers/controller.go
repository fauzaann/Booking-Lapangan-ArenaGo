package controllers


// Actor adalah identitas yang sudah terautentikasi
type Actor struct {
	ID       uint   `json:"id"`
	Username string `json:"username"`
	Role     string `json:"role"`
}

// IsAdmin memeriksa apakah aktor memiliki peran admin
func (a *Actor) IsAdmin() bool {
	return a.Role == "ADMIN"
}