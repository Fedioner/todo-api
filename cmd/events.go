package cmd

import (
	"github.com/Fedioner/todo-api/models"
	"github.com/Fedioner/todo-api/storage"
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