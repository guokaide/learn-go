package main

import (
	"encoding/binary"
	"fmt"
)

func main() {
	body := encode("Hello, Goim!")
	decode(body)
}

func decode(data []byte) {
	if len(data) <= 16 {
		fmt.Println("data len <= 16.")
		return
	}
	packetLen := binary.BigEndian.Uint32(data[:4])
	fmt.Printf("packetLen:%v\n", packetLen)

	headerLen := binary.BigEndian.Uint16(data[4:6])
	fmt.Printf("headerLen:%v\n", headerLen)

	version := binary.BigEndian.Uint16(data[6:8])
	fmt.Printf("version:%v\n", version)

	operation := binary.BigEndian.Uint32(data[8:12])
	fmt.Printf("operation:%v\n", operation)

	sequence := binary.BigEndian.Uint32(data[12:16])
	fmt.Printf("sequence:%v\n", sequence)

	body := string(data[16:])
	fmt.Printf("body:%v\n", body)
}

// Goim 协议结构
// PacketLen 4 bytes 包长度，在数据流传输过程中，先写入整个包的长度，方便整个包的数据读取。
// HeaderLen 2 bytes 头长度，在处理数据时，会先解析头部，可以知道具体业务操作。
// Version   2 bytes 协议版本号，主要用于上行和下行数据包按版本号进行解析。
// Operation 4 bytes 业务操作码，可以按操作码进行分发数据包到具体业务当中。
// Sequence  4 bytes 序列号，数据包的唯一标记，可以做具体业务处理，或者数据包去重。
// Body      PacketLen - HeaderLen 实际业务数据，在业务层中会进行数据解码和编码。
func encode(body string) []byte {
	headerLen := 16
	packetLen := headerLen + len(body)
	res := make([]byte, packetLen)

	// header
	binary.BigEndian.PutUint32(res[:4], uint32(packetLen))
	binary.BigEndian.PutUint16(res[4:6], uint16(headerLen))

	version := 1
	binary.BigEndian.PutUint16(res[6:8], uint16(version))
	operation := 1
	binary.BigEndian.PutUint32(res[8:12], uint32(operation))
	sequence := 1
	binary.BigEndian.PutUint32(res[12:16], uint32(sequence))

	// body
	bytes := []byte(body)
	copy(res[16:], bytes)

	return res
}
