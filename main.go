package main

import (
	"bufio"
	"github.com/Fedioner/todo-api/cmd"
	"github.com/Fedioner/todo-api/storage"
	"fmt"
	"os"
)

func main(){

	scanner := bufio.NewScanner(os.Stdin)
	var store storage.Storage = storage.NewInMemory()
	var events storage.EventLog = storage.NewInLog()

	scanner.Split(bufio.ScanWords)

	for {
		ok := scanner.Scan()

		if !ok {
			break
		}

		switch scanner.Text() {
		case "help":
			events.Log(cmd.HelpHandler())
		case "add": 
			event, err := cmd.AddHandler(store, scanner)
			events.Log(event)
			if err != nil {
				fmt.Println(err)
			}
		case "list":
			event := cmd.ListHandler(store)
			events.Log(event)	
		case "del": 
			event, err := cmd.DelHandler(store, scanner)
			events.Log(event)
			if err != nil {
				fmt.Println(err)
			}	
		case "done":
			event, err := cmd.DoneHandler(store, scanner)
			events.Log(event)
			if err != nil {
				fmt.Println(err)
			}	
		case "events":
			event := cmd.EventsHandler(events)
			events.Log(event)
		case "exit":
			return;
		default:
			fmt.Println("Неизвестная команда!")
		}
	}

}