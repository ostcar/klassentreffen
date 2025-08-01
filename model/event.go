package model

import (
	"time"

	"github.com/ostcar/sticky"
)

type Event = sticky.Event[Model]

func GetEvent(eventType string) Event {
	switch eventType {
	case eventParticipantSave{}.Name():
		return &eventParticipantSave{}
	default:
		return nil
	}
}

// eventParticipantSave creates or updates a participant
type eventParticipantSave struct {
	Participant Participant `json:"participant"`
	Mail        string      `json:"mail"`
}

func (e eventParticipantSave) Name() string {
	return "participant-save"
}

func (e eventParticipantSave) Validate(model Model) error {
	return nil
}

func (e eventParticipantSave) Execute(model Model, time time.Time) Model {
	if e.Mail != e.Participant.Mail {
		delete(model.Participant, e.Mail)
	}

	model.Participant[e.Participant.Mail] = Participant{
		Mail:     e.Participant.Mail,
		Name:     e.Participant.Name,
		OldName:  e.Participant.OldName,
		Info:     e.Participant.Info,
		Attend:   e.Participant.Attend,
		Public:   e.Participant.Public,
		Admin:    e.Participant.Admin,
		Verified: e.Participant.Verified,
	}
	return model
}
