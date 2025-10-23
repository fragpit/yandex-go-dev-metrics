//go:build dev
// +build dev

package main

import (
	"log"
	"net/http"
	_ "net/http/pprof"
)

func init() {
	go func() {
		log.Println(
			"starting pprof server on http://localhost:6060/debug/pprof/",
		)
		if err := http.ListenAndServe(
			"localhost:6060",
			nil,
		); err != nil {
			log.Printf(
				"pprof server error: %v\n",
				err,
			)
		}
	}()
}
