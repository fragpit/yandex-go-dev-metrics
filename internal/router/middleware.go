package router

import (
	"bytes"
	"compress/gzip"
	"io"
	"log/slog"
	"net/http"
	"time"
)

func (rt *Router) slogMiddleware(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		var err error
		var bodyBytes []byte
		var decompressedBody []byte
		if r.Body != nil {
			bodyBytes, err = io.ReadAll(r.Body)
			if err != nil {
				rt.logger.Error(
					"error reading request body",
					slog.Any("error", err),
				)
				http.Error(
					w,
					"error reading request body",
					http.StatusInternalServerError,
				)
				return
			}
			r.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

			if r.Header.Get("Content-Encoding") == "gzip" &&
				len(bodyBytes) > 0 {
				gz, err := gzip.NewReader(
					bytes.NewReader(bodyBytes))
				if err == nil {
					decompressedBody, _ = io.ReadAll(gz)
					gz.Close()
				}
			}
		}

		// Choose which body to log
		logBody := string(bodyBytes)
		if len(decompressedBody) > 0 {
			logBody = string(decompressedBody)
		}

		rt.logger.Debug("request details",
			slog.String("method", r.Method),
			slog.String("path", r.URL.Path),
			slog.String("query", r.URL.RawQuery),
			slog.String("user_agent", r.UserAgent()),
			slog.String("referer", r.Referer()),
			slog.String("content_type", r.Header.Get("Content-Type")),
			slog.String("content_encoding", r.Header.Get("Content-Encoding")),
			slog.Int("content_length", int(r.ContentLength)),
			slog.String("host", r.Host),
			slog.String("protocol", r.Proto),
			slog.String("request_id", r.Header.Get("X-Request-ID")),
			slog.String("accept", r.Header.Get("Accept")),
			slog.String("accept_encoding", r.Header.Get("Accept-Encoding")),
			slog.String("body", logBody),
		)

		ww := &responseWriter{ResponseWriter: w, statusCode: 200}

		h.ServeHTTP(ww, r)

		rt.logger.Info("request completed",
			slog.String("method", r.Method),
			slog.String("path", r.URL.Path),
			slog.Int("status", ww.statusCode),
			slog.Int("resp_size", ww.size),
			slog.Duration("duration", time.Since(start)),
			slog.String("remote_addr", r.RemoteAddr),
		)
	})
}

func (rt *Router) decompressMiddleware(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Content-Encoding") == "gzip" {
			if r.ContentLength == 0 {
				rt.logger.Warn("empty body received")
				r.Header.Del("Content-Encoding")
				h.ServeHTTP(w, r)
				return
			}

			gz, err := gzip.NewReader(r.Body)
			if err != nil {
				rt.logger.Error(
					"failed to create reader",
					slog.Any("error", err),
				)
				http.Error(
					w,
					"failed to create reader",
					http.StatusBadRequest,
				)
				return
			}
			defer gz.Close()

			decompressed, err := io.ReadAll(gz)
			if err != nil {
				rt.logger.Error(
					"failed to decompress body",
					slog.Any("error", err),
				)
				http.Error(
					w,
					"failed to decompress body",
					http.StatusBadRequest,
				)
				return
			}

			r.Body = io.NopCloser(bytes.NewReader(decompressed))
			r.Header.Del("Content-Encoding")
			r.ContentLength = int64(len(decompressed))
		}

		h.ServeHTTP(w, r)
	})
}
