package cmd

import (
	"github.com/Fedioner/todo-api/models"
	"fmt"
	"time"
)

const helpText = 
	`
	help — команды
	add — добавить
	list — список
	del — удалить
	done — выполнить
	exit — выход
	`

func HelpHandler() models.Event {
	fmt.Println(helpText)
	return models.Event{
		Command: "help",
		Time: time.Now(),
		Error: nil,
	}
}