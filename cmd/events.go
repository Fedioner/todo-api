package cmd

import (
	"education/models"
	"education/storage"
	"time"

	"github.com/k0kubun/pp"
)


func EventsHandler(events storage.EventLog) (models.Event){
	for event := range events.All{
		pp.Println(event)
	}

	return models.Event{
		Command: "events",
		Time: time.Now(),
		Error: nil,
	}
}