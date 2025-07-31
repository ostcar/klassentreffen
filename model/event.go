package model

import (
	"time"

	"github.com/ostcar/sticky"
)

type Event = sticky.Event[Model]

func GetEvent(eventType string) Event {
	switch eventType {
	case eventParticipantUpdate{}.Name():
		return &eventParticipantUpdate{}
	default:
		return nil
	}
}

// eventParticipantUpdate creates or updates a participant
type eventParticipantUpdate struct {
	Mail            string `json:"mail"`
	ParticipantName string `json:"name"`
	OldName         string `json:"old_name"`
	Info            bool   `json:"info"`
	Attend          bool   `json:"attend"`
	Public          bool   `json:"public"`
	Admin           bool   `json:"admin"`
	Verified        bool   `json:"verified"`
}

func (e eventParticipantUpdate) Name() string {
	return "participant-update"
}

func (e eventParticipantUpdate) Validate(model Model) error {
	return nil
}

func (e eventParticipantUpdate) Execute(model Model, time time.Time) Model {
	model.Participant[e.Mail] = Participant{
		Mail:     e.Mail,
		Name:     e.ParticipantName,
		OldName:  e.OldName,
		Info:     e.Info,
		Attend:   e.Attend,
		Public:   e.Public,
		Admin:    e.Admin,
		Verified: e.Verified,
	}
	return model
}
