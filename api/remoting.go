package api

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"errors"
	"io"

	"github.com/awgh/bencrypt/bc"

	"github.com/awgh/bencrypt/ecc"
	"github.com/awgh/bencrypt/rsa"
)

// API Parameter Data types
const (
	APITypeInvalid        byte = 0x0
	APITypeNil            byte = 0x1
	APITypeInt64          byte = 0x2
	APITypeUint64         byte = 0x3
	APITypeString         byte = 0x4
	APITypeBytes          byte = 0x5
	APITypeBytesBytes     byte = 0x6
	APITypeInterfaceArray byte = 0x7

	APITypePubKeyECC byte = 0x10
	APITypePubKeyRSA byte = 0x11

	APITypeContactArray byte = 0x20
	APITypeChannelArray byte = 0x21
	APITypeProfileArray byte = 0x22
	APITypePeerArray    byte = 0x23

	APITypeContact byte = 0x30
	APITypeChannel byte = 0x31
	APITypeProfile byte = 0x32
	APITypePeer    byte = 0x33

	APITypeBundle byte = 0x40
)

var (
	ErrInputTooShort = errors.New("input too short")
	ErrLenOverflow   = errors.New("uvarint overflow")
)

// RemoteCall : defines a Remote Procedure Call
type RemoteCall struct {
	Action Action
	Args   []interface{}
}

// RemoteResponse : defines a response returned from a Remote Procedure Call
type RemoteResponse struct {
	Error string
	Value interface{}
}

// IsNil - is this response Nil?
func (r *RemoteResponse) IsNil() bool { return r.Value == nil }

// IsErr - is this response an error?
func (r *RemoteResponse) IsErr() bool { return r.Error != "" }

// ArgsToBytes - converts an interface array to a byte array
func ArgsToBytes(args []interface{}) []byte {
	b := bytes.NewBuffer([]byte{})
	w := bufio.NewWriter(b)
	serialize(w, args)
	w.Flush()
	return b.Bytes()
}

// ArgsFromBytes - converts a byte array to an interface array
func ArgsFromBytes(args []byte) ([]interface{}, error) {
	r := bufio.NewReader(bytes.NewBuffer(args))
	rv, err := deserialize(r)
	var retval []interface{}
	if rv != nil {
		retval = rv.([]interface{})
	}
	return retval, err
}

// Serialization byte order is BigEndian / network-order

// RemoteCallToBytes - converts a RemoteCall to a byte array
func RemoteCallToBytes(call *RemoteCall) *[]byte {
	b := bytes.NewBuffer([]byte{})
	w := bufio.NewWriter(b)
	// Action - first byte
	binary.Write(w, binary.BigEndian, call.Action)
	// Args - everything else
	binary.Write(w, binary.BigEndian, ArgsToBytes(call.Args))
	w.Flush()
	rb := b.Bytes()
	return &rb
}

// RemoteCallFromBytes - converts a RemoteCall from a byte array
func RemoteCallFromBytes(input *[]byte) (*RemoteCall, error) {
	if len(*input) < 2 {
		return nil, ErrInputTooShort
	}
	call := new(RemoteCall)
	action := (*input)[0]
	call.Action = Action(action)
	args, err := ArgsFromBytes((*input)[1:])
	if err != nil {
		return nil, err
	}
	call.Args = args
	return call, nil
}

// RemoteResponseToBytes - converts a RemoteResponse to a byte array
func RemoteResponseToBytes(resp *RemoteResponse) *[]byte {
	b := bytes.NewBuffer([]byte{})
	w := bufio.NewWriter(b)
	serialize(w, resp.Error)
	serialize(w, resp.Value)
	w.Flush()
	retval := b.Bytes()
	return &retval
}

// RemoteResponseFromBytes - converts a RemoteResponse from a byte array
func RemoteResponseFromBytes(input *[]byte) (*RemoteResponse, error) {
	resp := new(RemoteResponse)
	r := bufio.NewReader(bytes.NewBuffer(*input))

	// read the two fields, add to struct
	// Error string
	errString, err := deserialize(r)
	if err != nil {
		return nil, err
	}
	resp.Error = errString.(string)

	// Value interface{}
	value, err := deserialize(r)
	if err != nil {
		return nil, err
	}
	resp.Value = value
	return resp, nil
}

