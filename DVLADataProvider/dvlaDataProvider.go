package DVLADataProvider

import (
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"strings"
	"sync"
	"time"
)

var (
	statuses = map[string]int{
		"Unknown":      0,
		"Suspended":    1,
		"LostOrStolen": 2,
		"Expired":      3,
	}

	csvLock sync.RWMutex
	csvData map[string]int
	dir     = `./DVLAfiles/`

	InvalidLicense = errors.New("invalidLicense")
)

func InitProvider() {
	csvData = make(map[string]int)

	if _, err := os.Stat(dir); os.IsNotExist(err) {
		os.Mkdir(dir, os.ModePerm)
	} else {
		LoadCSVData()
	}

	go func() {

		for {
			time.Sleep(5 * time.Second)
			LoadCSVData()
		}

	}()

}

func LoadCSVData() error {

	files, err := ioutil.ReadDir(dir)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	var modTime time.Time
	var names []string
	for _, fi := range files {

		filename := strings.Split(fi.Name(), ".")
		fileType := filename[len(filename)-1]
		if fileType != "csv" {
			continue
		}

		if fi.Mode().IsRegular() {
			if !fi.ModTime().Before(modTime) {
				if fi.ModTime().After(modTime) {
					modTime = fi.ModTime()
					names = names[:0]
				}
				names = append(names, fi.Name())
			}
		}
	}

	csvfile, err := os.Open(dir + names[0])

	if err != nil {
		return err
	}
	defer csvfile.Close()

	reader := csv.NewReader(csvfile)
	if _, err := reader.Read(); err != nil { //read header
		log.Fatal(err)
	}

	csvLock.Lock()
	for {
		rec, err := reader.Read()
		if err != nil {
			if err == io.EOF {
				break
			}
			log.Fatal(err)

		}
		status := 0
		if _, ok := statuses[rec[8]]; ok {
			status = statuses[rec[8]]
		}

		csvData[rec[0]] = status
	}
	csvLock.Unlock()
	return nil

}

func GetLicenseStatus(licenceNumber string) int {

	csvLock.RLock()
	defer csvLock.RUnlock()
	if _, ok := csvData[licenceNumber]; ok {
		return csvData[licenceNumber]
	}

	return -1
}

func IsInvalidLicense(licenceNumber string) bool {

	status := GetLicenseStatus(licenceNumber)

	return status != -1

}
