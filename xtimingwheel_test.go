package xtool

import (
	"fmt"
	"testing"
	"time"
)

// go test -v xtimingwheel_test.go xtimingwheel.go
func TestDefaultTimingWheel(t *testing.T) {
	t.Log("start")
	tw, err := DefaultTimingWheel()
	if err != nil {
		t.Log(err)
	}
	err = tw.AddTask("task_test", func(key string) {
		fmt.Println(key, "xiaoxucode")
	}, 15*time.Second, 2)
	if err != nil {
		t.Log(err)
	}
	fmt.Printf("%+v", tw)
	tw.Stop()

	time.Sleep(20 * time.Second)
	err = tw.RemoveTask("task_test")
	if err != nil {
		t.Log(err)
	}
	fmt.Printf("%+v", tw)
	time.Sleep(100 * time.Second)
	t.Log("end")
}
