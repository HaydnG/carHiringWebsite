package main

import (
	"carHiringWebsite/db"
	"carHiringWebsite/user"
	"errors"
	"flag"
	"log"
	"net/http"
	"os/exec"
	"strconv"
	"time"
)



func main(){
	var err error

	buildAll := flag.Bool("buildall", false, "tells the webserver to rebuild the frontEnd")

	flag.Parse()

	// Build front-end if param specified
	if *buildAll{
		cmd:= exec.Command("ng", "build", "--prod", "--output-path=../public")
		cmd.Dir = "./carHiringWebsite-Frontend"

		_, err := cmd.Output()
		if err !=nil{
			log.Fatal(err)
		}
	}

	// Initiate db connection
	err = db.InitDB()
	if err != nil{
		log.Fatal(err)
	}

	//Serve the website files generated from the build-job in public
	fileServe := http.FileServer(http.Dir("./public"))
	http.Handle("/", fileServe)

	http.HandleFunc("/user/register", RegistrationHandler)


	//Server operation
	err = http.ListenAndServe(":8080", nil)
	if err != nil {
		log.Fatal(err)
	}

	//Close db connection
	err = db.CloseDB()
	if err != nil{
		log.Fatal(err)
	}
}


func RegistrationHandler(w http.ResponseWriter, r *http.Request){
	var err error

	defer func() {
		if err != nil{
			log.Printf("RegistrationHandler error - err: %x", err)
			w.WriteHeader(http.StatusInternalServerError)
		}
 	}()

	if r.Method != http.MethodGet{
		err = errors.New("incorrect http method")
		return
	}

	name := r.FormValue("name")
	email := r.FormValue("email")
	password := r.FormValue("password")
	dobString := r.FormValue("dob")
	if len(email) == 0 || len(password) == 0 || len(dobString) == 0 || len(name) == 0{
		err = errors.New("incorrect parameters")
		return
	}

	dobUnix, err := strconv.ParseInt(dobString, 10, 64)
	if err != nil{
		err = errors.New("error reading DOB")
		return
	}

	dob := time.Unix(dobUnix, 0)

	if !user.ValidateCredentials(email, password){
		err = errors.New("user failed validation")
		return
	}

	newUser := user.CreateUser(email, password, name, dob)





}




