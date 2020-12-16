package main

import (
	"bytes"
	"carHiringWebsite/carService"
	"carHiringWebsite/db"
	"carHiringWebsite/response"
	"carHiringWebsite/userService"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os/exec"
	"path/filepath"
	"strconv"
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
	http.HandleFunc("/userService/register", RegistrationHandler)
	http.HandleFunc("/userService/login", LoginHandler)
	http.HandleFunc("/userService/logout", LogoutHandler)
	http.HandleFunc("/userService/sessionCheck", SessionCheckHandler)

	http.HandleFunc("/carService/getAll", GetAllCarsHandler)
	http.HandleFunc("/carService/get", GetCarHandler)

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
	fileServe := http.FileServer(http.Dir("./public"))

	if len(filepath.Ext(r.RequestURI)) == 0 {
		r.RequestURI = "/"
		r.URL.Path = "/"
	}
	fileServe.ServeHTTP(w, r)
}

func RegistrationHandler(w http.ResponseWriter, r *http.Request) {
	enableCors(&w)
	var err error

	defer func() {
		if err != nil {
			fmt.Printf("RegistrationHandler error - err: %v \ncookies: %+v\n", err, r.Cookies())
			log.Printf("RegistrationHandler error - err: %v \ncookies: %+v\n", err, r.Cookies())
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

func LoginHandler(w http.ResponseWriter, r *http.Request) {
	enableCors(&w)
	var err error

	defer func() {
		if err != nil {
			fmt.Printf("LoginHandler error - err: %v \ncookies: %+v\n", err, r.Cookies())
			log.Printf("LoginHandler error - err: %v \ncookies: %+v\n", err, r.Cookies())
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

func LogoutHandler(w http.ResponseWriter, r *http.Request) {
	enableCors(&w)
	var err error

	defer func() {
		if err != nil {
			fmt.Printf("LogoutHandler error - err: %v \ncookies: %+v\n", err, r.Cookies())
			log.Printf("LogoutHandler error - err: %v \ncookies: %+v\n", err, r.Cookies())
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

func SessionCheckHandler(w http.ResponseWriter, r *http.Request) {
	enableCors(&w)
	var err error

	defer func() {
		if err != nil {
			fmt.Printf("SessionCheckHandler error - err: %v \ncookies: %+v\n", err, r.Cookies())
			log.Printf("SessionCheckHandler error - err: %v \ncookies: %+v\n", err, r.Cookies())
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

func GetAllCarsHandler(w http.ResponseWriter, r *http.Request) {
	enableCors(&w)
	var err error

	defer func() {
		if err != nil {
			fmt.Printf("GetAllCarsHandler error - err: %v \ncookies: %+v\n", err, r.Cookies())
			log.Printf("GetAllCarsHandler error - err: %v \ncookies: %+v\n", err, r.Cookies())
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

func GetCarHandler(w http.ResponseWriter, r *http.Request) {
	enableCors(&w)
	var err error

	defer func() {
		if err != nil {
			fmt.Printf("GetCarsHandler error - err: %v \ncookies: %+v\n", err, r.Cookies())
			log.Printf("GetCarsHandler error - err: %v \ncookies: %+v\n", err, r.Cookies())
			w.WriteHeader(http.StatusInternalServerError)
		}
	}()

	if r.Method != http.MethodGet {
		err = errors.New("incorrect http method")
		return
	}

	id := r.FormValue("id")

	cars, err := carService.GetCar(id)
	if err != nil {
		return
	}

	var buffer bytes.Buffer
	encoder := json.NewEncoder(&buffer)
	encoder.Encode(cars)
	w.Write(buffer.Bytes())
}

func enableCors(w *http.ResponseWriter) {
	//(*w).Header().Set("Access-Control-Allow-Origin", "*")
	(*w).Header().Set("Access-Control-Allow-Credentials", "true")
	(*w).Header().Set("Access-Control-Allow-Origin", "http://localhost:4200")

}
