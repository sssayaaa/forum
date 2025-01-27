package handlers

import (
	"encoding/json"
	"errors"
	"forum/internal/models"
	helpers "forum/internal/web/handlers/helpers"
	"net/http"

	"golang.org/x/crypto/bcrypt"
)

func (h *Handler) RegistrationHandler(w http.ResponseWriter, r *http.Request) {
	registerPath := "internal/web/templates/registration.html"

	switch r.Method {
	case "GET":
		helpers.RenderTemplate(w, registerPath, nil)
		return
	case "POST":

		var validationErrors []string

		// Retrieve form values
		firstName := r.FormValue("firstName")
		secondName := r.FormValue("secondName")
		username := r.FormValue("username")
		email := r.FormValue("email")
		password := r.FormValue("password")

		// Input validation
		if firstName == "" {
			validationErrors = append(validationErrors, "First Name is required.")
		}
		if secondName == "" {
			validationErrors = append(validationErrors, "Second Name is required.")
		}
		if username == "" {
			validationErrors = append(validationErrors, "Username is required.")
		}
		if email == "" {
			validationErrors = append(validationErrors, "Email is required.")
		}
		if password == "" {
			validationErrors = append(validationErrors, "Password is required.")
		}

		if len(validationErrors) > 0 {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": false,
				"errors":  validationErrors,
			})
			return
		}

		// Encrypt password
		psw, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		var userRole string
		admin := r.FormValue("admin") == "on"
		if admin {
			userRole = "admin"
		} else {
			userRole = "user"
		}

		user := &models.User{
			FirstName:  firstName,
			SecondName: secondName,
			Username:   username,
			Email:      email,
			Password:   string(psw),
			Role:       userRole,
		}

		statusCode, id, err := h.service.UserServiceInterface.CreateUser(user)
		if err != nil {
			validationErrors = append(validationErrors, err.Error())

			// Use the statusCode to set appropriate HTTP response
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(statusCode)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": false,
				"errors":  validationErrors,
			})
			return
		}

		user.UserUserID = id

		// Respond with success JSON message
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
			"message": "Registration successful! Redirecting to login...",
		})
		return

	default:
		helpers.ErrorHandler(w, http.StatusMethodNotAllowed, errors.New("Error in Registration Handler"))
		return
	}
}

func (h *Handler) LoginHandler(w http.ResponseWriter, r *http.Request) {
	loginPath := "internal/web/templates/login.html"

	switch r.Method {
	case "GET":
		helpers.RenderTemplate(w, loginPath, nil)
		return
	case "POST":

		email := r.FormValue("email")
		password := r.FormValue("password")
		admin := r.FormValue("admin") == "on"

		var validationErrors []string

		if email == "" {
			validationErrors = append(validationErrors, "Email is required.")
		}
		if password == "" {
			validationErrors = append(validationErrors, "Password is required.")
		}

		if len(validationErrors) > 0 {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": false,
				"errors":  validationErrors,
			})
			return
		}

		session, err := h.service.UserServiceInterface.Login(email, password, admin)
		if err != nil {
			validationErrors = append(validationErrors, err.Error())
		}

		if len(validationErrors) > 0 {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": false,
				"errors":  validationErrors,
			})
			return
		}

		helpers.SessionCookieSet(w, session.Token, session.ExpTime)

		// if admin {
		// 	http.Redirect(w, r, "/admin_page", http.StatusSeeOther)
		// 	return
		// } else {
		// 	http.Redirect(w, r, "/", http.StatusSeeOther)
		// 	return
		// }

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
			"message": "Login successful! Redirecting...",
		})
		return

	default:
		helpers.ErrorHandler(w, http.StatusUnauthorized, errors.New("Error in Login Handler"))
		return
	}
}

func (h *Handler) LogoutHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		cookie := helpers.SessionCookieGet(r)
		if cookie == nil {
			helpers.ErrorHandler(w, http.StatusUnauthorized, errors.New("Conversion of postID failed"))
			return
		}

		//??
		if err := h.service.UserServiceInterface.Logout(cookie.Value); err != nil {
			helpers.ErrorHandler(w, http.StatusInternalServerError, err)
			return
		} else {
			helpers.SessionCookieExpire(w)
			http.Redirect(w, r, "/", http.StatusFound)
		}
	default:
		helpers.ErrorHandler(w, http.StatusUnauthorized, errors.New("Error in Logout Handler"))
		return
	}
}
