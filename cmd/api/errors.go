package main

import (
	"fmt"
	"net/http"
)

// Generic helper method for logging error messages
func (app *application) logError(r *http.Request, err error) {
	app.logger.PrintError(err, map[string]string{
		"request_method": r.Method,
		"request_url":    r.URL.String(),
	})
}

// Generic helper method for sending JSON-formatted error messages to client
func (app *application) errorResponse(rw http.ResponseWriter, r *http.Request, status int, message interface{}) {
	env := envelope{"error": message}

	err := app.writeJSON(rw, status, env, nil)
	if err != nil {
		app.logError(r, err)
		rw.WriteHeader(500)
	}
}

// Method for unexpected problems at runtime
func (app *application) serverErrorResponse(rw http.ResponseWriter, r *http.Request, err error) {
	app.logError(r, err)

	message := "The server encountered a problem and could not process your request"
	app.errorResponse(rw, r, http.StatusInternalServerError, message)
}

// 404 Not Found
func (app *application) notFoundResponse(rw http.ResponseWriter, r *http.Request) {
	message := "the requested resource could not be found"
	app.errorResponse(rw, r, http.StatusNotFound, message)
}

// 405 Method Not Allowed
func (app *application) methodNotAllowedResponse(w http.ResponseWriter, r *http.Request) {
	message := fmt.Sprintf("the %s method is not supported for this resource", r.Method)
	app.errorResponse(w, r, http.StatusMethodNotAllowed, message)
}

// 400 Bad Request
func (app *application) badRequestResponse(rw http.ResponseWriter, r *http.Request, err error) {
	app.errorResponse(rw, r, http.StatusBadRequest, err.Error())
}

// 422 Unprocessable Entity
func (app *application) failedValidationResponse(rw http.ResponseWriter, r *http.Request, errors map[string]string) {
	app.errorResponse(rw, r, http.StatusUnprocessableEntity, errors)
}

// 429 Too Many Requests
func (app *application) rateLimitExceededResponse(rw http.ResponseWriter, r *http.Request) {
	message := "rate limit exceeded"
	app.errorResponse(rw, r, http.StatusTooManyRequests, message)
}

// 409 Conflict
func (app *application) editConflictResponse(rw http.ResponseWriter, r *http.Request) {
	message := "unable to update the record due to an edit conflict, please try again"
	app.errorResponse(rw, r, http.StatusConflict, message)
}

// 401 Unauthorized
func (app *application) invalidCredentialsResponse(rw http.ResponseWriter, r *http.Request) {
	message := "invalid authentication credentials"
	app.errorResponse(rw, r, http.StatusUnauthorized, message)
}

func (app *application) invalidAuthenticationTokenResponse(rw http.ResponseWriter, r *http.Request) {
	rw.Header().Set("WWW-Authenticate", "Bearer")

	message := "invalid or missing authentication token"
	app.errorResponse(rw, r, http.StatusUnauthorized, message)
}

func (app *application) authenticationRequiredResponse(rw http.ResponseWriter, r *http.Request) {
	message := "you must be authenticated to access this resource"
	app.errorResponse(rw, r, http.StatusUnauthorized, message)
}

// 403 Forbidden
func (app *application) inactiveAccountResponse(rw http.ResponseWriter, r *http.Request) {
	message := "your user account must be activated to access this resource"
	app.errorResponse(rw, r, http.StatusForbidden, message)
}

func (app *application) notPermittedResponse(rw http.ResponseWriter, r *http.Request) {
	message := "your user account does not have the necessary permissions to access this resource"
	app.errorResponse(rw, r, http.StatusForbidden, message)
}
