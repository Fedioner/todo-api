package storage

import (
	"github.com/Fedioner/todo-api/models"
	"fmt"
)

type Storage interface {
	Add(task models.Task) error
	Get(title string) (models.Task, error)
	Delete(title string) error
	List() []models.Task
	Update (task models.Task) error
}

type EventLog interface {
	Log(models.Event) 
	GetAll() []models.Event
	All(yield func(models.Event) bool)
}

type InLog struct {
	events []models.Event 
}

func NewInLog () *InLog {
	return &InLog{events: make([]models.Event, 0)}
}

func (l *InLog) Log(event models.Event) {
	l.events = append(l.events, event)
}

func (l *InLog) GetAll() []models.Event {
	return l.events
}

func (l *InLog) All(yield func(models.Event) bool) {
	for _, val := range l.events {
		if !yield(val){
			return
		}
	}
}
type InMemory struct {
	data map[string]models.Task
}

func NewInMemory() *InMemory{
	return &InMemory{data: make(map[string]models.Task)}
}

func (s *InMemory) Add(task models.Task) error{
	
	if _, ok := s.data[task.Title]; ok {
		return fmt.Errorf("Задача %v уже сущствует", task)
	}

	s.data[task.Title] = task
	return nil
}

func (s *InMemory) Get(title string) (models.Task, error) {
	
	task, ok := s.data[title]
	if !ok {
		return models.Task{}, fmt.Errorf("Задача %s не найдена", title)
	}

	return task, nil
}

func (s *InMemory) Delete(title string) error {
	if _, ok := s.data[title]; !ok {
		return fmt.Errorf("Задача %s не найдена", title)
	}

	delete(s.data, title)
	return nil
}

func (s *InMemory) List() []models.Task {
	
	list := make([]models.Task, 0, len(s.data))

	for _, value := range s.data {
		list = append(list, value)
	}

	return list
}

func (s *InMemory) Update(task models.Task) error {
    if _, ok := s.data[task.Title]; !ok {
        return fmt.Errorf("задача %s не найдена", task.Title)
    }
    s.data[task.Title] = task  
    return nil
}
