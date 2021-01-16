package api

import (
	"strings"
	"testing"
)

func Test_ArgsRoundTrip_1(t *testing.T) {
	// Nil
	argsBytes := ArgsToBytes(nil)
	args, err := ArgsFromBytes(argsBytes)
	if err != nil {
		t.Fatal(err)
	}
	// int64
	argsBytes = ArgsToBytes([]interface{}{int64(4096)})
	args, err = ArgsFromBytes(argsBytes)
	if err != nil {
		t.Fatal(err)
	}

	t.Logf("%+v\n", args)
}

func Test_RoundTrip_1(t *testing.T) {
	var call RemoteCall
	call.Action = ID
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
	call.Action = AddProfile
	var x uint64
	x = 1234
	y := "abcd1234"
	z := []byte{1, 2, 3, 4, 5, 6}
	call.Args = append(call.Args, x, y, z)
	t.Logf("%+v", call)
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

func Test_ResponseRoundTrip_1(t *testing.T) {
	var resp RemoteResponse
	resp.Error = "This is my error.  There are many like it, but this one is mine."
	resp.Value = []byte{1, 2, 3, 4, 5, 6, 7, 8}
	t.Log(resp)
	b := RemoteResponseToBytes(&resp)
	t.Log(b)

	reresp, err := RemoteResponseFromBytes(b)
	if err != nil {
		t.Fatal(err)
	}
	t.Log(reresp)

	if 0 != strings.Compare(resp.Error, reresp.Error) {
		t.Log(resp.Error)
		t.Log(reresp.Error)

		t.Fatal("Before and After Errors do not match")
	}
}

func Test_ResponseRoundTrip_2(t *testing.T) {
	var resp RemoteResponse
	resp.Error = ""
	resp.Value = []byte{}
	t.Log(resp)
	b := RemoteResponseToBytes(&resp)
	t.Log(b)

	reresp, err := RemoteResponseFromBytes(b)
	if err != nil {
		t.Fatal(err)
	}
	t.Log(reresp)

	if 0 != strings.Compare(resp.Error, reresp.Error) {
		t.Log(resp.Error)
		t.Log(reresp.Error)

		t.Fatal("Before and After Errors do not match")
	}
}
