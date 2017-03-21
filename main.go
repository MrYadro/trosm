package main

import (
	"log"
	"net/http"

	"github.com/MrYadro/trosm/scheme"
)

func main() {
	http.Handle("/", http.FileServer(http.Dir("images/")))
	http.Handle("/scheme", http.HandlerFunc(scheme.MTrans))
	err := http.ListenAndServe(":2003", nil)
	if err != nil {
		log.Fatal("ListenAndServe:", err)
	}
}

// [out:json];
// rel["network"="berdskpt"]["ref"="21"]["operator"=""];
// node(r) -> .stops;
// (
//   node(around.stops:500.0)["railway"="station"];
//   node(around.stops:500.0)["public_transport"="stop_position"];
// );
// (
//   ._;
//   rel(bn);
// );
// out body;
