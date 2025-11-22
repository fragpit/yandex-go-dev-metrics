package router

import (
	"bytes"
	"compress/gzip"
	"crypto/hmac"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"io"
	"log/slog"
	"net/http"
	"time"
)

func (rt *Router) slogMiddleware(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		if rt.logger.Enabled(r.Context(), slog.LevelDebug) {
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
						http.StatusText(http.StatusInternalServerError),
						http.StatusInternalServerError,
					)
					return
				}
				r.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

				if r.Header.Get("Content-Encoding") == "gzip" &&
					len(bodyBytes) > 0 {
					gz, err := gzip.NewReader(bytes.NewReader(bodyBytes))
					if err != nil {
						rt.logger.Error(
							"failed to create gzip reader",
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

					decompressedBody, err = io.ReadAll(gz)
					if err != nil {
						rt.logger.Error(
							"error reading decompressed request body",
							slog.Any("error", err),
						)
						http.Error(
							w,
							http.StatusText(http.StatusInternalServerError),
							http.StatusInternalServerError,
						)
						return
					}
				}
			}

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
				slog.Int("content_length", int(r.ContentLength)),
				slog.String("host", r.Host),
				slog.String("protocol", r.Proto),
				slog.Any("headers", r.Header),
				slog.String("body", logBody),
			)
		}

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

func (rt *Router) decryptMiddleware(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Encrypted") != "" {
			if r.ContentLength == 0 {
				rt.logger.Warn("empty body received")
				r.Header.Del("X-Encrypted")
				h.ServeHTTP(w, r)
				return
			}

			data, err := io.ReadAll(r.Body)
			if err != nil {
				rt.logger.Error("failed to read body", slog.Any("error", err))
				http.Error(
					w,
					http.StatusText(http.StatusBadRequest),
					http.StatusBadRequest,
				)
				return
			}

			decrypted, err := rsa.DecryptPKCS1v15(rand.Reader, rt.privateKey, data)
			if err != nil {
				rt.logger.Error("failed to decrypt body", slog.Any("error", err))
				http.Error(
					w,
					http.StatusText(http.StatusBadRequest),
					http.StatusBadRequest,
				)
				return
			}

			r.Body = io.NopCloser(bytes.NewReader(decrypted))
			r.Header.Del("X-Encrypted")
			r.ContentLength = int64(len(decrypted))
		}

		h.ServeHTTP(w, r)
	})
}

func (rt *Router) checksumMiddleware(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("HashSHA256") == "" {
			rt.logger.Error("checksum header is nil or unset")
			http.Error(
				w,
				"checksum header is nil or unset",
				http.StatusBadRequest,
			)
			return
		}

		mac := hmac.New(sha256.New, rt.secretKey)
		body, err := io.ReadAll(r.Body)
		if err != nil {
			rt.logger.Error(
				"failed to read request body",
				slog.Any("error", err),
			)
			http.Error(
				w,
				http.StatusText(http.StatusInternalServerError),
				http.StatusInternalServerError,
			)
			return
		}

		mac.Write(body)
		sum := mac.Sum(nil)
		sumEncoded := base64.RawStdEncoding.EncodeToString(sum)
		sumFromHeader := r.Header.Get("HashSHA256")

		if sumFromHeader != sumEncoded {
			rt.logger.Error("invalid request checksum")
			http.Error(w, "invalid request checksum", http.StatusBadRequest)
			return
		}

		r.Body = io.NopCloser(bytes.NewReader(body))

		h.ServeHTTP(w, r)
	})
}
