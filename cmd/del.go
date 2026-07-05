package cmd

import (
	"bufio"
	"education/models"
	"education/storage"
	"fmt"
	"time"
)

func DelHandler(store storage.Storage, scanner *bufio.Scanner) (models.Event, error) {

	scanner.Scan()
	title := scanner.Text()

	if err := store.Delete(title); err != nil {
		fmt.Println(err)
		return models.Event{
			Command: "delete",
			Time: time.Now(),
			Error: err,
		},
		err
	}

	fmt.Printf("Задача %s удалена\n", title)

	return models.Event{
			Command: "delete",
			Time: time.Now(),
			Error: nil,
		},
		nil
}