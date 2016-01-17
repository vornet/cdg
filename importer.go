package cdg

import (
	"fmt"
	"syscall"
	"unsafe"
	"bytes"
	"encoding/binary"
	"strconv"
	"os"
)

const (
	MaxNumberTracks = 100
)

var (
	kernel32, _        = syscall.LoadLibrary("kernel32.dll")
	getModuleHandle, _ = syscall.GetProcAddress(kernel32, "GetModuleHandleW")
    
    offsets = []int { 0, 66, 125, 191, 100, 50, 150, 175,
        8, 33, 58, 83, 108, 133, 158, 183,
        16, 41, 25, 91, 116, 141, 166, 75 }
)

type TrackData struct {
	Unused [4]uint8
	Address [4]uint8
}

type CDRomToc struct {
	Length [2]uint8
	FirstTrack uint8
	LastTrack uint8
	TrackData [MaxNumberTracks]TrackData
}

// 16
type RawReadInfo struct {
	DiskOffset uint64
	SectorCount uint32
	TrackMode uint32
}

func NewImporter(driveLetter string) *Importer {
	importer := &Importer{ driveLetter }
	importer.init()
	return importer
}

type Importer struct {
	driveLetter string
}

func (i *Importer) init() {

}

func (i *Importer) ImportDisc() {
	fmt.Println("Working on it!!!")

	cdHandle, err := syscall.CreateFile(
		syscall.StringToUTF16Ptr("\\\\.\\" + i.driveLetter),
		0x80000000, // Generic Read
		0x00000001, // File Share Read
		nil,
		3, // Open Existing
		0,
		0)

	if err != nil {
		fmt.Println("\\\\.\\" + i.driveLetter)
		fmt.Println(err)
		return
	}

	var cd CDRomToc
	var buffer [unsafe.Sizeof(cd)]byte
	var bytes_returned uint32 = 0

	err = syscall.DeviceIoControl(
		cdHandle,
		0x00024000, // IOCTL_CDROM_READ_TOC
		nil,
		0,
		(*byte)(unsafe.Pointer(&buffer)),
		uint32(len(buffer)),
		(*uint32)(unsafe.Pointer(&bytes_returned)),
		nil)

	if err != nil {
		fmt.Println("foo")
		fmt.Println(int32(len(buffer)))
		fmt.Println(err)
		return
	}

	buf := bytes.NewReader(buffer[:])
	err = binary.Read(buf, binary.LittleEndian, &cd)
	if err != nil {
		fmt.Println("binary.Read failed:", err)
	}
	// 38084
	trackCount := int(cd.LastTrack - cd.FirstTrack + 1)
    cdgData := []byte {}
        
	for track := 0; track < trackCount; track++ {

		pcmFile, _ := os.Create("testing_" + strconv.Itoa(track) + ".pcm")
		cdgFile, _ := os.Create("testing_" + strconv.Itoa(track) + ".cdg")

		address1 := cd.TrackData[track].Address[1]
		address2 := cd.TrackData[track].Address[2]
		address3 := cd.TrackData[track].Address[3]
		trackStartSector := int(address1) * 60 * 75 + int(address2) * 75 + int(address3) - 150

		address1 = cd.TrackData[track+1].Address[1]
		address2 = cd.TrackData[track+1].Address[2]
		address3 = cd.TrackData[track+1].Address[3]
		trackEndSector := int(address1) * 60 * 75 + int(address2) * 75 + int(address3) - 151

		currentSector := trackStartSector
		const SectorsPerRead = 20

		for currentSector <= trackEndSector {
			rwReadInfo := RawReadInfo {
				DiskOffset: uint64(currentSector) * 2048,
				SectorCount: uint32(SectorsPerRead),
				TrackMode: 5,
			}

			var buffer [2448 * SectorsPerRead]byte

			readSuccessful := syscall.DeviceIoControl(
				cdHandle,
				0x0002403E, // IOCTL_CDROM_RAW_READ
				(*byte)(unsafe.Pointer(&rwReadInfo)),
				16,
				(*byte)(unsafe.Pointer(&buffer)),
				uint32(len(buffer)),
				(*uint32)(unsafe.Pointer(&bytes_returned)),
				nil)

			if readSuccessful == nil {
				for sectorCount := 0; sectorCount < SectorsPerRead; sectorCount++ {
					sectorOffset := sectorCount * 2448
					audioSlice := buffer[sectorOffset + 0:sectorOffset + 2352]
					cdgSlice := buffer[sectorOffset + 2352:sectorOffset + 2448]

                    pcmFile.Write(audioSlice)
					
					for _, cdgByte := range cdgSlice {
                       cdgData = append(cdgData, cdgByte & 0x3F) 
                    }
					   
				}
			}

			currentSector += SectorsPerRead
		}
        
        sectors := (len(cdgData) / 96) - 2

        for s := 0; s < sectors; s++ {
            for p := 0; p < 4; p++ {
                deinterleavedData := []byte {}
                for column := 0; column < 24; column++ {
                    deinterleavedData = append(deinterleavedData, cdgData[s * 96 + p * 24 + offsets[column]])                    
                }                    
                cdgFile.Write(deinterleavedData)
            }                    
        }
                
		pcmFile.Close()
		cdgFile.Close()

		fmt.Println(".")
	}

	fmt.Println(trackCount)
	fmt.Println(cd)
	fmt.Println(buffer)
}
