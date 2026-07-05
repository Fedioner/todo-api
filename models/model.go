package models

import "time"

type Task struct {
	Title 			string
	Description 	string
	CreatedAt		time.Time
	IsDone			bool
	CompletionTime	time.Duration
}

type Event struct{
	Command	string
	Time 	time.Time
	Error	error
}
