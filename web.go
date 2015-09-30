/**
*
*kexue shang wang
*
*use some gopee code: github.com/madhurjain/gopee
*
 */
package main

import (
	"flag"
	"fmt"
	"github.com/hidu/kx-proxy/handler"
	"log"
	"net/http"
	"os"
)

var addr = flag.String("addr", ":8085", "listen addr,eg :8085")

func init() {
	flag.BoolVar(&handler.BodyStreamEnc, "enc", false, "encrypts the body stream")
}

func main() {
	flag.Parse()

	var httpHost = os.Getenv("HOST")
	var httpPort = os.Getenv("PORT")

	laddr := httpHost + ":" + httpPort

	if httpPort == "" {
		laddr = *addr
	}
	log.Println("bodyStreamEnc:", handler.BodyStreamEnc)

	if len(laddr) < 2 {
		fmt.Println("listening addr [", laddr, "] is wrong")
		os.Exit(1)
	}

	fmt.Printf("kx-proxy listening on :%s\n", laddr)
	err := http.ListenAndServe(laddr, nil)
	fmt.Println("exit with err:", err)
}
