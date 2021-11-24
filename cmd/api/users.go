package main

import (
	"errors"
	"net/http"

	"github.com/BunnyTheLifeguard/greenlight/internal/data"
	"github.com/BunnyTheLifeguard/greenlight/internal/validator"
)

func (app *application) registerUserHandler(rw http.ResponseWriter, r *http.Request) {
	// Anonymous struct to hold expected data from req body
	var input struct {
		Name     string `json:"name"`
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	// Parse req body into anonymous struct
	err := app.readJSON(rw, r, &input)
	if err != nil {
		app.badRequestResponse(rw, r, err)
		return
	}

	// Copy data from req body into new User struct
	user := &data.User{
		Name:      input.Name,
		Email:     input.Email,
		Activated: false,
	}

	// Generate and store hashed & plaintext passwords
	err = user.Password.Set(input.Password)
	if err != nil {
		app.serverErrorResponse(rw, r, err)
		return
	}

	v := validator.New()

	// Validate user struct, return error messages to client if checks fail
	if data.ValidateUser(v, user); !v.Valid() {
		app.failedValidationResponse(rw, r, v.Errors)
		return
	}

	// Insert user data into DB
	id, err := app.models.User.Insert(user)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrDuplicateEmail):
			v.AddError("email", "a user with this email address already exists")
			app.failedValidationResponse(rw, r, v.Errors)
		default:
			app.serverErrorResponse(rw, r, err)
		}
		return
	}

	user.ID = id

	// Write & send 201 JSON response with user data
	err = app.writeJSON(rw, http.StatusCreated, envelope{"user": user}, nil)
	if err != nil {
		app.serverErrorResponse(rw, r, err)
	}
}