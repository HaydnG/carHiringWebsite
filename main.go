package main

import (
	"bytes"
	"carHiringWebsite/db"
	"carHiringWebsite/response"
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

	http.HandleFunc("/carService/getAll", setAllCarsHandler)
	http.HandleFunc("/carService/get", getCarHandler)
	http.HandleFunc("/carService/getAccessories", getCarAccessoriesHandler)
	http.HandleFunc("/carService/getBookings", getCarBookingsHandler)

	http.HandleFunc("/bookingService/create", createBookingHandler)
	http.HandleFunc("/bookingService/makePayment", makePaymentHandler)
	http.HandleFunc("/bookingService/getUserBookings", getUsersBookingsHandler)
	http.HandleFunc("/bookingService/cancelBooking", cancelBookingHandler)

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
			fmt.Printf("registrationHandler error - err: %v\nurl:%v\ncookies: %+v\n", err, r.URL, r.Cookies())
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
			fmt.Printf("loginHandler error - err: %v\nurl:%v\ncookies: %+v\n", err, r.URL, r.Cookies())
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
			fmt.Printf("logoutHandler error - err: %v\nurl:%v\ncookies: %+v\n", err, r.URL, r.Cookies())
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
			fmt.Printf("sessionCheckHandler error - err: %v\nurl:%v\ncookies: %+v\n", err, r.URL, r.Cookies())
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

func setAllCarsHandler(w http.ResponseWriter, r *http.Request) {
	enableCors(&w)
	var err error

	defer func() {
		if err != nil {
			fmt.Printf("setAllCarsHandler error - err: %v\nurl:%v\ncookies: %+v\n", err, r.URL, r.Cookies())
			log.Printf("setAllCarsHandler error - err: %v\nurl:%v\ncookies: %+v\n", err, r.URL, r.Cookies())
			w.WriteHeader(http.StatusInternalServerError)
		}
	}()

	if r.Method != http.MethodGet {
		err = errors.New("incorrect http method")
		return
	}

	cars, err := carService.GetAllCars()
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
			fmt.Printf("GetCarsHandler error - err: %v\nurl:%v\ncookies: %+v\n", err, r.URL, r.Cookies())
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
			fmt.Printf("getCarAccessoriesHandler error - err: %v\nurl:%v\ncookies: %+v\n", err, r.URL, r.Cookies())
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

func getCarBookingsHandler(w http.ResponseWriter, r *http.Request) {
	enableCors(&w)
	var err error

	defer func() {
		if err != nil {
			fmt.Printf("getCarBookingsHandler error - err: %v\nurl:%v\ncookies: %+v\n", err, r.URL, r.Cookies())
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
			fmt.Printf("CreateBookingHandler error - err: %v\nurl:%v\ncookies: %+v\n", err, r.URL, r.Cookies())
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
			fmt.Printf("makePaymentHandler error - err: %v\nurl:%v\ncookies: %+v\n", err, r.URL, r.Cookies())
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
			fmt.Printf("cancelBookingHandler error - err: %v\nurl:%v\ncookies: %+v\n", err, r.URL, r.Cookies())
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

func getUsersBookingsHandler(w http.ResponseWriter, r *http.Request) {
	enableCors(&w)
	var err error

	defer func() {
		if err != nil {
			fmt.Printf("getUsersBookingsHandler error - err: %v\nurl:%v\ncookies: %+v\n", err, r.URL, r.Cookies())
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

func enableCors(w *http.ResponseWriter) {
	//(*w).Header().Set("Access-Control-Allow-Origin", "*")
	(*w).Header().Set("Access-Control-Allow-Credentials", "true")
	(*w).Header().Set("Access-Control-Allow-Origin", "http://localhost:4200")
}
