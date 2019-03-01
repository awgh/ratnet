package api

import (
	"testing"
)

func Test_RoundTrip_1(t *testing.T) {
	var call RemoteCall
	call.Action = ActionFromUint16(APIID)
	b := RemoteCallToBytes(&call)
	recall, err := RemoteCallFromBytes(b)
	if err != nil {
		t.Fatal(err)
	}
	if call.Action != recall.Action {
		t.Fatal("Before and After Actions do not match")
	}
}

func Test_RoundTrip_2(t *testing.T) {
	var call RemoteCall
	call.Action = ActionFromUint16(APIAddProfile)
	var x uint64
	x = 1234
	y := "abcd1234"
	z := []byte{1, 2, 3, 4, 5, 6}
	call.Args = []interface{}{}
	call.Args = append(call.Args, x, y, z)
	t.Log(call)
	b := RemoteCallToBytes(&call)
	t.Log(b)
	recall, err := RemoteCallFromBytes(b)
	if err != nil {
		t.Fatal(err)
	}
	t.Log(recall)
	if call.Action != recall.Action {
		t.Log(call, recall)

		t.Fatal("Before and After Actions do not match")
	}
}
