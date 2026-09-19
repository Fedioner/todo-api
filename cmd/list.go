package cmd

import (
	"github.com/Fedioner/todo-api/models"
	"github.com/Fedioner/todo-api/storage"
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
