package Response

import (
	"fmt"
	"log"
	"net/http"
)

func SendJson(w http.ResponseWriter, r *http.Request, b []byte) {
	w.Header().Add("Content-Type", "application/json")
	_, err := fmt.Fprintf(w, string(b))
	if err != nil {
		log.Printf("[ERROR][%s] %s\n", r.RemoteAddr, err)
	}
}