// BytesBytesToBytes - converts an array of byte arrays to a byte array
func BytesBytesToBytes(bba *[][]byte) *[]byte {
	b := bytes.NewBuffer([]byte{})
	w := bufio.NewWriter(b)
	serialize(w, *bba)
	w.Flush()
	retval := b.Bytes()
	return &retval
}

// BytesBytesFromBytes - converts an array of byte arrays from a byte array
func BytesBytesFromBytes(input *[]byte) (*[][]byte, error) {
	r := bufio.NewReader(bytes.NewBuffer(*input))
	bytesBytesArray, err := deserialize(r)
	if err != nil {
		return nil, err
	}
	retval := bytesBytesArray.([][]byte)
	return &retval, nil
}

func serialize(w io.Writer, v interface{}) {
	switch v.(type) {
	case nil:
		writeTLV(w, APITypeNil, nil)
	case int64:
		binary.Write(w, binary.BigEndian, APITypeInt64) // type
		binary.Write(w, binary.BigEndian, v)            // value
	case uint64:
		binary.Write(w, binary.BigEndian, APITypeUint64) // type
		binary.Write(w, binary.BigEndian, v)             // value
	case string:
		s := v.(string)
		writeTLV(w, APITypeString, []byte(s))
	case []byte:
		ba := v.([]byte)
		writeTLV(w, APITypeBytes, ba)
	case [][]byte:
		bba := v.([][]byte)
		lenBuf := make([]byte, binary.MaxVarintLen64)
		n := binary.PutUvarint(lenBuf, uint64(len(bba))) // number of byte arrays
		b := bytes.NewBuffer(lenBuf[:n])
		for i := range bba {
			writeLV(b, bba[i])
		}
		writeTLV(w, APITypeBytesBytes, b.Bytes())
	case []interface{}:
		ia := v.([]interface{})
		lenBuf := make([]byte, binary.MaxVarintLen64)
		n := binary.PutUvarint(lenBuf, uint64(len(ia))) // number of elements in array
		b := bytes.NewBuffer(lenBuf[:n])
		for _, i := range ia {
			serialize(b, i) // recursively serialize
		}
		writeTLV(w, APITypeInterfaceArray, b.Bytes())
	case bc.PubKey:
		pk, ok := v.(*ecc.PubKey)
		var kb []byte
		var typ byte
		if ok {
			kb = pk.ToBytes()
			typ = APITypePubKeyECC
		} else {
			rk := v.(*rsa.PubKey)
			kb = rk.ToBytes()
			typ = APITypePubKeyRSA
		}
		writeTLV(w, typ, kb)
	case *Contact:
		ap := v.(*Contact)
		b := bytes.NewBuffer([]byte{})
		writeLV(b, []byte(ap.Name))
		writeLV(b, []byte(ap.Pubkey))
		writeTLV(w, APITypeContact, b.Bytes())
	case []Contact:
		ac := v.([]Contact)
		lenBuf := make([]byte, binary.MaxVarintLen64)
		n := binary.PutUvarint(lenBuf, uint64(len(ac))) // number of elements in array
		b := bytes.NewBuffer(lenBuf[:n])
		for _, c := range ac {
			writeLV(b, []byte(c.Name))
			writeLV(b, []byte(c.Pubkey))
		}
		writeTLV(w, APITypeContactArray, b.Bytes())
	case *Channel:
		ap := v.(*Channel)
		b := bytes.NewBuffer([]byte{})
		writeLV(b, []byte(ap.Name))
		writeLV(b, []byte(ap.Pubkey))
		writeTLV(w, APITypeChannel, b.Bytes())
	case []Channel:
		ac := v.([]Channel)
		lenBuf := make([]byte, binary.MaxVarintLen64)
		n := binary.PutUvarint(lenBuf, uint64(len(ac))) // number of elements in array
		b := bytes.NewBuffer(lenBuf[:n])
		for _, c := range ac {
			writeLV(b, []byte(c.Name))
			writeLV(b, []byte(c.Pubkey))
		}
		writeTLV(w, APITypeChannelArray, b.Bytes())
	case *Profile:
		ap := v.(*Profile)
		b := bytes.NewBuffer([]byte{})
		writeLV(b, []byte(ap.Name))
		writeLV(b, []byte(ap.Pubkey))
		if ap.Enabled {
			b.WriteByte(1)
		} else {
			b.WriteByte(0)
		}
		writeTLV(w, APITypeProfile, b.Bytes())
	case []Profile:
		ac := v.([]Profile)
		lenBuf := make([]byte, binary.MaxVarintLen64)
		n := binary.PutUvarint(lenBuf, uint64(len(ac))) // number of elements in array
		b := bytes.NewBuffer(lenBuf[:n])
		for _, c := range ac {
			writeLV(b, []byte(c.Name))
			writeLV(b, []byte(c.Pubkey))
			if c.Enabled {
				b.WriteByte(1)
			} else {
				b.WriteByte(0)
			}
		}
		writeTLV(w, APITypeProfileArray, b.Bytes())

	case *Peer:
		ap := v.(*Peer)
		b := bytes.NewBuffer([]byte{})
		writeLV(b, []byte(ap.Name))
		writeLV(b, []byte(ap.Group))
		writeLV(b, []byte(ap.URI))
		if ap.Enabled {
			b.WriteByte(1)
		} else {
			b.WriteByte(0)
		}
		writeTLV(w, APITypePeer, b.Bytes())
	case []Peer:
		ac := v.([]Peer)
		lenBuf := make([]byte, binary.MaxVarintLen64)
		n := binary.PutUvarint(lenBuf, uint64(len(ac))) // number of elements in array
		b := bytes.NewBuffer(lenBuf[:n])
		for _, c := range ac {
			writeLV(b, []byte(c.Name))
			writeLV(b, []byte(c.Group))
			writeLV(b, []byte(c.URI))
			if c.Enabled {
				b.WriteByte(1)
			} else {
				b.WriteByte(0)
			}
		}
		writeTLV(w, APITypePeerArray, b.Bytes())
	case Bundle:
		bundle := v.(Bundle)
		b := bytes.NewBuffer([]byte{})
		writeLV(b, bundle.Data)
		binary.Write(b, binary.BigEndian, bundle.Time)
		writeTLV(w, APITypeBundle, b.Bytes())
		// default:
		//	log.Printf("Unknown type in serialize: %T\n", v)
	}
}

