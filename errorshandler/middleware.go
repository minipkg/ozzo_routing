package errorshandler

import (
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"runtime/debug"

	routing "github.com/go-ozzo/ozzo-routing/v2"
	validation "github.com/go-ozzo/ozzo-validation/v4"

	"github.com/minipkg/go-app-common/log"
)

// Handler creates a middleware that handles panics and errors encountered during HTTP request processing.
func Handler(logger log.ILogger) routing.Handler {
	return func(c *routing.Context) (err error) {
		defer func() {
			l := logger.With(c.Request.Context())
			e := recover()
			if e == nil {
				return
			}
			var ok bool
			if err, ok = e.(error); !ok {
				err = fmt.Errorf("%v", e)
			}
			l.Errorf("recovered from panic (%v): %s", err, debug.Stack())

			if err == nil {
				return
			}
			res := buildErrorResponse(err)
			if res.StatusCode() == http.StatusInternalServerError {
				l.Errorf("encountered internal server error: %v", err)
			}
			c.Response.WriteHeader(res.StatusCode())
			if err = c.Write(res); err != nil {
				l.Errorf("failed writing error response: %v", err)
			}
			c.Abort() // skip any pending handlers since an error has occurred
			err = nil // return nil because the error is already handled
		}()
		return c.Next()
	}
}

// buildErrorResponse builds an error response from an error.
func buildErrorResponse(err error) Response {
	switch err := err.(type) {
	case Response:
		return err
	case validation.Errors:
		return InvalidInput(err)
	case routing.HTTPError:
		switch err.StatusCode() {
		case http.StatusNotFound:
			return NotFound("")
		default:
			return Response{
				Status:  err.StatusCode(),
				Message: err.Error(),
			}
		}
	}

	if errors.Is(err, sql.ErrNoRows) {
		return NotFound("")
	}
	return InternalServerError("")
}
