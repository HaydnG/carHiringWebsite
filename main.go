package main

import (
	"carHiringWebsite/db"
	"flag"
	"log"
	"net/http"
	"os/exec"
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




