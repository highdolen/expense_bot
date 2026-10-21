package middleware

import (
	"log"
	"net/http"
)

func Recovery(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if recovered := recover(); recovered != nil {
				log.Printf(
					"panic recovered: method=%s path=%s error=%v",
					r.Method,
					r.URL.Path,
					recovered,
				)

				http.Error(
					w,
					`{"error":{"code":"internal_error","message":"Внутренняя ошибка сервера"}}`,
					http.StatusInternalServerError,
				)
			}
		}()

		next.ServeHTTP(w, r)
	})
}
