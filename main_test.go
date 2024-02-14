package main

import (
	"testing"
	"time"
)

func TestLimiter(t *testing.T) {
	rl := NewLimiter(10, time.Millisecond, time.Millisecond)

	for i := 0; i < 10; i++ {
		err := rl.Set("ip")
		if err != nil {
			t.Fatalf("expected no err. got %v", err)
		}
	}

	err := rl.Set("ip")
	if err == nil {
		t.Fatalf("want err. got no err")
	}
	blocked, _ := rl.Blacklisted("ip")
	if !blocked {
		t.Fatalf("want true. got %v", blocked)
	}

	// backoff err
	err = rl.Set("ip")
	if err == nil {
		t.Fatalf("want err. got no err")
	}

	time.Sleep(2 * time.Millisecond)

	err = rl.Set("ip")
	if err != nil {
		t.Fatalf("want no err. got %v", err)
	}
	blocked, _ = rl.Blacklisted("ip")
	if blocked {
		t.Fatalf("want false. got %v", blocked)
	}
}
