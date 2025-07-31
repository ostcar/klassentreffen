package model

// New returns an initialized model.
func New() Model {
	return Model{
		Participant: make(map[string]Participant),
	}
}

type Model struct {
	Participant map[string]Participant
}

type Participant struct {
	Mail     string `json:"mail"`
	Name     string `json:"name"`
	OldName  string `json:"old_name"`
	Info     bool   `json:"info"`
	Attend   bool   `json:"attend"`
	Public   bool   `json:"public"`
	Admin    bool   `json:"admin"`
	Verified bool   `json:"verified"`
}
