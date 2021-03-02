package VehicleScanner

import (
	"carHiringWebsite/cacheStore"
	"log"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

var (
	request = http.Client{}

	//carData = cacheStore.NewStore("CarData", 60)
	priceCache = cacheStore.NewStore("priceCache", 60*time.Second)
)

//type VehicleType struct {
//	Sizes map[string][]string
//}
//
//type vehicleGroupInfo struct {
//	Doors         int    `json:"doors"`
//	MaxPassengers int    `json:"maxPassengers"`
//	VehicleType   string `json:"bodyStyle"`
//}
//
//type VehicleData struct {
//	AcrissCode   string           `json:"acrissCode"`
//	Info         vehicleGroupInfo `json:"vehicleGroupInfo"`
//	VehicleClass string
//}
//
//func GetVehicleTypes() (map[string]VehicleType, error) {
//
//	data, err := carData.GetData("https://web-api.orange.sixt.com/v2/apps/fleet/country/GB?vehicleType=car", func(key string) (interface{}, error) {
//
//		data := make([]VehicleData, 1)
//
//		err := requestData(key, &data)
//		if err != nil {
//			return nil, err
//		}
//
//		vehicleStore := make(map[string]VehicleType)
//
//		for _, vehicle := range data {
//			carType := convertCarType(vehicle.Info.VehicleType)
//			size := getSize(vehicle.Info.MaxPassengers)
//
//			data, ok := vehicleStore[carType]
//			if !ok {
//				vehicleStore[carType] = VehicleType{Sizes: make(map[string][]string)}
//				vehicleStore[carType].Sizes[size] = []string{vehicle.AcrissCode}
//			} else {
//				sizes, ok := data.Sizes[size]
//				if !ok {
//					data.Sizes[size] = []string{vehicle.AcrissCode}
//				} else {
//					data.Sizes[size] = append(sizes, vehicle.AcrissCode)
//				}
//			}
//		}
//
//		return vehicleStore, nil
//	})
//	if err != nil {
//		return map[string]VehicleType{}, err
//	}
//
//	return data.(map[string]VehicleType), nil
//}
//
//func convertCarType(carType string) string {
//	switch carType {
//	case "Minibus":
//		return "Van"
//	case "SUV":
//		return "Hatchback"
//	case "Multiseater 7":
//		return "Saloon"
//	case "Saloon":
//		return "Saloon"
//	case "Estate":
//		return "Estate"
//	case "Convertible":
//		return "Town"
//	}
//
//	return "Saloon"
//}
//
//func getSize(seats int) string {
//	if seats < 5 {
//		return "Small"
//	} else if seats > 6 {
//		return "Large"
//	} else {
//		return "Medium"
//	}
//
//}
//
//func requestData(url string, data interface{}) error {
//
//	req, err := http.NewRequest("GET", url, nil)
//	if err != nil {
//		log.Fatalln(err)
//	}
//
//	req.Header.Set("Accept-Language", "en-GB,en-US;q=0.9,en;q=0.8")
//	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/88.0.4324.150 Safari/537.36")
//	resp, err := request.Do(req)
//	if err != nil {
//		return err
//	}
//	defer resp.Body.Close()
//
//	err = json.NewDecoder(resp.Body).Decode(data)
//	if err != nil {
//		return err
//	}
//
//	return nil
//
//}

func GetVehiclePrice(vehicleType, vehicleSize int, start time.Time, end time.Time) (float64, error) {

	typeID := vehicleSize

	if vehicleType == 5 {
		typeID += 3
	}

	carTypeID := strconv.Itoa(typeID)
	days := strconv.Itoa(int(end.Sub(start).Hours() / 24))

	url := "https://www.affordrentacar.co.uk/booking/vehicle?SearchForm%5Bsub_category%5D=" + carTypeID +
		"&SearchForm%5Bdate_from%5D=" + start.Format("01/02/06") +
		"&SearchForm%5Bdate_from_time%5D=9%3A00+am&SearchForm%5Bdate_return%5D=" + end.Format("01/02/06") +
		"&SearchForm%5Bdate_return_time%5D=5%3A00+pm"

	data, err := priceCache.GetData(carTypeID+"-"+days, func(key string) (interface{}, error) {

		price, err := requestPrice(url)
		if err != nil {
			return nil, err
		}

		return price, nil
	})

	if err != nil {
		return 0, err
	}

	return data.(float64), nil
}

func requestPrice(url string) (float64, error) {
	var err error

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.Fatalln(err)
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/88.0.4324.150 Safari/537.36")

	resp, err := request.Do(req)
	if err != nil {
		return 0, err
	}

	defer resp.Body.Close()

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return 0, err
	}

	var price float64

	priceSring := strings.TrimSpace(doc.Find(".recommend-price").Text())
	priceSplit := strings.Split(priceSring, ".")
	priceStringClean := priceSplit[0] + "." + priceSplit[1][:2]

	price, err = strconv.ParseFloat(priceStringClean, 64)
	if err != nil {
		return 0, err
	}

	discountedPrice := price * 0.95

	return math.Ceil(discountedPrice*100) / 100, nil
}

//doc.Find("div#psearch-results").Each(func(i int, s *goquery.Selection) {
//	if price != 0 {
//		return
//	}
//	s.Find("a").Each(func(i int, row *goquery.Selection) {
//		if price != 0 {
//			return
//		}
//		row.Find(".cell.pri").Each(func(i int, cell *goquery.Selection) {
//			if price != 0 {
//				return
//			}
//			if strings.TrimSpace(cell.Find(".busper").Text()) == "Personal price" {
//				priceSring := cell.Find(".price.fg-red").Text()[2:]
//
//				price, err = strconv.ParseFloat(priceSring, 64)
//			}
//		})
//	})
//})
