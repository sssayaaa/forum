package helpers

import (
	"fmt"
	models "forum/internal/models"
	"html/template"
	"net/http"
)

func ErrorHandler(w http.ResponseWriter, errorNum int, errDetails error) {
	var resp models.ErrorResponse
	resp.ErrorNum = errorNum
	resp.ErrorMessage = http.StatusText(errorNum) + "\n" + errDetails.Error()
	w.WriteHeader(errorNum)

	temp, err := template.ParseFiles("./internal/web/templates/errors.html")
	if err != nil {
		fmt.Printf("Error parsing errors.html")
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
	}
	err = temp.Execute(w, resp)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
}
