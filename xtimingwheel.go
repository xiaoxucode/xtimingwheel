package xtool

import (
	"container/list"
	"errors"
	"sync"
	"time"
)

type Job func(key string)

type Task struct {
	key       string
	job       Job
	executeAt time.Duration
	times     int
	slot      int
	circle    int
}

type XTimeWheel struct {
	interval     time.Duration
	ticker       *time.Ticker
	currentSlot  int
	slotNum      int
	slots        []*list.List
	stopCh       chan struct{}
	removeTaskCh chan string
	addTaskCh    chan *Task
	taskRecords  sync.Map
	mux          sync.Mutex
	isRun        bool
}

func DefaultTimingWheel() (*XTimeWheel, error) {
	tw, _ := NewXTimingWheel(time.Second, 12)
	return tw, nil
}

func NewXTimingWheel(interval time.Duration, slotNum int) (*XTimeWheel, error) {
	if interval <= 0 {
		return nil, errors.New("minimum interval need one second")
	}
	if slotNum <= 0 {
		return nil, errors.New("minimum slotNum need greater than zero")
	}
	t := &XTimeWheel{
		interval:     interval,
		currentSlot:  0,
		slotNum:      slotNum,
		slots:        make([]*list.List, slotNum),
		stopCh:       make(chan struct{}),
		removeTaskCh: make(chan string),
		addTaskCh:    make(chan *Task),
		isRun:        false,
	}
	t.start()
	return t, nil
}

func (t *XTimeWheel) Stop() {
	if t.isRun {
		t.mux.Lock()
		t.isRun = false
		t.ticker.Stop()
		t.mux.Unlock()
		close(t.stopCh)
	}
}

func (t *XTimeWheel) AddTask(key string, job Job, executeAt time.Duration, times int) error {
	if key == "" {
		return errors.New("key is empty")
	}
	if executeAt < t.interval {
		return errors.New("key is empty")
	}
	_, ok := t.taskRecords.Load(key)
	if ok {
		return errors.New("key of job already exists")
	}
	task := &Task{
		key:       key,
		job:       job,
		times:     times,
		executeAt: executeAt,
	}
	t.addTaskCh <- task
	return nil
}

func (t *XTimeWheel) RemoveTask(key string) error {
	if key == "" {
		return errors.New("key is empty")
	}
	t.removeTaskCh <- key
	return nil
}

func (t *XTimeWheel) start() {
	if !t.isRun {
		for i := 0; i < t.slotNum; i++ {
			t.slots[i] = list.New()
		}
		t.ticker = time.NewTicker(t.interval)
		t.mux.Lock()
		t.isRun = true
		go t.run()
		t.mux.Unlock()
	}
}

func (t *XTimeWheel) run() {
	for {
		select {
		case <-t.stopCh:
			return
		case task := <-t.addTaskCh:
			t.addTask(task)
		case key := <-t.removeTaskCh:
			t.removeTask(key)
		case <-t.ticker.C:
			t.execute()
		}
	}
}

func (t *XTimeWheel) addTask(task *Task) {
	slot, circle := t.calSlotAndCircle(task.executeAt)
	task.slot = slot
	task.circle = circle
	ele := t.slots[slot].PushBack(task)
	t.taskRecords.Store(task.key, ele)
}

func (t *XTimeWheel) removeTask(key string) {
	taskRec, ok := t.taskRecords.Load(key)
	if !ok {
		return
	}
	ele := taskRec.(*list.Element)
	task, _ := ele.Value.(*Task)
	t.slots[task.slot].Remove(ele)
	t.taskRecords.Delete(key)
}

func (t *XTimeWheel) execute() {
	taskList := t.slots[t.currentSlot]
	if taskList != nil {
		for ele := taskList.Front(); ele != nil; {
			taskEle, _ := ele.Value.(*Task)
			if taskEle.circle > 0 {
				taskEle.circle--
				ele = ele.Next()
				continue
			}
			go taskEle.job(taskEle.key)
			t.taskRecords.Delete(taskEle.key)
			taskList.Remove(ele)

			if taskEle.times-1 > 0 {
				taskEle.times--
				t.addTask(taskEle)
			}
			if taskEle.times == -1 {
				t.addTask(taskEle)
			}
			ele = ele.Next()
		}
	}
	t.incrCurrentSlot()
}

func (t *XTimeWheel) incrCurrentSlot() {
	t.currentSlot = (t.currentSlot + 1) % len(t.slots)
}

func (t *XTimeWheel) calSlotAndCircle(executeAt time.Duration) (slot, circle int) {
	delay := int(executeAt.Seconds())
	circleTime := len(t.slots) * int(t.interval.Seconds())
	circle = delay / circleTime
	steps := delay / int(t.interval.Seconds())
	slot = (t.currentSlot + steps) % len(t.slots)
	return
}
