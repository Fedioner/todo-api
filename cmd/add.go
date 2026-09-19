package cmd

import (
	"bufio"
	"github.com/Fedioner/todo-api/models"
	"github.com/Fedioner/todo-api/storage"
	"errors"
	"fmt"
	"strings"
	"time"
)

func AddHandler(store storage.Storage, scanner *bufio.Scanner) (models.Event, error){

	if !scanner.Scan() {
		return models.Event{
			Command: "add",
			Time: time.Now(),
			Error: errors.New("ошибка чтения с клавиатуры"),
		}, errors.New("ошибка чтения с клавиатуры")
	}

	title := scanner.Text()

	if strings.TrimSpace(title) == "" {
		return models.Event{
			Command: "add",
			Time: time.Now(),
			Error: errors.New("Заголовок не может быть пустым"),
		}, errors.New("Заголовок не может быть пустым")
	}

	scanner.Scan()

	description := strings.TrimSpace(scanner.Text())

	err := store.Add(models.Task{
		Title: title, 
		Description: description, 
		CreatedAt: time.Now(), 
		IsDone: false,
	})

	if err != nil {	
		return models.Event{
			Command: "add",
			Time: time.Now(),
			Error: err,
		},
		err
	}

	fmt.Println("Запись успешно создана!!!")

	return models.Event{
			Command: "add",
			Time: time.Now(),
			Error: nil,
		},
		err
}