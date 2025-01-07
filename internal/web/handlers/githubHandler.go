package handlers

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"forum/internal/models"
	"forum/internal/web/handlers/helpers"
	"io/ioutil"
	"net/http"
)

func (h *Handler) GithubAuthHandler(w http.ResponseWriter, r *http.Request) {
	// Ensure the redirect URI is correct
	url := fmt.Sprintf("%s?client_id=%s&redirect_uri=%s&scope=offline&prompt=consent", models.GitHubAuthURL, models.GitHubClientID, models.GitHubRedirectURL)
	http.Redirect(w, r, url, http.StatusTemporaryRedirect)
}

func (h *Handler) GithubCallback(w http.ResponseWriter, r *http.Request) {
	// log.Println("Received callback at /github/callback")

	code := r.URL.Query().Get("code") // temporary token given by Github
	if code == "" {
		helpers.ErrorHandler(w, http.StatusUnauthorized, errors.New("Temporary token is invalid"))
		return
	}

	// Get OAuth token using the code
	tokenRes, err := getGithubOauthToken(code)
	if err != nil {
		helpers.ErrorHandler(w, http.StatusBadGateway, fmt.Errorf("Error retrieving token: %v", err))
		return
	}

	// Get user data from GitHub
	githubData, err := getGithubData(tokenRes.AccessToken)
	if err != nil {
		helpers.ErrorHandler(w, http.StatusBadGateway, fmt.Errorf("Error retrieving user data: %v", err))
		return
	}

	// Parse the user data
	userData, err := getUserData(githubData)
	if err != nil {
		helpers.ErrorHandler(w, http.StatusBadGateway, fmt.Errorf("Error parsing GitHub user data: %v", err))
		return
	}

	// Store session and redirect
	session, err := h.service.GitHubAuthorization(&userData)
	if err != nil {
		helpers.ErrorHandler(w, http.StatusBadRequest, fmt.Errorf("Error during GitHub authorization: %v", err))
		return
	}

	helpers.SessionCookieSet(w, session.Token, session.ExpTime)
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

// Function to parse GitHub data into a struct
func getUserData(data string) (models.GitHubLoginUserData, error) {
	userData := models.GitHubLoginUserData{}
	if err := json.Unmarshal([]byte(data), &userData); err != nil {
		return models.GitHubLoginUserData{}, fmt.Errorf("Error unmarshalling GitHub data: %v", err)
	}
	return userData, nil
}

// Get OAuth token from GitHub
func getGithubOauthToken(code string) (*models.GitHubResponseToken, error) {
	requestBodyMap := map[string]string{
		"client_id":     models.GitHubClientID,
		"client_secret": models.GitHubClientSecret,
		"code":          code,
		"redirect_uri":  models.GitHubRedirectURL, // Ensure this matches the registered URL
	}
	requestJSON, err := json.Marshal(requestBodyMap)
	if err != nil {
		return nil, fmt.Errorf("Error marshalling request body: %v", err)
	}

	req, reqerr := http.NewRequest("POST", "https://github.com/login/oauth/access_token", bytes.NewBuffer(requestJSON))
	if reqerr != nil {
		return nil, reqerr
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, resperr := http.DefaultClient.Do(req)
	if resperr != nil {
		return nil, resperr
	}
	defer resp.Body.Close()

	// Check for errors in the response
	if resp.StatusCode != http.StatusOK {
		respBody, _ := ioutil.ReadAll(resp.Body)
		return nil, fmt.Errorf("GitHub OAuth failed: %s - %s", resp.Status, string(respBody))
	}

	respbody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("Error reading response body: %v", err)
	}

	var ghresp models.GitHubResponseToken
	if err := json.Unmarshal(respbody, &ghresp); err != nil {
		return nil, fmt.Errorf("Error unmarshalling response body: %v", err)
	}

	return &ghresp, nil
}

// Get user data from GitHub using the access token
func getGithubData(accessToken string) (string, error) {
	req, reqerr := http.NewRequest("GET", "https://api.github.com/user", nil)
	if reqerr != nil {
		return "", reqerr
	}

	authorizationHeaderValue := fmt.Sprintf("token %s", accessToken)
	req.Header.Set("Authorization", authorizationHeaderValue)

	resp, resperr := http.DefaultClient.Do(req)
	if resperr != nil {
		return "", resperr
	}
	defer resp.Body.Close()

	// Check for errors in the response
	if resp.StatusCode != http.StatusOK {
		respBody, _ := ioutil.ReadAll(resp.Body)
		return "", fmt.Errorf("GitHub API request failed: %s - %s", resp.Status, string(respBody))
	}

	respbody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("Error reading response body: %v", err)
	}

	return string(respbody), nil
}
