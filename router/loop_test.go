package router

import (
	"testing"

	"github.com/awgh/bencrypt/bc"
)

func Test_Loop_OneMessage_1(t *testing.T) {

	recentBuffer := newRecentBuffer()
	b, err := bc.GenerateRandomBytes(nonceSize)
	if err != nil {
		t.Fatal(err)
	}
	everSeen := false
	for i := 0; i < 1000; i++ {
		seen := recentBuffer.SeenRecently(b)
		t.Log("seen? ", i, seen)
		if seen {
			everSeen = true
		}
	}
	if !everSeen {
		t.Fatal("SeenRecently never returned true on one-message loop test")
	}
}

func Test_Loop_Random_1(t *testing.T) {

	recentBuffer := newRecentBuffer()
	for i := 0; i < 100000; i++ {
		b, err := bc.GenerateRandomBytes(nonceSize)
		if err != nil {
			t.Fatal(err)
		}
		seen := recentBuffer.SeenRecently(b)
		//t.Log("seen? ", i, seen)
		if seen {
			t.Fatal("SeenRecently returned true on random loop test on iteration:", i)
		}
	}
}

func Test_Loop_Fixed_1(t *testing.T) {

	recentBuffer := newRecentBuffer()
	var sendBuffers [][]byte
	for i := 0; i < 1000; i++ {
		b, err := bc.GenerateRandomBytes(nonceSize)
		if err != nil {
			t.Fatal(err)
		}
		sendBuffers = append(sendBuffers, b)
	}
	everSeen := false
	for i := 0; i < 5000; i++ {
		seen := recentBuffer.SeenRecently(sendBuffers[i%len(sendBuffers)])
		t.Log("seen? ", i, i%len(sendBuffers), seen)
		if seen {
			everSeen = true
		}
	}
	if !everSeen {
		t.Fatal("SeenRecently never returned true on fixed loop test")
	}
}
