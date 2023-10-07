package main

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/julienschmidt/httprouter"
	"snippetbox.jll32.me/internal/models"
	"snippetbox.jll32.me/internal/validator"
	"snippetbox.jll32.me/ui/html/forms"
	"snippetbox.jll32.me/ui/html/pages"
)

func (app *application) home(w http.ResponseWriter, r *http.Request) {
	snippets, err := app.snippets.Latest()
	if err != nil {
		app.serverError(w, err)
		return
	}

	w.WriteHeader(http.StatusAccepted)
	contextData := app.newContextData(r)
	pages.Home(snippets, contextData.Flash, contextData.IsAuthenticated, contextData.CSRFToken).Render(r.Context(), w)
}

func (app *application) snippetView(w http.ResponseWriter, r *http.Request) {
	params := httprouter.ParamsFromContext(r.Context())

	id, err := strconv.Atoi(params.ByName("id"))
	if err != nil || id < 1 {
		app.notFound(w)
		return
	}

	snippet, err := app.snippets.Get(id)
	if err != nil {
		if errors.Is(err, models.ErrNoRecord) {
			app.notFound(w)
		} else {
			app.serverError(w, err)
		}
		return
	}

	w.WriteHeader(http.StatusAccepted)
	contextData := app.newContextData(r)
	pages.View(snippet, contextData.Flash, contextData.IsAuthenticated, contextData.CSRFToken).Render(r.Context(), w)
}

func (app *application) snippetCreateForm(w http.ResponseWriter, r *http.Request) {
	form := forms.SnippetCreateForm{
		Expires: 365,
	}

	w.WriteHeader(http.StatusAccepted)
	contextData := app.newContextData(r)
	pages.Create(form, contextData.Flash, contextData.IsAuthenticated, contextData.CSRFToken).Render(r.Context(), w)
}

func (app *application) snippetCreatePost(w http.ResponseWriter, r *http.Request) {
	var form forms.SnippetCreateForm

	err := app.decodePostFrom(r, &form)
	if err != nil {
		app.clientError(w, http.StatusBadRequest)
		return
	}

	form.CheckField(validator.NotBlank(form.Title), "title", "this field cannot be blank")
	form.CheckField(validator.MaxChars(form.Title, 100), "title", "this field cannot be more than 100 characters long")
	form.CheckField(validator.NotBlank(form.Content), "content", "this field cannot be blank")
	form.CheckField(validator.PermittedValue(form.Expires, 1, 7, 365), "expires", "this field must equal 1, 7, 365")

	if !form.Valid() {
		w.WriteHeader(http.StatusUnprocessableEntity)
		contextData := app.newContextData(r)
		pages.Create(form, contextData.Flash, contextData.IsAuthenticated, contextData.CSRFToken).Render(r.Context(), w)
		return
	}

	id, err := app.snippets.Insert(form.Title, form.Content, form.Expires)
	if err != nil {
		app.serverError(w, err)
		return
	}

	app.sessionManager.Put(r.Context(), "flash", "Snippet successfully created!")

	http.Redirect(w, r, fmt.Sprintf("/snippet/view/%d", id), http.StatusSeeOther)
}

func (app *application) userSignupForm(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusAccepted)
	contextData := app.newContextData(r)
	pages.Signup(forms.UserSignupForm{}, contextData.Flash, contextData.IsAuthenticated, contextData.CSRFToken).Render(r.Context(), w)
}

func (app *application) userSignupPost(w http.ResponseWriter, r *http.Request) {
	var form forms.UserSignupForm

	err := app.decodePostFrom(r, &form)
	if err != nil {
		app.clientError(w, http.StatusBadRequest)
		return
	}

	form.CheckField(validator.NotBlank(form.Name), "name", "this field cannot be blank")
	form.CheckField(validator.NotBlank(form.Email), "email", "this field cannot be blank")
	form.CheckField(validator.Matches(form.Email, validator.EmailRx), "email", "this field must have valid email address")
	form.CheckField(validator.NotBlank(form.Password), "password", "this field cannot be blank")
	form.CheckField(validator.MinChars(form.Password, 8), "password", "this field must be at least 8 characters long")

	if !form.Valid() {
		w.WriteHeader(http.StatusUnprocessableEntity)
		contextData := app.newContextData(r)
		pages.Signup(form, contextData.Flash, contextData.IsAuthenticated, contextData.CSRFToken).Render(r.Context(), w)
		return
	}

	err = app.users.Insert(form.Name, form.Email, form.Password)
	if err != nil {
		if errors.Is(err, models.ErrDuplicateEmail) {
			form.AddFieldError("email", "Email address is already in use")

			w.WriteHeader(http.StatusUnprocessableEntity)
			contextData := app.newContextData(r)
			pages.Signup(form, contextData.Flash, contextData.IsAuthenticated, contextData.CSRFToken).Render(r.Context(), w)
		} else {
			app.serverError(w, err)
		}

		return
	}

	app.sessionManager.Put(r.Context(), "flash", "Your signup was successful. Please log in.")

	http.Redirect(w, r, "/user/login", http.StatusSeeOther)
}

func (app *application) userLoginForm(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusAccepted)
	contextData := app.newContextData(r)
	pages.Login(forms.UserLoginForm{}, contextData.Flash, contextData.IsAuthenticated, contextData.CSRFToken).Render(r.Context(), w)
}

func (app *application) userLoginPost(w http.ResponseWriter, r *http.Request) {
	var form forms.UserLoginForm

	err := app.decodePostFrom(r, &form)
	if err != nil {
		app.clientError(w, http.StatusBadRequest)
		return
	}

	form.CheckField(validator.NotBlank(form.Email), "email", "This field cannot be blank")
	form.CheckField(validator.Matches(form.Email, validator.EmailRx), "email", "This field must be a valid email address")
	form.CheckField(validator.NotBlank(form.Password), "password", "This field cannot be blank")

	if !form.Valid() {
		w.WriteHeader(http.StatusUnprocessableEntity)
		contextData := app.newContextData(r)
		pages.Login(form, contextData.Flash, contextData.IsAuthenticated, contextData.CSRFToken).Render(r.Context(), w)
		return
	}

	id, err := app.users.Authenticate(form.Email, form.Password)
	if err != nil {
		if errors.Is(err, models.ErrInvalidCredentials) {
			form.AddNonFieldError("Email or password is incorrect")

			w.WriteHeader(http.StatusUnprocessableEntity)
			contextData := app.newContextData(r)
			pages.Login(form, contextData.Flash, contextData.IsAuthenticated, contextData.CSRFToken).Render(r.Context(), w)
		} else {
			app.serverError(w, err)
		}
		return
	}

	err = app.sessionManager.RenewToken(r.Context())
	if err != nil {
		app.serverError(w, err)
		return
	}

	app.sessionManager.Put(r.Context(), "authenticatedUserID", id)

	http.Redirect(w, r, "/snippet/create", http.StatusSeeOther)
}

func (app *application) userLogoutPost(w http.ResponseWriter, r *http.Request) {
	err := app.sessionManager.RenewToken(r.Context())
	if err != nil {
		app.serverError(w, err)
		return
	}

	app.sessionManager.Remove(r.Context(), "authenticatedUserID")

	app.sessionManager.Put(r.Context(), "flash", "You've been logged out successfully!")

	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func ping(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("OK"))
}
