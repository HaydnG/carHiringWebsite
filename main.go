package main

import (
	"bytes"
	"carHiringWebsite/ABIDataProvider"
	"carHiringWebsite/DVLADataProvider"
	"carHiringWebsite/VehicleScanner"
	"carHiringWebsite/data"
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
	"io"
	"log"
	"net/http"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

func main() {
	var err error

	buildAll := flag.Bool("buildall", false, "tells the webserver to rebuild the frontEnd")

	db.User = flag.String("user", "root", "the database user to use")
	db.Pass = flag.String("pass", "pass", "the database pass to use")
	db.Address = flag.String("address", "localhost:3306", "the database address to use")
	db.Schema = flag.String("schema", "carrental", "the schema user to use")

	port := flag.String("port", "8080", "the port the server will run on")

	flag.Parse()

	// Build front-end if param specified
	if *buildAll {
		fmt.Println("Started Building Frontend...")
		cmd := exec.Command("ng", "build", "--prod", "--output-path=../public")
		cmd.Dir = "./carHiringWebsite-Frontend"

		output, err := cmd.Output()
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println(string(output))
		fmt.Println("Finished building")
	}

	// Initiate db connection
	err = db.InitDB()
	if err != nil {
		log.Fatal(err)
	}

	err = ABIDataProvider.InitProvider()
	if err != nil {
		log.Fatal(err)
	}

	DVLADataProvider.InitProvider()

	//Serve the website files generated from the build-job in public

	http.HandleFunc("/", SiteHandler)

	//Service endpoints
	http.HandleFunc("/userService/register", registrationHandler)
	http.HandleFunc("/userService/login", loginHandler)
	http.HandleFunc("/userService/logout", logoutHandler)
	http.HandleFunc("/userService/sessionCheck", sessionCheckHandler)
	http.HandleFunc("/userService/get", getUserHandler)
	http.HandleFunc("/userService/edit", editUserHandler)

	http.HandleFunc("/carService/getAll", getAllCarsHandler)
	http.HandleFunc("/carService/get", getCarHandler)
	http.HandleFunc("/carService/getAccessories", getCarAccessoriesHandler)
	http.HandleFunc("/carService/getBookings", getCarBookingsHandler)
	http.HandleFunc("/carService/getCarAttributes", getCarAttributesHandler)

	http.HandleFunc("/carService/testCarData", testCarData)
	http.HandleFunc("/carService/testInsure", testInsure)
	http.HandleFunc("/carService/testDVLA", testDVLA)

	http.HandleFunc("/bookingService/create", createBookingHandler)
	http.HandleFunc("/bookingService/makePayment", makePaymentHandler)
	http.HandleFunc("/bookingService/getUserBookings", getUsersBookingsHandler)
	http.HandleFunc("/bookingService/getExtensionDays", getExtensionDaysHandler)
	http.HandleFunc("/bookingService/cancelBooking", cancelBookingHandler)
	http.HandleFunc("/bookingService/history", historyBookingHandler)
	http.HandleFunc("/bookingService/editBooking", editBookingHandler)
	http.HandleFunc("/bookingService/extendBooking", extendBookingHandler)
	http.HandleFunc("/bookingService/payExtension", payExtensionHandler)

	http.HandleFunc("/adminService/getBookingStats", getBookingStatsHandler)
	http.HandleFunc("/adminService/getUserStats", getUserStatsHandler)
	http.HandleFunc("/adminService/getUsers", getUsersHandler)
	http.HandleFunc("/adminService/getCarStats", getCarStatsHandler)
	http.HandleFunc("/adminService/getAccessoryStats", getAccessoryStatsHandler)
	http.HandleFunc("/adminService/getSearchedBookings", getSearchedBookingsHandler)
	http.HandleFunc("/adminService/getAwaitingBookings", getAwaitingBookingsHandler)
	http.HandleFunc("/adminService/getQueryingRefundBookings", getQueryingRefundBookings)
	http.HandleFunc("/adminService/getBookingStatuses", getBookingStatusesHandler)
	http.HandleFunc("/adminService/getBooking", getAdminBookingHandler)
	http.HandleFunc("/adminService/getUser", getAdminUserHandler)
	http.HandleFunc("/adminService/progressBooking", progressBookingHandler)
	http.HandleFunc("/adminService/processExtraPayment", processExtraPaymentHandler)
	http.HandleFunc("/adminService/processRefund", processRefundHandler)
	http.HandleFunc("/adminService/getCars", adminGetCarsHandler)
	http.HandleFunc("/adminService/createCar", createCarHandler)
	http.HandleFunc("/adminService/updateCar", updateCarHandler)
	http.HandleFunc("/adminService/setUser", setUserHandler)
	http.HandleFunc("/adminService/createUser", adminCreateUserHandler)
	http.HandleFunc("/adminService/verifyDriver", verifyDriverUserHandler)

	fmt.Printf("\nDB settings - User: %s, Pass: %s, Address: %s, Schema: %s\n\n", *db.User, *db.Pass, *db.Address, *db.Schema)
	fmt.Printf("Server Start Listening on port %s\n\n", *port)
	//Server operation
	err = http.ListenAndServe(":"+*port, nil)
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
			enableCors(&w)
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

func verifyDriverUserHandler(w http.ResponseWriter, r *http.Request) {
	enableCors(&w)
	var err error

	defer func() {
		if err == adminService.BlackListedDriver || err == DVLADataProvider.InvalidLicense || err == ABIDataProvider.FraudulentClaim {
			w.Write([]byte(err.Error()))
		} else if err != nil {
			log.Printf("verifyDriverUserHandler error - err: %v\nurl:%v\ncookies: %+v\n", err, r.URL, r.Cookies())
			w.WriteHeader(http.StatusInternalServerError)
		}
	}()

	if r.Method != http.MethodPost {
		err = errors.New("incorrect http method")
		return
	}

	lastname := r.FormValue("lastname")
	names := r.FormValue("names")
	dob := r.FormValue("dob")
	address := r.FormValue("address")
	postcode := r.FormValue("postcode")
	license := r.FormValue("license")
	bookingID := r.FormValue("bookingID")

	if len(dob) == 0 || len(address) == 0 || len(postcode) == 0 || len(lastname) == 0 || len(names) == 0 || len(license) == 0 || len(bookingID) == 0 {
		err = errors.New("incorrect parameters")
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

	var images data.ImageBundle
	err = json.NewDecoder(r.Body).Decode(&images)
	if err != nil {
		return
	}

	err = adminService.VerifyDriver(token.Value, dob, lastname, names, address, postcode, license, bookingID, images)
	if err != nil {
		return
	}

	data, err := adminService.GetBooking(token.Value, bookingID)
	if err != nil {
		return
	}

	var buffer bytes.Buffer
	encoder := json.NewEncoder(&buffer)
	encoder.Encode(&data)
	w.Write(buffer.Bytes())
}

func adminCreateUserHandler(w http.ResponseWriter, r *http.Request) {
	enableCors(&w)
	var err error

	defer func() {
		if err != nil {
			log.Printf("adminCreateUserHandler error - err: %v\nurl:%v\ncookies: %+v\n", err, r.URL, r.Cookies())
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

	token, err := r.Cookie("session-token")
	if err != nil {
		return
	}

	if len(token.Value) == 0 {
		err = errors.New("incorrect session")
		return
	}

	created, newUser, err := adminService.CreateUser(token.Value, email, password, firstname, names, dobString)
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

func editUserHandler(w http.ResponseWriter, r *http.Request) {
	enableCors(&w)
	var err error

	defer func() {
		if err != nil {
			log.Printf("editUserHandler error - err: %v\nurl:%v\ncookies: %+v\n", err, r.URL, r.Cookies())
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
	userID := r.FormValue("id")
	password := r.FormValue("password")
	oldPassword := r.FormValue("oldpassword")
	dobString := r.FormValue("dob")
	if len(userID) == 0 {
		err = errors.New("incorrect parameters")
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

	newUser, err := userService.EditUser(token.Value, userID, email, oldPassword, password, firstname, names, dobString)
	var buffer bytes.Buffer
	encoder := json.NewEncoder(&buffer)
	if err == userService.UsernameAlreadyExists {
		encoder.Encode(response.DuplicateUser)
		w.Write(buffer.Bytes())
		return
	}
	if err == userService.InvalidPassword {
		encoder.Encode(response.IncorrectPassword)
		w.Write(buffer.Bytes())
		return
	}
	if err != nil {
		return
	}

	encoder.Encode(&newUser)
	w.Write(buffer.Bytes())
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

	created, newUser, err := userService.CreateUser(email, password, firstname, names, dobString)
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

func getUserHandler(w http.ResponseWriter, r *http.Request) {
	enableCors(&w)
	var err error

	defer func() {
		if err != nil {
			log.Printf("getUserHandler error - err: %v\nurl:%v\ncookies: %+v\n", err, r.URL, r.Cookies())
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
	outputUser, err := userService.Get(token.Value)
	if err != nil {
		return
	}

	var buffer bytes.Buffer
	encoder := json.NewEncoder(&buffer)
	encoder.Encode(&outputUser)
	w.Write(buffer.Bytes())
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

func testDVLA(w http.ResponseWriter, r *http.Request) {
	enableCors(&w)
	var err error

	defer func() {
		if err != nil {
			log.Printf("testDVLA error - err: %v\nurl:%v\ncookies: %+v\n", err, r.URL, r.Cookies())
			w.WriteHeader(http.StatusInternalServerError)
		}
	}()

	if r.Method != http.MethodGet {
		err = errors.New("incorrect http method")
		return
	}

	number := r.FormValue("number")
	if number == "" {
		err = errors.New("incorrect parameters")
		return
	}

	data := DVLADataProvider.IsInvalidLicense(number)

	fmt.Println(data)

	var buffer bytes.Buffer
	encoder := json.NewEncoder(&buffer)
	encoder.Encode(data)
	w.Write(buffer.Bytes())
}

func testInsure(w http.ResponseWriter, r *http.Request) {
	enableCors(&w)
	var err error

	defer func() {
		if err != nil {
			log.Printf("testInsure error - err: %v\nurl:%v\ncookies: %+v\n", err, r.URL, r.Cookies())
			w.WriteHeader(http.StatusInternalServerError)
		}
	}()

	if r.Method != http.MethodGet {
		err = errors.New("incorrect http method")
		return
	}

	dob, err := time.Parse("02/01/2006", "04/06/1957")
	if err != nil {
		return
	}

	dataList, err := ABIDataProvider.HasFraudulentClaim("DUCK",
		"DONALD",
		"Duckulla Villa",
		"WM2 9DA",
		dob)
	if err != nil {
		return
	}

	fmt.Println(dataList)

	var buffer bytes.Buffer
	encoder := json.NewEncoder(&buffer)
	encoder.Encode(dataList)
	w.Write(buffer.Bytes())
}

func testCarData(w http.ResponseWriter, r *http.Request) {
	enableCors(&w)
	var err error

	defer func() {
		if err != nil {
			log.Printf("testCarData error - err: %v\nurl:%v\ncookies: %+v\n", err, r.URL, r.Cookies())
			w.WriteHeader(http.StatusInternalServerError)
		}
	}()

	if r.Method != http.MethodGet {
		err = errors.New("incorrect http method")
		return
	}

	details, err := VehicleScanner.GetVehiclePrice(5, 1, time.Now().Add(time.Hour*24), time.Now().Add(time.Hour*24*10))
	if err != nil {
		return
	}

	fmt.Println(details)

	var buffer bytes.Buffer
	encoder := json.NewEncoder(&buffer)
	encoder.Encode(details)
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
	fullDay := r.FormValue("fullday")
	accessories := r.FormValue("accessories")
	days := r.FormValue("days")

	if start == "" || end == "" || carID == "" || late == "" || days == "" || len(token.Value) == 0 {
		err = errors.New("incorrect parameters")
		return
	}

	booking, err := bookingService.Create(token.Value, start, end, carID, late, fullDay, accessories, days)
	if err != nil {
		return
	}

	var buffer bytes.Buffer
	encoder := json.NewEncoder(&buffer)
	encoder.Encode(&booking)
	w.Write(buffer.Bytes())
}

func payExtensionHandler(w http.ResponseWriter, r *http.Request) {
	enableCors(&w)
	var err error

	defer func() {
		if err != nil {
			log.Printf("payExtensionHandler error - err: %v\nurl:%v\ncookies: %+v\n", err, r.URL, r.Cookies())
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

	err = bookingService.MakeExtensionPayment(token.Value, bookingID)
	if err != nil {
		return
	}

	w.WriteHeader(200)
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

func extendBookingHandler(w http.ResponseWriter, r *http.Request) {
	enableCors(&w)
	var err error

	defer func() {
		if err != nil {
			log.Printf("extendBookingHandler error - err: %v\nurl:%v\ncookies: %+v\n", err, r.URL, r.Cookies())
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
	lateReturn := r.FormValue("lateReturn")
	fullDay := r.FormValue("fullDay")
	days := r.FormValue("days")

	if bookingID == "" || lateReturn == "" || fullDay == "" || days == "" {
		err = errors.New("incorrect parameters")
		return
	}

	err = bookingService.ExtendBooking(token.Value, bookingID, lateReturn, fullDay, days)
	if err != nil {
		return
	}

	w.WriteHeader(200)
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
	fullDay := r.FormValue("fullday")

	if bookingID == "" || lateReturn == "" || fullDay == "" {
		err = errors.New("incorrect parameters")
		return
	}

	err = bookingService.EditBooking(token.Value, bookingID, remove, add, lateReturn, fullDay)
	if err != nil {
		return
	}

	w.WriteHeader(200)
}

func getExtensionDaysHandler(w http.ResponseWriter, r *http.Request) {
	enableCors(&w)
	var err error

	defer func() {
		if err != nil {
			log.Printf("getExtensionDaysHandler error - err: %v\nurl:%v\ncookies: %+v\n", err, r.URL, r.Cookies())
			w.WriteHeader(http.StatusInternalServerError)
		}
	}()

	if r.Method != http.MethodGet {
		err = errors.New("incorrect http method")
		return
	}

	bookingID := r.FormValue("bookingID")
	if bookingID == "" {
		err = errors.New("incorrect parameters")
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
	response, err := bookingService.CountExtensionDays(token.Value, bookingID)
	if err != nil {
		return
	}

	var buffer bytes.Buffer
	encoder := json.NewEncoder(&buffer)
	encoder.Encode(&response)
	w.Write(buffer.Bytes())
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

func getUsersHandler(w http.ResponseWriter, r *http.Request) {
	enableCors(&w)
	var err error

	defer func() {
		if err != nil {
			log.Printf("getUsersHandler error - err: %v\nurl:%v\ncookies: %+v\n", err, r.URL, r.Cookies())
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

	search := r.FormValue("search")

	users, err := adminService.GetUsers(token.Value, search)
	if err != nil {
		return
	}

	var buffer bytes.Buffer
	encoder := json.NewEncoder(&buffer)
	encoder.Encode(&users)
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
	failed := r.FormValue("failed")
	if bookingID == "" || failed == "" {
		err = errors.New("incorrect parameters")
		return
	}

	err = adminService.ProgressBooking(token.Value, bookingID, failed)
	if err != nil {
		return
	}

	w.WriteHeader(200)
}
func getAdminUserHandler(w http.ResponseWriter, r *http.Request) {
	enableCors(&w)
	var err error

	defer func() {
		if err != nil {
			log.Printf("getAdminUserHandler error - err: %v\nurl:%v\ncookies: %+v\n", err, r.URL, r.Cookies())
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

	userID := r.FormValue("userID")
	if userID == "" {
		err = errors.New("incorrect parameters")
		return
	}

	userBundle, err := adminService.GetUser(token.Value, userID)
	if err != nil {
		return
	}

	var buffer bytes.Buffer
	encoder := json.NewEncoder(&buffer)
	encoder.Encode(&userBundle)
	w.Write(buffer.Bytes())
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

func setUserHandler(w http.ResponseWriter, r *http.Request) {
	enableCors(&w)
	var err error

	defer func() {
		if err != nil {
			log.Printf("setUserHandler error - err: %v\nurl:%v\ncookies: %+v\n", err, r.URL, r.Cookies())
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

	mode := r.FormValue("mode")
	value := r.FormValue("value")
	userID := r.FormValue("userID")

	if mode == "" || value == "" || userID == "" {
		err = errors.New("incorrect parameters")
		return
	}

	err = adminService.SetUser(token.Value, userID, mode, value)
	if err != nil {
		return
	}

	w.WriteHeader(200)
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

func createCarHandler(w http.ResponseWriter, r *http.Request) {
	enableCors(&w)
	var err error

	defer func() {
		if err != nil {
			log.Printf("createCarHandler error - err: %v\nurl:%v\ncookies: %+v\n", err, r.URL, r.Cookies())
			w.WriteHeader(http.StatusInternalServerError)
		}
	}()

	if r.Method != http.MethodPost {
		err = errors.New("incorrect http method")
		return
	}

	token, err := r.Cookie("session-token")
	if err != nil {
		return
	}

	fuelType := r.FormValue("fuelType")
	gearType := r.FormValue("gearType")
	carType := r.FormValue("carType")
	size := r.FormValue("size")
	colour := r.FormValue("colour")
	seats := r.FormValue("seats")
	price := r.FormValue("price")
	description := r.FormValue("description")
	disabled := r.FormValue("disabled")
	over25 := r.FormValue("over25")

	if over25 == "" || fuelType == "" || gearType == "" || carType == "" || size == "" || colour == "" || seats == "" || price == "" || description == "" || disabled == "" {
		err = errors.New("incorrect parameters")
		return
	}

	err = adminService.CreateCar(token.Value, fuelType, gearType, carType, size, colour, seats, price, disabled, over25, description, r.Body)
	if err != nil {
		return
	}

	w.WriteHeader(200)
}

func updateCarHandler(w http.ResponseWriter, r *http.Request) {
	enableCors(&w)
	var err error

	defer func() {
		if err != nil {
			log.Printf("createCarHandler error - err: %v\nurl:%v\ncookies: %+v\n", err, r.URL, r.Cookies())
			w.WriteHeader(http.StatusInternalServerError)
		}
	}()

	if r.Method != http.MethodPost {
		err = errors.New("incorrect http method")
		return
	}

	token, err := r.Cookie("session-token")
	if err != nil {
		return
	}

	fuelType := r.FormValue("fuelType")
	gearType := r.FormValue("gearType")
	carType := r.FormValue("carType")
	size := r.FormValue("size")
	colour := r.FormValue("colour")
	seats := r.FormValue("seats")
	price := r.FormValue("price")
	description := r.FormValue("description")
	disabled := r.FormValue("disabled")
	carID := r.FormValue("carID")
	over25 := r.FormValue("over25")

	if over25 == "" || carID == "" || fuelType == "" || gearType == "" || carType == "" || size == "" || colour == "" || seats == "" || price == "" || description == "" || disabled == "" {
		err = errors.New("incorrect parameters")
		return
	}

	var body io.ReadCloser

	if r.ContentLength <= 0 {
		body = nil
	} else {
		body = r.Body
	}

	err = adminService.UpdateCar(token.Value, carID, fuelType, gearType, carType, size, colour, seats, price, disabled, description, over25, body)
	if err != nil {
		return
	}

	w.WriteHeader(200)
}

func adminGetCarsHandler(w http.ResponseWriter, r *http.Request) {
	enableCors(&w)
	var err error

	defer func() {
		if err != nil {
			log.Printf("adminGetCarsHandler error - err: %v\nurl:%v\ncookies: %+v\n", err, r.URL, r.Cookies())
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

	fuelTypes := r.FormValue("fuelTypes")
	gearTypes := r.FormValue("gearTypes")
	carTypes := r.FormValue("carTypes")
	carSizes := r.FormValue("carSizes")
	colours := r.FormValue("colours")
	search := r.FormValue("search")

	stats, err := adminService.GetCars(token.Value, fuelTypes, gearTypes, carTypes, carSizes, colours, search)
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