// deserialize - reads the next value from the io.Reader
func deserialize(r io.Reader) (interface{}, error) {
	// read the type byte
	var t byte
	if err := binary.Read(r, binary.BigEndian, &t); err != nil {
		return nil, err
	}

	// Values with no length
	switch t {
	case APITypeNil:
		return nil, nil
	case APITypeInt64:
		var vint int64
		if err := binary.Read(r, binary.BigEndian, &vint); err != nil {
			return nil, err
		}
		return vint, nil
	case APITypeUint64:
		var vint uint64
		if err := binary.Read(r, binary.BigEndian, &vint); err != nil {
			return nil, err
		}
		return vint, nil
	}

	// read length and value
	v, err := readLV(r)
	if err != nil {
		return nil, err
	}

	// Values with a length
	switch t {
	case APITypeString:
		return string(v), nil
	case APITypeBytes:
		return v, nil
	case APITypeBytesBytes:
		var bba [][]byte
		l, n := binary.Uvarint(v)
		if n == 0 {
			return nil, ErrInputTooShort
		} else if n < 0 {
			return nil, ErrLenOverflow
		}
		b := bytes.NewReader(v[n:])
		for i := uint64(0); i < l; i++ {
			ba, err := readLV(b)
			if err != nil {
				return nil, err
			}
			bba = append(bba, ba)
		}
		return bba, nil
	case APITypeInterfaceArray:
		var ia []interface{}
		l, n := binary.Uvarint(v)
		if n == 0 {
			return nil, ErrInputTooShort
		} else if n < 0 {
			return nil, ErrLenOverflow
		}
		b := bytes.NewReader(v[n:])
		for i := uint64(0); i < l; i++ {
			element, err := deserialize(b)
			if err != nil {
				return nil, err
			}
			ia = append(ia, element)
		}
		return ia, nil
	case APITypePubKeyECC:
		key := new(ecc.PubKey)
		if err := key.FromBytes(v); err != nil {
			return nil, err
		}
		return key, nil
	case APITypePubKeyRSA:
		key := new(rsa.PubKey)
		if err := key.FromBytes(v); err != nil {
			return nil, err
		}
		return key, nil

	case APITypeContact:
		var contact Contact
		b := bytes.NewBuffer(v)
		va, err := readLV(b)
		if err != nil {
			return nil, err
		}
		contact.Name = string(va)
		va, err = readLV(b)
		if err != nil {
			return nil, err
		}
		contact.Pubkey = string(va)
		return &contact, nil

	case APITypeContactArray:
		var contacts []Contact
		l, n := binary.Uvarint(v)
		if n == 0 {
			return nil, ErrInputTooShort
		} else if n < 0 {
			return nil, ErrLenOverflow
		}
		b := bytes.NewReader(v[n:])
		for i := uint64(0); i < l; i++ {
			var contact Contact
			va, err := readLV(b)
			if err != nil {
				return nil, err
			}
			contact.Name = string(va)
			va, err = readLV(b)
			if err != nil {
				return nil, err
			}
			contact.Pubkey = string(va)
			contacts = append(contacts, contact)
		}
		return contacts, nil

	case APITypeChannel:
		var channel Channel
		b := bytes.NewBuffer(v)
		va, err := readLV(b)
		if err != nil {
			return nil, err
		}
		channel.Name = string(va)
		va, err = readLV(b)
		if err != nil {
			return nil, err
		}
		channel.Pubkey = string(va)
		return &channel, nil

	case APITypeChannelArray:
		var channels []Channel
		l, n := binary.Uvarint(v)
		if n == 0 {
			return nil, ErrInputTooShort
		} else if n < 0 {
			return nil, ErrLenOverflow
		}
		b := bytes.NewReader(v[n:])
		for i := uint64(0); i < l; i++ {
			var channel Channel
			va, err := readLV(b)
			if err != nil {
				return nil, err
			}
			channel.Name = string(va)
			va, err = readLV(b)
			if err != nil {
				return nil, err
			}
			channel.Pubkey = string(va)
			channels = append(channels, channel)
		}
		return channels, nil

	case APITypeProfile:
		var profile Profile
		b := bytes.NewBuffer(v)
		va, err := readLV(b)
		if err != nil {
			return nil, err
		}
		profile.Name = string(va)
		va, err = readLV(b)
		if err != nil {
			return nil, err
		}
		profile.Pubkey = string(va)
		bt, err := b.ReadByte()
		if err != nil {
			return nil, err
		}
		if bt == 1 {
			profile.Enabled = true
		} else {
			profile.Enabled = false
		}
		return &profile, nil

	case APITypeProfileArray:
		var profiles []Profile
		l, n := binary.Uvarint(v)
		if n == 0 {
			return nil, ErrInputTooShort
		} else if n < 0 {
			return nil, ErrLenOverflow
		}
		b := bytes.NewReader(v[n:])
		for i := uint64(0); i < l; i++ {
			var profile Profile
			va, err := readLV(b)
			if err != nil {
				return nil, err
			}
			profile.Name = string(va)
			va, err = readLV(b)
			if err != nil {
				return nil, err
			}
			profile.Pubkey = string(va)

			bt, err := b.ReadByte()
			if err != nil {
				return nil, err
			}
			if bt == 1 {
				profile.Enabled = true
			} else {
				profile.Enabled = false
			}
			profiles = append(profiles, profile)
		}
		return profiles, nil

	case APITypePeer:
		var peer Peer
		b := bytes.NewBuffer(v)
		va, err := readLV(b)
		if err != nil {
			return nil, err
		}
		peer.Name = string(va)
		va, err = readLV(b)
		if err != nil {
			return nil, err
		}
		peer.Group = string(va)
		va, err = readLV(b)
		if err != nil {
			return nil, err
		}
		peer.URI = string(va)
		bt, err := b.ReadByte()
		if err != nil {
			return nil, err
		}
		if bt == 1 {
			peer.Enabled = true
		} else {
			peer.Enabled = false
		}
		return &peer, nil

	case APITypePeerArray:
		var peers []Peer
		l, n := binary.Uvarint(v)
		if n == 0 {
			return nil, ErrInputTooShort
		} else if n < 0 {
			return nil, ErrLenOverflow
		}
		b := bytes.NewReader(v[n:])
		for i := uint64(0); i < l; i++ {
			var peer Peer
			va, err := readLV(b)
			if err != nil {
				return nil, err
			}
			peer.Name = string(va)
			va, err = readLV(b)
			if err != nil {
				return nil, err
			}
			peer.Group = string(va)
			va, err = readLV(b)
			if err != nil {
				return nil, err
			}
			peer.URI = string(va)
			bt, err := b.ReadByte()
			if err != nil {
				return nil, err
			}
			if bt == 1 {
				peer.Enabled = true
			} else {
				peer.Enabled = false
			}
			peers = append(peers, peer)
		}
		return peers, nil

	case APITypeBundle:
		var bundle Bundle
		b := bytes.NewBuffer(v)
		data, err := readLV(b)
		if err != nil {
			return nil, err
		}
		bundle.Data = data
		var vint int64
		if err := binary.Read(b, binary.BigEndian, &vint); err != nil {
			return nil, err
		}
		bundle.Time = vint
		return bundle, nil
	}
	return nil, errors.New("Unknown Type")
}

