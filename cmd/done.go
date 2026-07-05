package cmd

import (
	"bufio"
	"education/models"
	"education/storage"
	"time"
)

func DoneHandler(store storage.Storage, scanner *bufio.Scanner) (models.Event, error) {

	scanner.Scan()
	title := scanner.Text()

	task, err := store.Get(title);
	if err != nil {
		return models.Event{
			Command: "done",
			Time: time.Now(),
			Error: err,
		},
		err
	}
	
	task.IsDone = true
	task.CompletionTime = time.Since(task.CreatedAt)
	
	store.Update(task)
		
	return models.Event{
				Command: "done",
				Time: time.Now(),
				Error: nil,
			},
		nil
}