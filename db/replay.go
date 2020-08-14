package db

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"time"

	"github.com/MakeNowJust/hotkey"
	"github.com/l3lackShark/gosumemory/memory"
	"github.com/skratchdot/open-golang/open"

	"github.com/ulikunitz/xz/lzma"
)

type osr struct {
	Gamemode         uint8
	OsuVer           int32
	MD5              string
	PlayerName       string
	BmChecksum       string
	Hit300s          uint16
	Hit100s          uint16
	Hit50s           uint16
	HitGekis         uint16
	HitKatus         uint16
	HitMisses        uint16
	Score            int32
	MaxCombo         uint16
	IsPerfect        bool
	Mods             int32
	Lifebar          string
	DateTime         int64
	LengthReplayData int32
	ReplayData       []uint8
	ScoreID          int64
}

var tempBeatmapFailTime int32

const ticksUnix = 621355968000000000 //C# DateTime

//WriteOSR does the write replay magic
func WriteOSR() {
	if _, err := os.Stat("FailedReplays"); os.IsNotExist(err) {
		fmt.Println("FailedReplays Directory does not exist. Making one..")
		os.Mkdir("FailedReplays", 0644)
	}
	hkey := hotkey.New()
	for {
		hkey.Register(0, hotkey.F2, func() { //Windows only for now
			if memory.DynamicAddresses.IsReady == true && memory.GameplayData.GameMode != 3 && memory.GameplayData.IsFailed == 1 && tempBeatmapFailTime != memory.GameplayData.FailTime && memory.GameplayData.FailTime != 0 {
				tempBeatmapFailTime = memory.GameplayData.FailTime
				fmt.Println("Writing replay file...")
				name := fmt.Sprintf("FailedReplays/%s - %s - %s [%s] (%s) %s.osr", memory.GameplayData.Name+"(Failed)", memory.MenuData.Bm.Metadata.Artist, memory.MenuData.Bm.Metadata.Title, memory.MenuData.Bm.Metadata.Version, strings.ReplaceAll(time.Now().Format(time.RFC1123), ":", "-"), gamemodeToStr(memory.GameplayData.GameMode))
				fmt.Println(name)
				file, err := os.Create(name)
				if err != nil {
					fmt.Println(err)
				}
				replayWriter := bufio.NewWriter(file)
				OsrStruct := convertMemoryDataToOSRStruct()
				v := reflect.ValueOf(OsrStruct)
				values := make([]interface{}, v.NumField())
				for i := 0; i < v.NumField(); i++ {
					values[i] = v.Field(i).Interface()
					switch v.Field(i).Kind() {
					case reflect.String:
						replayWriter.WriteByte(0x0B) //please never exceed 255 (TODO: proper strings handler)
						replayWriter.WriteByte(byte(len(v.Field(i).String())))
						replayWriter.WriteString(v.Field(i).String())
					case reflect.Uint8:
						writeUint8(replayWriter, uint8(v.Field(i).Uint()))
					case reflect.Uint16:
						writeUint16(replayWriter, uint16(v.Field(i).Uint()))
					case reflect.Int32:
						writeInt32(replayWriter, int32(v.Field(i).Int()))
					case reflect.Bool:
						writeBool(replayWriter, v.Field(i).Bool())
					case reflect.Int64:
						writeInt64(replayWriter, v.Field(i).Int())
					case reflect.Slice:
						replayWriter.Write(v.Field(i).Bytes())
					default:
						log.Fatalln("Unsupported struct type!")
					}

				}
				replayWriter.Flush()
				file.Close()
				fmt.Println("Finished writing replay file!")
				err = open.Start(filepath.Join(name))
				if err != nil {
					fmt.Println("Replay open err: ", err)
				}
			}

		})
		time.Sleep(time.Duration(memory.UpdateTime) * time.Millisecond)
	}

}

func writeUint8(replayFile *bufio.Writer, number uint8) {
	binary.Write(replayFile, binary.LittleEndian, number)
}
func writeInt32(replayFile *bufio.Writer, number int32) {
	binary.Write(replayFile, binary.LittleEndian, number)
}
func writeUint16(replayFile *bufio.Writer, number uint16) {
	binary.Write(replayFile, binary.LittleEndian, number)
}
func writeInt64(replayFile *bufio.Writer, number int64) {
	binary.Write(replayFile, binary.LittleEndian, number)
}
func writeBool(replayFile *bufio.Writer, number bool) {
	binary.Write(replayFile, binary.LittleEndian, number)
}

func compressToLZMA(input string) []byte {
	text := input
	var buf bytes.Buffer
	// compress text
	w, err := lzma.WriterConfig{DictCap: 16 * 1024 * 1024}.NewWriter(&buf)
	if err != nil {
		log.Fatalf("xz.NewWriter error %s", err)
	}
	if _, err := io.WriteString(w, text); err != nil {
		log.Fatalf("WriteString error %s", err)
	}
	if err := w.Close(); err != nil {
		log.Fatalf("w.Close error %s", err)
	}
	return buf.Bytes()
}

type lzmaString []string

func convertMemoryDataToOSRStruct() osr {
	osrStruct := memory.GameplayData.Replay
	var lzma lzmaString
	lzma = make(lzmaString, len(osrStruct.Replays)+1)
	for i, replayTick := range osrStruct.Replays {
		if i > 0 {
			replayTick.Time = replayTick.Time - osrStruct.Replays[i-1].Time
		}

		lzma[i] = fmt.Sprintf("%d|%f|%f|%d", replayTick.Time, replayTick.X, replayTick.Y, replayTick.WasButtonPressed) //0|256|-500|0, f.e.
	}

	lzma[len(lzma)-1] = "-12345|0|0|0," //every replay has this at the end
	decompressedLZMAStr := strings.Join(lzma, ",")
	compressed := compressToLZMA(decompressedLZMAStr)

	var replay = osr{
		Gamemode:         uint8(memory.GameplayData.GameMode),
		OsuVer:           20190828, //doesn't really matter
		MD5:              memory.MenuData.Bm.BeatmapMD5,
		PlayerName:       memory.GameplayData.Name + ("(Failed)"),
		BmChecksum:       "", //not needed for a functioning replay
		Hit300s:          uint16(memory.GameplayData.Hits.H300),
		Hit100s:          uint16(memory.GameplayData.Hits.H100),
		Hit50s:           uint16(memory.GameplayData.Hits.H50),
		HitGekis:         uint16(memory.GameplayData.Hits.HGeki),
		HitKatus:         uint16(memory.GameplayData.Hits.HKatu),
		HitMisses:        uint16(memory.GameplayData.Hits.H0),
		Score:            memory.GameplayData.Score,
		MaxCombo:         uint16(memory.GameplayData.Combo.Max),
		IsPerfect:        false,
		Mods:             memory.MenuData.Mods.AppliedMods,
		Lifebar:          "",                                     //not needed for a functioning replay
		DateTime:         time.Now().Unix()*10000000 + ticksUnix, //(C# DateTime)
		LengthReplayData: int32(len(compressed)),
		ReplayData:       []uint8(compressed),
		ScoreID:          0,
	}

	return replay
}

func gamemodeToStr(num int32) string {
	switch num {
	case 0:
		return "osuSTD"
	case 1:
		return "osuTaiko"
	case 2:
		return "osuCatch"
	case 3:
		return "osuMania"
	}
	return "Unsupported num"
}