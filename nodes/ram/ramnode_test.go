package ram

import (
	"bytes"
	"log"
	"os"
	"testing"

	"github.com/awgh/bencrypt/ecc"
	"github.com/awgh/ratnet/api"
)

var node *Node

func Test_init(t *testing.T) {
	node = New(new(ecc.KeyPair), new(ecc.KeyPair))
	os.Mkdir("tmp", os.FileMode(int(0755)))
	node.FlushOutbox(0)
	if err := node.routingKey.FromB64(pubprivkeyb64Ecc); err != nil {
		log.Fatal(err)
	}
	if err := node.Start(); err != nil {
		log.Fatal(err)
	}
}

func Test_apicall_ID_1(t *testing.T) {
	result, err := node.ID()
	if err != nil {
		t.Error(err.Error())
	}
	t.Log("API ID RESULT: ", result)
}

func Test_apicall_AddContact_1(t *testing.T) {
	p1 := pubkeyb64Ecc
	if err := node.AddContact("destname1", p1); err != nil {
		t.Error(err.Error())
	}
	contact, err := node.GetContact("destname1")
	if err != nil {
		t.Error(err.Error())
	}
	t.Logf("API AddContact RESULT: %+v\n", contact)
	if contact == nil {
		t.Fail()
	}
}

func Test_apicall_Send_1(t *testing.T) {
	err := node.Send("destname1", []byte(pubkeyb64))
	if err != nil {
		t.Error(err.Error())
	}
	t.Log("API Send RESULT: OK")
}

func Test_apicall_Pickup_1(t *testing.T) {
	rpk, err := node.ID()
	if err != nil {
		t.Error(err.Error())
	}
	_, err = node.Pickup(rpk, 0, 0)
	if err != nil {
		t.Error(err.Error())
	}
	t.Log("API Pickup RESULT: OK")
}

func Test_apicall_Channels_1(t *testing.T) {
	message := api.Msg{Name: "destname1", IsChan: false}
	message.Content = bytes.NewBufferString(testMessage1)
	node.In() <- message

	t.Log("API Channel TX: ")
	t.Log(message)
}

func Test_stop(t *testing.T) {
	node.Stop()
}

// Test Messages

var testMessage1 = `'In THAT direction,' the Cat said, waving its right paw round, 'lives a Hatter: and in THAT direction,' waving the other paw, 'lives a March Hare. Visit either you like: they're both mad.'
'But I don't want to go among mad people,' Alice remarked.
'Oh, you can't help that,' said the Cat: 'we're all mad here. I'm mad. You're mad.'
'How do you know I'm mad?' said Alice.
'You must be,' said the Cat, 'or you wouldn't have come here.'`

// RSA TEST KEYS
var pubkeyb64 = "LS0tLS1CRUdJTiBQVUJMSUMgS0VZLS0tLS0KTUlJQ0lqQU5CZ2txaGtpRzl3MEJB" +
	"UUVGQUFPQ0FnOEFNSUlDQ2dLQ0FnRUFzSFpRNndSTS9WNXI2REdDcjJpbwpVczEw" +
	"T1JheUlQWkVtNFJ3YXFKU2Y4S2RuYVdhOHNQZFFJbnJwZjBsOWIyZHFPSFdrNDVw" +
	"YkhxUlJleWhPQzhJCk9tbWRmSXdxYm14cXpuUXhDWHRsZWsrd3dyQTdLWGRyVWty" +
	"NGVJSGJkbzFnNlRGQkd3ZVJtR2tsR2t5Wm5MNVgKV2tNWUZnQ2JuN3MxOTFFcm9u" +
	"L3l4ajBXdUtEM3dwZ1pvTjdxeW1UMWRSTEVROGJnSUU0WUQ3UDdRYnBjRjMrRApp" +
	"YnVFUW53R1FxM1lYQnlCa0ZCOTdzVDNjUjVqM2Z2ZlJwd1UweXowYTdxRXp0Nm5F" +
	"NVJXcmtoNGJDUTZPNHg4CjNZdjZqSGtPampNdFNUVlRsNE8zNW51QWFYRXB1NEo5" +
	"S0E2VXpXdzN0eDF6UHNFNkdhaTd3S0kxWmpEOENicHkKU1M3emdrSmR4WWgzRmFn" +
	"Q3dIN2U4emVDRW5YbWdIR01FaUJPZWFoN1MrejE3WlRhSHFzZW1sMjBRR1NEN0F4" +
	"egpMVjlLWHl0NjNmVno4UGE5enAwOW4zUS8yakpYRlFvNzYyQ0dKbGVuT1dOejlL" +
	"ZFVub1MxOE5ZSUdDMi9oOTRECjdWbktDbzVKRVJyRy9Xa3Z6a3hvSnMzTGZESUw1" +
	"VkhFUlI5T3FsVnBWM3oxQ3JHYzV6WitTTXluQ1VsYVdTWHQKMUZzUjNqdFplcXc4" +
	"dmZZZUxXYkRMei8zQUJQME1wbG9tMWxVWUFQWUs5UCtXTEt5OVBYaVlGV2ZJdmJX" +
	"YkV0ZwpjZkl3VXBtYWozLzlETkVwOHBWSThSWFJTa3IxQXVaa0tYMTA1Y0R6amdU" +
	"bjd1Uk1mUFJEeVZwcGw1aWxhN2QvCnA3SnE3eHk5MGxnMnpVWFUwVXVDWGhrQ0F3" +
	"RUFBUT09Ci0tLS0tRU5EIFBVQkxJQyBLRVktLS0tLQo="

// ECC TEST KEYS
var (
	pubprivkeyb64Ecc = "Tcksa18txiwMEocq7NXdeMwz6PPBD+nxCjb/WCtxq1+dln3M3IaOmg+YfTIbBpk+jIbZZZiT+4CoeFzaJGEWmg=="
	pubkeyb64Ecc     = "Tcksa18txiwMEocq7NXdeMwz6PPBD+nxCjb/WCtxq18="
)
