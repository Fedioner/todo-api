package cmd

import (
	"education/models"
	"education/storage"
	"time"

	"github.com/k0kubun/pp"
)

func ListHandler(store storage.Storage) (models.Event) {
	for _, task := range store.List(){
		pp.Println(task)
	}
	return models.Event{
			Command: "list",
			Time: time.Now(),
			Error: nil,
		}
}
