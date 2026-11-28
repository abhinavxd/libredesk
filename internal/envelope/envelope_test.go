package envelope

import (
	"testing"

	"github.com/valyala/fasthttp"
)

func TestNewErrorHTTPStatusCodes(t *testing.T) {
	tests := []struct {
		errorType    string
		wantCode     int
		wantErrType  string
	}{
		{GeneralError, fasthttp.StatusInternalServerError, GeneralError},
		{PermissionError, fasthttp.StatusForbidden, PermissionError},
		{InputError, fasthttp.StatusBadRequest, InputError},
		{DataError, fasthttp.StatusUnprocessableEntity, DataError},
		{NetworkError, fasthttp.StatusGatewayTimeout, NetworkError},
		{NotFoundError, fasthttp.StatusNotFound, NotFoundError},
		{ConflictError, fasthttp.StatusConflict, ConflictError},
		{UnauthorizedError, fasthttp.StatusUnauthorized, UnauthorizedError},
		{RateLimitError, fasthttp.StatusTooManyRequests, RateLimitError},
	}

	for _, tc := range tests {
		t.Run(tc.errorType, func(t *testing.T) {
			err := NewError(tc.errorType, "test message", nil)
			e, ok := err.(Error)
			if !ok {
				t.Fatalf("NewError did not return an Error value")
			}
			if e.Code != tc.wantCode {
				t.Errorf("Code = %d, want %d", e.Code, tc.wantCode)
			}
			if e.ErrorType != tc.wantErrType {
				t.Errorf("ErrorType = %q, want %q", e.ErrorType, tc.wantErrType)
			}
		})
	}
}

func TestNewErrorUnknownTypeFallsBackToGeneral(t *testing.T) {
	err := NewError("SomeUnknownError", "oops", nil)
	e, ok := err.(Error)
	if !ok {
		t.Fatalf("NewError did not return an Error value")
	}
	if e.Code != fasthttp.StatusInternalServerError {
		t.Errorf("Code = %d, want %d", e.Code, fasthttp.StatusInternalServerError)
	}
	if e.ErrorType != GeneralError {
		t.Errorf("ErrorType = %q, want %q", e.ErrorType, GeneralError)
	}
}

func TestNewErrorMessageAndData(t *testing.T) {
	msg := "something went wrong"
	data := map[string]string{"field": "email"}
	err := NewError(InputError, msg, data)
	e, ok := err.(Error)
	if !ok {
		t.Fatalf("NewError did not return an Error value")
	}
	if e.Message != msg {
		t.Errorf("Message = %q, want %q", e.Message, msg)
	}
	if e.Error() != msg {
		t.Errorf("Error() = %q, want %q", e.Error(), msg)
	}
	if e.Data == nil {
		t.Error("Data should not be nil")
	}
}

func TestNewErrorWithCode(t *testing.T) {
	err := NewErrorWithCode("CustomError", 418, "I'm a teapot", nil)
	e, ok := err.(Error)
	if !ok {
		t.Fatalf("NewErrorWithCode did not return an Error value")
	}
	if e.Code != 418 {
		t.Errorf("Code = %d, want 418", e.Code)
	}
	if e.ErrorType != "CustomError" {
		t.Errorf("ErrorType = %q, want %q", e.ErrorType, "CustomError")
	}
	if e.Message != "I'm a teapot" {
		t.Errorf("Message = %q, want %q", e.Message, "I'm a teapot")
	}
}

func TestErrorImplementsErrorInterface(t *testing.T) {
	var err error = NewError(GeneralError, "test", nil)
	if err.Error() != "test" {
		t.Errorf("Error() = %q, want %q", err.Error(), "test")
	}
}
