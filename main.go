package main

import (
	"flag"
	"fmt"
	"log"
"net/http"
	"os/exec"
)



func main(){


	buildAll := flag.Bool("buildall", false, "tells the webserver to rebuild the frontEnd")

	flag.Parse()

	if *buildAll{
		cmd:= exec.Command("ng", "build", "--prod", "--output-path=../public")
		cmd.Dir = "./carHiringWebsite-Frontend"

		a, err := cmd.Output()
		if err !=nil{
			log.Fatal(err)
		}
		fmt.Println(string(a))
	}

	//Serve the website files
	fileServe := http.FileServer(http.Dir("./public"))
	http.Handle("/", fileServe)


	//Server operation
	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		log.Fatal(err)
	}
}