func writeTLV(w io.Writer, typ byte, value []byte) {
	binary.Write(w, binary.BigEndian, typ) // type
	if typ != APITypeNil {
		writeLV(w, value)
	}
}

func writeLV(w io.Writer, value []byte) {
	lenBuf := make([]byte, binary.MaxVarintLen64)
	n := binary.PutUvarint(lenBuf, uint64(len(value)))
	w.Write(lenBuf[:n]) // length
	w.Write(value)      // value
}

/*
// readTLV - returns type as byte, value as []byte
func readTLV(r io.Reader) (byte, []byte, error) {
	var t byte
	if err := binary.Read(r, binary.BigEndian, &t); err != io.EOF {
		return 0, nil, nil //EOF
	} else if err != nil {
		return t, nil, err
	} else if t == APITypeNil {
		return t, nil, nil
	}
	v, err := readLV(r)
	return t, v, err
}
*/

func readLV(r io.Reader) ([]byte, error) {
	l, err := binary.ReadUvarint(r.(io.ByteReader))
	if err != nil {
		return nil, err
	}
	if l == 0 {
		return nil, nil
	}
	v := make([]byte, l)
	if err := binary.Read(r, binary.BigEndian, &v); err != nil {
		return nil, err
	}
	return v, nil
}

// ReadBuffer - reads a serialized buffer from the wire, returns buffer.
// If the reader is not an io.ByteReader, it will be wrapped with one
// internally
func ReadBuffer(reader io.Reader) (*[]byte, error) {
	br, ok := reader.(io.ByteReader)
	if !ok {
		br = newByteReader(reader)
	}
	rlen, err := binary.ReadUvarint(br)
	if err != nil {
		return nil, err
	}
	buf := make([]byte, rlen)
	n, err := io.ReadFull(reader, buf)
	if uint64(n) != rlen {
		return nil, errors.New("ReadBuffer read underflow")
	}
	if err != nil {
		return nil, err
	}
	return &buf, nil
}

// WriteBuffer - writes a serialized buffer to the wire
func WriteBuffer(writer io.Writer, b *[]byte) error {
	lenBuf := make([]byte, binary.MaxVarintLen64)
	n := binary.PutUvarint(lenBuf, uint64(len(*b)))
	rb := append(lenBuf[:n], *b...)
	if _, err := writer.Write(rb); err != nil {
		return err
	}
	return nil
}

/*
// BundleToBytes - converts a Bundle to a byte array
func BundleToBytes(bundle *Bundle) *[]byte {
	b := bytes.NewBuffer([]byte{})
	w := bufio.NewWriter(b)
	serialize(w, bundle)
	w.Flush()
	retval := b.Bytes()
	return &retval
}

// BundleFromBytes - converts a Bundle from a byte array
func BundleFromBytes(input *[]byte) (*Bundle, error) {
	r := bufio.NewReader(bytes.NewBuffer(*input))
	t, v, err := readTLV(r)
	if err != nil {
		return nil, err
	}
	rv, err := deserialize(r, t, v)
	if err != nil {
		return nil, err
	}
	bundle := rv.(Bundle)
	return &bundle, nil
}
*/
