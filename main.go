package main

import (
	"bytes"
	"carHiringWebsite/db"
	"carHiringWebsite/response"
	"carHiringWebsite/services/adminService"
	"carHiringWebsite/services/bookingService"
	"carHiringWebsite/services/carService"
	"carHiringWebsite/services/userService"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

func main() {
	var err error

	buildAll := flag.Bool("buildall", false, "tells the webserver to rebuild the frontEnd")

	flag.Parse()

	// Build front-end if param specified
	if *buildAll {
		cmd := exec.Command("ng", "build", "--prod", "--output-path=../public")
		cmd.Dir = "./carHiringWebsite-Frontend"

		_, err := cmd.Output()
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println("Finished building")
	}

	// Initiate db connection
	err = db.InitDB()
	if err != nil {
		log.Fatal(err)
	}

	//Serve the website files generated from the build-job in public
	//fileServe := http.FileServer(http.Dir("./public"))

	http.HandleFunc("/", SiteHandler)

	//Service endpoints
	http.HandleFunc("/userService/register", registrationHandler)
	http.HandleFunc("/userService/login", loginHandler)
	http.HandleFunc("/userService/logout", logoutHandler)
	http.HandleFunc("/userService/sessionCheck", sessionCheckHandler)

	http.HandleFunc("/carService/getAll", getAllCarsHandler)
	http.HandleFunc("/carService/get", getCarHandler)
	http.HandleFunc("/carService/getAccessories", getCarAccessoriesHandler)
	http.HandleFunc("/carService/getBookings", getCarBookingsHandler)
	http.HandleFunc("/carService/getCarAttributes", getCarAttributesHandler)

	http.HandleFunc("/bookingService/create", createBookingHandler)
	http.HandleFunc("/bookingService/makePayment", makePaymentHandler)
	http.HandleFunc("/bookingService/getUserBookings", getUsersBookingsHandler)
	http.HandleFunc("/bookingService/cancelBooking", cancelBookingHandler)
	http.HandleFunc("/bookingService/history", historyBookingHandler)
	http.HandleFunc("/bookingService/editBooking", editBookingHandler)

	http.HandleFunc("/adminService/getBookingStats", getBookingStatsHandler)
	http.HandleFunc("/adminService/getUserStats", getUserStatsHandler)
	http.HandleFunc("/adminService/getCarStats", getCarStatsHandler)
	http.HandleFunc("/adminService/getAccessoryStats", getAccessoryStatsHandler)
	http.HandleFunc("/adminService/getSearchedBookings", getSearchedBookingsHandler)
	http.HandleFunc("/adminService/getAwaitingBookings", getAwaitingBookingsHandler)
	http.HandleFunc("/adminService/getQueryingRefundBookings", getQueryingRefundBookings)
	http.HandleFunc("/adminService/getBookingStatuses", getBookingStatusesHandler)
	http.HandleFunc("/adminService/getBooking", getAdminBookingHandler)
	http.HandleFunc("/adminService/progressBooking", progressBookingHandler)
	http.HandleFunc("/adminService/processExtraPayment", processExtraPaymentHandler)
	http.HandleFunc("/adminService/processRefund", processRefundHandler)
	//Server operation
	err = http.ListenAndServe(":8080", nil)
	if err != nil {
		log.Fatal(err)
	}

	//Close db connection
	err = db.CloseDB()
	if err != nil {
		log.Fatal(err)
	}
}

func SiteHandler(w http.ResponseWriter, r *http.Request) {

	paths := strings.Split(r.RequestURI, "/")
	if len(paths) >= 0 {
		if strings.Compare(paths[1], "cars") == 0 {
			r.RequestURI = paths[2]
			r.URL.Path = paths[2]
			carFileServe := http.FileServer(http.Dir("./cars"))
			carFileServe.ServeHTTP(w, r)
			return
		}
	}

	fileServe := http.FileServer(http.Dir("./public"))
	if len(filepath.Ext(r.RequestURI)) == 0 {
		r.RequestURI = "/"
		r.URL.Path = "/"
	}
	fileServe.ServeHTTP(w, r)
}

func registrationHandler(w http.ResponseWriter, r *http.Request) {
	enableCors(&w)
	var err error

	defer func() {
		if err != nil {
			log.Printf("registrationHandler error - err: %v\nurl:%v\ncookies: %+v\n", err, r.URL, r.Cookies())
			w.WriteHeader(http.StatusInternalServerError)
		}
	}()

	if r.Method != http.MethodGet {
		err = errors.New("incorrect http method")
		return
	}

	firstname := r.FormValue("firstname")
	names := r.FormValue("names")
	email := r.FormValue("email")
	password := r.FormValue("password")
	dobString := r.FormValue("dob")
	if len(email) == 0 || len(password) == 0 || len(dobString) == 0 || len(firstname) == 0 || len(names) == 0 {
		err = errors.New("incorrect parameters")
		return
	}

	dobUnix, err := strconv.ParseInt(dobString, 10, 64)
	if err != nil {
		err = errors.New("error reading DOB")
		return
	}

	dob := time.Unix(dobUnix, 0)

	if !userService.ValidateCredentials(email, password) {
		err = errors.New("userService failed validation")
		return
	}

	created, newUser, err := userService.CreateUser(email, password, firstname, names, dob)
	if err != nil {
		return
	}

	var buffer bytes.Buffer
	encoder := json.NewEncoder(&buffer)

	if !created {
		encoder.Encode(response.DuplicateUser)
		w.Write(buffer.Bytes())
		return
	}

	encoder.Encode(&newUser)
	w.Write(buffer.Bytes())
}

func loginHandler(w http.ResponseWriter, r *http.Request) {
	enableCors(&w)
	var err error

	defer func() {
		if err != nil {
			log.Printf("loginHandler error - err: %v\nurl:%v\ncookies: %+v\n", err, r.URL, r.Cookies())
			w.WriteHeader(http.StatusInternalServerError)
		}
	}()

	if r.Method != http.MethodGet {
		err = errors.New("incorrect http method")
		return
	}

	email := r.FormValue("email")
	password := r.FormValue("password")

	if len(email) == 0 || len(password) == 0 {
		err = errors.New("incorrect parameters")
		return
	}

	authUser, authorised, err := userService.Authenticate(email, password)
	if err != nil {
		return
	}

	var buffer bytes.Buffer
	encoder := json.NewEncoder(&buffer)

	if !authorised {
		encoder.Encode(response.IncorrectPassword)
		w.Write(buffer.Bytes())
		return
	}

	encoder.Encode(&authUser)
	w.Write(buffer.Bytes())

}

func logoutHandler(w http.ResponseWriter, r *http.Request) {
	enableCors(&w)
	var err error

	defer func() {
		if err != nil {
			log.Printf("logoutHandler error - err: %v\nurl:%v\ncookies: %+v\n", err, r.URL, r.Cookies())
			w.WriteHeader(http.StatusInternalServerError)
		}
	}()

	if r.Method != http.MethodGet {
		err = errors.New("incorrect http method")
		return
	}

	token, err := r.Cookie("session-token")
	if err != nil {
		return
	}

	if len(token.Value) == 0 {
		err = errors.New("incorrect session")
		return
	}

	err = userService.Logout(token.Value)
	if err != nil {
		return
	}

}

func sessionCheckHandler(w http.ResponseWriter, r *http.Request) {
	enableCors(&w)
	var err error

	defer func() {
		if err != nil {
			log.Printf("sessionCheckHandler error - err: %v\nurl:%v\ncookies: %+v\n", err, r.URL, r.Cookies())
			w.WriteHeader(http.StatusInternalServerError)
		}
	}()

	if r.Method != http.MethodGet {
		err = errors.New("incorrect http method")
		return
	}

	token, err := r.Cookie("session-token")
	if err != nil {
		return
	}

	if len(token.Value) == 0 {
		err = errors.New("incorrect parameters")
		return
	}
	outputUser, err := userService.ValidateSession(token.Value)
	if err != nil {
		return
	}

	var buffer bytes.Buffer
	encoder := json.NewEncoder(&buffer)
	encoder.Encode(&outputUser)
	w.Write(buffer.Bytes())
}

//CAR SERVICE

func getAllCarsHandler(w http.ResponseWriter, r *http.Request) {
	enableCors(&w)
	w.Header().Set("Cache-Control", "public")
	w.Header().Set("Cache-Control", "max-age=60")

	var err error

	defer func() {
		if err != nil {
			log.Printf("getAllCarsHandler error - err: %v\nurl:%v\ncookies: %+v\n", err, r.URL, r.Cookies())
			w.WriteHeader(http.StatusInternalServerError)
		}
	}()

	if r.Method != http.MethodGet {
		err = errors.New("incorrect http method")
		return
	}

	fuelTypes := r.FormValue("fuelTypes")
	gearTypes := r.FormValue("gearTypes")
	carTypes := r.FormValue("carTypes")
	carSizes := r.FormValue("carSizes")
	colours := r.FormValue("colours")
	search := r.FormValue("search")

	cars, err := carService.GetAllCars(fuelTypes, gearTypes, carTypes, carSizes, colours, search)
	if err != nil {
		return
	}

	var buffer bytes.Buffer
	encoder := json.NewEncoder(&buffer)
	encoder.Encode(cars)
	w.Write(buffer.Bytes())
}

func getCarHandler(w http.ResponseWriter, r *http.Request) {
	enableCors(&w)
	var err error

	defer func() {
		if err != nil {
			log.Printf("GetCarsHandler error - err: %v\nurl:%v\ncookies: %+v\n", err, r.URL, r.Cookies())
			w.WriteHeader(http.StatusInternalServerError)
		}
	}()

	if r.Method != http.MethodGet {
		err = errors.New("incorrect http method")
		return
	}

	id := r.FormValue("id")
	if id == "" {
		err = errors.New("incorrect parameters")
		return
	}

	cars, err := carService.GetCar(id)
	if err != nil {
		return
	}

	var buffer bytes.Buffer
	encoder := json.NewEncoder(&buffer)
	encoder.Encode(cars)
	w.Write(buffer.Bytes())
}

func getCarAccessoriesHandler(w http.ResponseWriter, r *http.Request) {
	enableCors(&w)
	var err error

	defer func() {
		if err != nil {
			log.Printf("getCarAccessoriesHandler error - err: %v\nurl:%v\ncookies: %+v\n", err, r.URL, r.Cookies())
			w.WriteHeader(http.StatusInternalServerError)
		}
	}()

	if r.Method != http.MethodGet {
		err = errors.New("incorrect http method")
		return
	}

	start := r.FormValue("start")
	end := r.FormValue("end")
	if start == "" || end == "" {
		err = errors.New("incorrect parameters")
		return
	}

	accessories, err := carService.GetCarAccessories(start, end)
	if err != nil {
		return
	}

	var buffer bytes.Buffer
	encoder := json.NewEncoder(&buffer)
	encoder.Encode(accessories)
	w.Write(buffer.Bytes())
}

func getCarAttributesHandler(w http.ResponseWriter, r *http.Request) {
	enableCors(&w)
	var err error

	defer func() {
		if err != nil {
			log.Printf("getCarAttributesHandler error - err: %v\nurl:%v\ncookies: %+v\n", err, r.URL, r.Cookies())
			w.WriteHeader(http.StatusInternalServerError)
		}
	}()

	if r.Method != http.MethodGet {
		err = errors.New("incorrect http method")
		return
	}

	attributes, err := carService.GetCarAttributes()
	if err != nil {
		return
	}

	var buffer bytes.Buffer
	encoder := json.NewEncoder(&buffer)
	encoder.Encode(attributes)
	w.Write(buffer.Bytes())
}

func getCarBookingsHandler(w http.ResponseWriter, r *http.Request) {
	enableCors(&w)
	var err error

	defer func() {
		if err != nil {
			log.Printf("getCarBookingsHandler error - err: %v\nurl:%v\ncookies: %+v\n", err, r.URL, r.Cookies())
			w.WriteHeader(http.StatusInternalServerError)
		}
	}()

	if r.Method != http.MethodGet {
		err = errors.New("incorrect http method")
		return
	}

	start := r.FormValue("start")
	end := r.FormValue("end")
	carID := r.FormValue("carid")
	if start == "" || end == "" || carID == "" {
		err = errors.New("incorrect parameters")
		return
	}

	timeRanges, err := carService.GetCarBookings(start, end, carID)
	if err != nil {
		return
	}

	var buffer bytes.Buffer
	encoder := json.NewEncoder(&buffer)
	encoder.Encode(timeRanges)
	w.Write(buffer.Bytes())
}

//BOOKING Service

func createBookingHandler(w http.ResponseWriter, r *http.Request) {
	enableCors(&w)
	var err error

	defer func() {
		if err != nil {
			log.Printf("CreateBookingHandler error - err: %v\nurl:%v\ncookies: %+v\n", err, r.URL, r.Cookies())
			w.WriteHeader(http.StatusInternalServerError)
		}
	}()

	if r.Method != http.MethodGet {
		err = errors.New("incorrect http method")
		return
	}

	token, err := r.Cookie("session-token")
	if err != nil {
		return
	}

	start := r.FormValue("start")
	end := r.FormValue("end")
	carID := r.FormValue("carid")
	late := r.FormValue("late")
	extension := r.FormValue("extension")
	accessories := r.FormValue("accessories")
	days := r.FormValue("days")

	if start == "" || end == "" || carID == "" || late == "" || days == "" || len(token.Value) == 0 {
		err = errors.New("incorrect parameters")
		return
	}

	booking, err := bookingService.Create(token.Value, start, end, carID, late, extension, accessories, days)
	if err != nil {
		return
	}

	var buffer bytes.Buffer
	encoder := json.NewEncoder(&buffer)
	encoder.Encode(&booking)
	w.Write(buffer.Bytes())
}

func makePaymentHandler(w http.ResponseWriter, r *http.Request) {
	enableCors(&w)
	var err error

	defer func() {
		if err != nil {
			log.Printf("makePaymentHandler error - err: %v\nurl:%v\ncookies: %+v\n", err, r.URL, r.Cookies())
			w.WriteHeader(http.StatusInternalServerError)
		}
	}()

	if r.Method != http.MethodGet {
		err = errors.New("incorrect http method")
		return
	}

	token, err := r.Cookie("session-token")
	if err != nil {
		return
	}

	bookingID := r.FormValue("bookingID")

	if bookingID == "" {
		err = errors.New("incorrect parameters")
		return
	}

	err = bookingService.MakePayment(token.Value, bookingID)
	if err != nil {
		return
	}

	w.WriteHeader(200)
}

func cancelBookingHandler(w http.ResponseWriter, r *http.Request) {
	enableCors(&w)
	var err error

	defer func() {
		if err != nil {
			log.Printf("cancelBookingHandler error - err: %v\nurl:%v\ncookies: %+v\n", err, r.URL, r.Cookies())
			w.WriteHeader(http.StatusInternalServerError)
		}
	}()

	if r.Method != http.MethodGet {
		err = errors.New("incorrect http method")
		return
	}

	token, err := r.Cookie("session-token")
	if err != nil {
		return
	}

	bookingID := r.FormValue("bookingID")

	if bookingID == "" {
		err = errors.New("incorrect parameters")
		return
	}

	err = bookingService.CancelBooking(token.Value, bookingID)
	if err != nil {
		return
	}

	w.WriteHeader(200)
}

func historyBookingHandler(w http.ResponseWriter, r *http.Request) {
	enableCors(&w)
	var err error

	defer func() {
		if err != nil {
			log.Printf("historyBookingHandler error - err: %v\nurl:%v\ncookies: %+v\n", err, r.URL, r.Cookies())
			w.WriteHeader(http.StatusInternalServerError)
		}
	}()

	if r.Method != http.MethodGet {
		err = errors.New("incorrect http method")
		return
	}

	token, err := r.Cookie("session-token")
	if err != nil {
		return
	}

	bookingID := r.FormValue("bookingID")

	if bookingID == "" {
		err = errors.New("incorrect parameters")
		return
	}

	history, err := bookingService.GetHistory(token.Value, bookingID)
	if err != nil {
		return
	}

	var buffer bytes.Buffer
	encoder := json.NewEncoder(&buffer)
	encoder.Encode(&history)
	w.Write(buffer.Bytes())
}

func editBookingHandler(w http.ResponseWriter, r *http.Request) {
	enableCors(&w)
	var err error

	defer func() {
		if err != nil {
			log.Printf("editBookingHandler error - err: %v\nurl:%v\ncookies: %+v\n", err, r.URL, r.Cookies())
			w.WriteHeader(http.StatusInternalServerError)
		}
	}()

	if r.Method != http.MethodGet {
		err = errors.New("incorrect http method")
		return
	}

	token, err := r.Cookie("session-token")
	if err != nil {
		return
	}

	bookingID := r.FormValue("bookingID")
	remove := r.FormValue("remove")
	add := r.FormValue("add")
	lateReturn := r.FormValue("lateReturn")
	extension := r.FormValue("extension")

	if bookingID == "" || lateReturn == "" || extension == "" {
		err = errors.New("incorrect parameters")
		return
	}

	err = bookingService.EditBooking(token.Value, bookingID, remove, add, lateReturn, extension)
	if err != nil {
		return
	}

	w.WriteHeader(200)
}

func getUsersBookingsHandler(w http.ResponseWriter, r *http.Request) {
	enableCors(&w)
	var err error

	defer func() {
		if err != nil {
			log.Printf("getUsersBookingsHandler error - err: %v\nurl:%v\ncookies: %+v\n", err, r.URL, r.Cookies())
			w.WriteHeader(http.StatusInternalServerError)
		}
	}()

	if r.Method != http.MethodGet {
		err = errors.New("incorrect http method")
		return
	}

	token, err := r.Cookie("session-token")
	if err != nil {
		return
	}

	if len(token.Value) == 0 {
		err = errors.New("incorrect parameters")
		return
	}
	bookings, err := bookingService.GetUsersBookings(token.Value)
	if err != nil {
		return
	}

	var buffer bytes.Buffer
	encoder := json.NewEncoder(&buffer)
	encoder.Encode(&bookings)
	w.Write(buffer.Bytes())
}

func getBookingStatsHandler(w http.ResponseWriter, r *http.Request) {
	enableCors(&w)
	var err error

	defer func() {
		if err != nil {
			log.Printf("getBookingStatsHandler error - err: %v\nurl:%v\ncookies: %+v\n", err, r.URL, r.Cookies())
			w.WriteHeader(http.StatusInternalServerError)
		}
	}()

	if r.Method != http.MethodGet {
		err = errors.New("incorrect http method")
		return
	}

	token, err := r.Cookie("session-token")
	if err != nil {
		return
	}

	stats, err := adminService.GetBookingStats(token.Value)
	if err != nil {
		return
	}

	var buffer bytes.Buffer
	encoder := json.NewEncoder(&buffer)
	encoder.Encode(&stats)
	w.Write(buffer.Bytes())
}

func getUserStatsHandler(w http.ResponseWriter, r *http.Request) {
	enableCors(&w)
	var err error

	defer func() {
		if err != nil {
			log.Printf("getUserStatsHandler error - err: %v\nurl:%v\ncookies: %+v\n", err, r.URL, r.Cookies())
			w.WriteHeader(http.StatusInternalServerError)
		}
	}()

	if r.Method != http.MethodGet {
		err = errors.New("incorrect http method")
		return
	}

	token, err := r.Cookie("session-token")
	if err != nil {
		return
	}

	stats, err := adminService.GetUserStats(token.Value)
	if err != nil {
		return
	}

	var buffer bytes.Buffer
	encoder := json.NewEncoder(&buffer)
	encoder.Encode(&stats)
	w.Write(buffer.Bytes())
}

func getBookingStatusesHandler(w http.ResponseWriter, r *http.Request) {
	enableCors(&w)
	var err error

	defer func() {
		if err != nil {
			log.Printf("getBookingStatusesHandler error - err: %v\nurl:%v\ncookies: %+v\n", err, r.URL, r.Cookies())
			w.WriteHeader(http.StatusInternalServerError)
		}
	}()

	if r.Method != http.MethodGet {
		err = errors.New("incorrect http method")
		return
	}

	token, err := r.Cookie("session-token")
	if err != nil {
		return
	}

	status, err := adminService.GetBookingStatuses(token.Value)
	if err != nil {
		return
	}

	var buffer bytes.Buffer
	encoder := json.NewEncoder(&buffer)
	encoder.Encode(&status)
	w.Write(buffer.Bytes())
}

func getQueryingRefundBookings(w http.ResponseWriter, r *http.Request) {
	enableCors(&w)
	var err error

	defer func() {
		if err != nil {
			log.Printf("getQueryingRefundBookings error - err: %v\nurl:%v\ncookies: %+v\n", err, r.URL, r.Cookies())
			w.WriteHeader(http.StatusInternalServerError)
		}
	}()

	if r.Method != http.MethodGet {
		err = errors.New("incorrect http method")
		return
	}

	token, err := r.Cookie("session-token")
	if err != nil {
		return
	}

	bookings, err := adminService.GetQueryingRefundBookings(token.Value)
	if err != nil {
		return
	}

	var buffer bytes.Buffer
	encoder := json.NewEncoder(&buffer)
	encoder.Encode(&bookings)
	w.Write(buffer.Bytes())
}

func getAwaitingBookingsHandler(w http.ResponseWriter, r *http.Request) {
	enableCors(&w)
	var err error

	defer func() {
		if err != nil {
			log.Printf("getAwaitingBookingsHandler error - err: %v\nurl:%v\ncookies: %+v\n", err, r.URL, r.Cookies())
			w.WriteHeader(http.StatusInternalServerError)
		}
	}()

	if r.Method != http.MethodGet {
		err = errors.New("incorrect http method")
		return
	}

	token, err := r.Cookie("session-token")
	if err != nil {
		return
	}

	status := r.FormValue("status")
	limit := r.FormValue("limit")
	if status == "" || limit == "" {
		err = errors.New("incorrect parameters")
		return
	}

	bookings, err := adminService.GetAwaitingBookings(token.Value, status, limit)
	if err != nil {
		return
	}

	var buffer bytes.Buffer
	encoder := json.NewEncoder(&buffer)
	encoder.Encode(&bookings)
	w.Write(buffer.Bytes())
}

func processRefundHandler(w http.ResponseWriter, r *http.Request) {
	enableCors(&w)
	var err error

	defer func() {
		if err != nil {
			log.Printf("processRefundHandler error - err: %v\nurl:%v\ncookies: %+v\n", err, r.URL, r.Cookies())
			w.WriteHeader(http.StatusInternalServerError)
		}
	}()

	if r.Method != http.MethodGet {
		err = errors.New("incorrect http method")
		return
	}

	token, err := r.Cookie("session-token")
	if err != nil {
		return
	}

	accept := r.FormValue("accept")
	bookingID := r.FormValue("bookingID")
	reason := r.FormValue("reason")
	if bookingID == "" || accept == "" {
		err = errors.New("incorrect parameters")
		return
	}

	err = adminService.ProcessRefundHandler(token.Value, bookingID, accept, reason)
	if err != nil {
		return
	}

	w.WriteHeader(200)
}

func processExtraPaymentHandler(w http.ResponseWriter, r *http.Request) {
	enableCors(&w)
	var err error

	defer func() {
		if err != nil {
			log.Printf("processExtraPaymentHandler error - err: %v\nurl:%v\ncookies: %+v\n", err, r.URL, r.Cookies())
			w.WriteHeader(http.StatusInternalServerError)
		}
	}()

	if r.Method != http.MethodGet {
		err = errors.New("incorrect http method")
		return
	}

	token, err := r.Cookie("session-token")
	if err != nil {
		return
	}

	bookingID := r.FormValue("bookingID")
	if bookingID == "" {
		err = errors.New("incorrect parameters")
		return
	}

	err = adminService.ProcessExtraPayment(token.Value, bookingID)
	if err != nil {
		return
	}

	w.WriteHeader(200)
}

func progressBookingHandler(w http.ResponseWriter, r *http.Request) {
	enableCors(&w)
	var err error

	defer func() {
		if err != nil {
			log.Printf("progressBookingHandler error - err: %v\nurl:%v\ncookies: %+v\n", err, r.URL, r.Cookies())
			w.WriteHeader(http.StatusInternalServerError)
		}
	}()

	if r.Method != http.MethodGet {
		err = errors.New("incorrect http method")
		return
	}

	token, err := r.Cookie("session-token")
	if err != nil {
		return
	}

	bookingID := r.FormValue("bookingID")
	if bookingID == "" {
		err = errors.New("incorrect parameters")
		return
	}

	err = adminService.ProgressBooking(token.Value, bookingID)
	if err != nil {
		return
	}

	w.WriteHeader(200)
}

func getAdminBookingHandler(w http.ResponseWriter, r *http.Request) {
	enableCors(&w)
	var err error

	defer func() {
		if err != nil {
			log.Printf("getAdminBookingHandler error - err: %v\nurl:%v\ncookies: %+v\n", err, r.URL, r.Cookies())
			w.WriteHeader(http.StatusInternalServerError)
		}
	}()

	if r.Method != http.MethodGet {
		err = errors.New("incorrect http method")
		return
	}

	token, err := r.Cookie("session-token")
	if err != nil {
		return
	}

	bookingID := r.FormValue("bookingID")
	if bookingID == "" {
		err = errors.New("incorrect parameters")
		return
	}

	adminBooking, err := adminService.GetBooking(token.Value, bookingID)
	if err != nil {
		return
	}

	var buffer bytes.Buffer
	encoder := json.NewEncoder(&buffer)
	encoder.Encode(&adminBooking)
	w.Write(buffer.Bytes())
}

func getSearchedBookingsHandler(w http.ResponseWriter, r *http.Request) {
	enableCors(&w)
	var err error

	defer func() {
		if err != nil {
			log.Printf("getSearchedBookingsHandler error - err: %v\nurl:%v\ncookies: %+v\n", err, r.URL, r.Cookies())
			w.WriteHeader(http.StatusInternalServerError)
		}
	}()

	if r.Method != http.MethodGet {
		err = errors.New("incorrect http method")
		return
	}

	token, err := r.Cookie("session-token")
	if err != nil {
		return
	}

	userSearch := r.FormValue("userSearch")
	bookingSearch := r.FormValue("bookingSearch")
	statusFilter := r.FormValue("statusFilter")

	stats, err := adminService.GetSearchedBookings(token.Value, userSearch, bookingSearch, statusFilter)
	if err != nil {
		return
	}

	var buffer bytes.Buffer
	encoder := json.NewEncoder(&buffer)
	encoder.Encode(&stats)
	w.Write(buffer.Bytes())
}

func getAccessoryStatsHandler(w http.ResponseWriter, r *http.Request) {
	enableCors(&w)
	var err error

	defer func() {
		if err != nil {
			log.Printf("getAccessoryStatsHandler error - err: %v\nurl:%v\ncookies: %+v\n", err, r.URL, r.Cookies())
			w.WriteHeader(http.StatusInternalServerError)
		}
	}()

	if r.Method != http.MethodGet {
		err = errors.New("incorrect http method")
		return
	}

	token, err := r.Cookie("session-token")
	if err != nil {
		return
	}

	stats, err := adminService.GetAccessoryStats(token.Value)
	if err != nil {
		return
	}

	var buffer bytes.Buffer
	encoder := json.NewEncoder(&buffer)
	encoder.Encode(&stats)
	w.Write(buffer.Bytes())
}

func getCarStatsHandler(w http.ResponseWriter, r *http.Request) {
	enableCors(&w)
	var err error

	defer func() {
		if err != nil {
			log.Printf("getCarStatsHandler error - err: %v\nurl:%v\ncookies: %+v\n", err, r.URL, r.Cookies())
			w.WriteHeader(http.StatusInternalServerError)
		}
	}()

	if r.Method != http.MethodGet {
		err = errors.New("incorrect http method")
		return
	}

	token, err := r.Cookie("session-token")
	if err != nil {
		return
	}

	stats, err := adminService.GetCarStats(token.Value)
	if err != nil {
		return
	}

	var buffer bytes.Buffer
	encoder := json.NewEncoder(&buffer)
	encoder.Encode(&stats)
	w.Write(buffer.Bytes())
}

func enableCors(w *http.ResponseWriter) {
	//(*w).Header().Set("Access-Control-Allow-Origin", "*")
	(*w).Header().Set("Access-Control-Allow-Credentials", "true")
	(*w).Header().Set("Access-Control-Allow-Origin", "http://localhost:4200")
}
