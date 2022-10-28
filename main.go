package main

import (
	"Rosalia64/exe"
	"Rosalia64/ia64"
	"bytes"
	"encoding/binary"
	"fmt"
	"os"
	"strconv"
	"strings"

	"lukechampine.com/uint128"
)

func main() {
	data, err := os.ReadFile("Rimukoro.exe")

	if err != nil {
		panic(err)
	}

	buffer := bytes.NewBuffer(data)

	//Actual reading part
	var dosHeader exe.DOSHeader

	binary.Read(buffer, binary.LittleEndian, &dosHeader)

	peData := data[dosHeader.PEPointer:]
	peBuffer := bytes.NewBuffer(peData)

	var coffHeader exe.COFFHeader
	var peHeaderBytes [4]byte

	binary.Read(peBuffer, binary.LittleEndian, &peHeaderBytes)
	binary.Read(peBuffer, binary.LittleEndian, &coffHeader)

	var signature exe.Signature

	binary.Read(peBuffer, binary.LittleEndian, &signature)

	if signature == exe.SignatureExecutable32bit {
		panic("IA64 isnt 32bit")
	}

	var peOptHeader exe.COFFOptionalHeader64

	binary.Read(peBuffer, binary.LittleEndian, &peOptHeader)

	var dataDirectories []exe.DataDirectory

	for i := int32(0); i != peOptHeader.OptionalHeader.NumberOfRvaAndSizes; i++ {
		var dataDirectory exe.DataDirectory

		binary.Read(peBuffer, binary.LittleEndian, &dataDirectory)

		dataDirectories = append(dataDirectories, dataDirectory)
	}

	var imageSections []exe.ImageSectionHeader

	for i := int32(0); i != int32(coffHeader.NumberOfSections); i++ {
		var imageSectionHeader exe.ImageSectionHeader

		binary.Read(peBuffer, binary.LittleEndian, &imageSectionHeader)

		imageSections = append(imageSections, imageSectionHeader)
	}

	//TODO: don't do this wrong! you arent supposed to start from .text
	var instructionData *bytes.Buffer

	for _, image := range imageSections {
		isText := strings.HasPrefix(string(image.Name[:]), ".text")

		if isText {
			rawData := data[image.PointerToRawData : image.PointerToRawData+image.SizeOfRawData]
			instructionData = bytes.NewBuffer(rawData)
			break
		}
	}

	for {
		var bundle [16]byte

		err := binary.Read(instructionData, binary.LittleEndian, &bundle)

		if err != nil {
			break
		}

		asUint128 := uint128.FromBytes(bundle[:])

		fmt.Print("whole: ")
		fmt.Print(strconv.FormatUint(asUint128.Hi, 2))
		fmt.Print(strconv.FormatUint(asUint128.Lo, 2))
		fmt.Print("\n")

		fmt.Print("split: ")
		fmt.Print(strconv.FormatUint(asUint128.Hi, 2))
		fmt.Print("|")
		fmt.Print(strconv.FormatUint(asUint128.Lo, 2))
		fmt.Print("\n")

		template := asUint128.Lo & 0b11111

		unitOrder := ia64.UnitTable[template]
		slot0 := (asUint128.Lo & 0b000000000001111111111111111111111111111111111111111100000)
		slot1 :=
			(asUint128.Lo&0b111111111110000000000000000000000000000000000000000000000)>>30 |
				(asUint128.Hi&0b000000000000000000000000000111111111111111111111111111111)<<27

		fmt.Printf("slot0: %064b\n", slot0)
		fmt.Printf("slot1: %064b\n", slot1)

		DecodeInstructionSlot(slot0, unitOrder.Slot0)

		break
	}
}

func DecodeInstructionSlot(slot uint64, unit ia64.Unit) {
	majorOpcode := slot & (0b1111 << 42) >> 42

	fmt.Printf("Major Opcode: %d\n", majorOpcode)

	switch unit {
	case ia64.M_Unit:
		ia64.M_UnitInstructionTable[majorOpcode](slot)
	}
}
