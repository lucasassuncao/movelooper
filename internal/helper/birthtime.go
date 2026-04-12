package helper

import (
	"math"
	"os"
	"reflect"
	"runtime"
	"time"
)

// getBirthTime returns the file creation (birth) time.
// On Windows it reads CreationTime from Win32FileAttributeData.
// On macOS it reads Birthtimespec from Stat_t.
// On other platforms it falls back to modification time.
func getBirthTime(info os.FileInfo) time.Time {
	sys := info.Sys()
	if sys == nil {
		return info.ModTime()
	}

	v := reflect.ValueOf(sys)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	switch runtime.GOOS {
	case "windows":
		// syscall.Win32FileAttributeData.CreationTime is a syscall.Filetime{LowDateTime, HighDateTime uint32}
		ct := v.FieldByName("CreationTime")
		if !ct.IsValid() {
			return info.ModTime()
		}
		lo := ct.FieldByName("LowDateTime").Uint()
		hi := ct.FieldByName("HighDateTime").Uint()
		ft := hi<<32 | lo
		// Windows FILETIME: 100-ns intervals since 1601-01-01; Unix epoch offset in same unit.
		const windowsEpochOffset uint64 = 116444736000000000
		if ft < windowsEpochOffset {
			return info.ModTime()
		}
		ns := ft - windowsEpochOffset
		if ns > math.MaxInt64/100 {
			return info.ModTime()
		}
		return time.Unix(0, int64(ns)*100)

	case "darwin":
		// syscall.Stat_t.Birthtimespec is a syscall.Timespec{Sec, Nsec int64}
		bts := v.FieldByName("Birthtimespec")
		if !bts.IsValid() {
			return info.ModTime()
		}
		return time.Unix(bts.FieldByName("Sec").Int(), bts.FieldByName("Nsec").Int())

	default:
		return info.ModTime()
	}
}
