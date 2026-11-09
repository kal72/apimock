package handler

import (
	"apimock/internal/usecase"
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"net/url"
	"strings"
)

func parseMockRequest(w http.ResponseWriter, r *http.Request) (usecase.Request, error) {
	request := usecase.Request{Method: r.Method, Path: "/" + routeParam(r, "path"), Headers: r.Header, Query: r.URL.Query()}
	var raw []byte
	if r.Body != nil {
		var err error
		raw, err = io.ReadAll(http.MaxBytesReader(w, r.Body, 1<<20))
		if err != nil {
			return request, errors.New("request body exceeds 1 MiB or could not be read")
		}
	}
	request.RawBody = string(raw)
	request.BodyPresent = len(raw) > 0
	mediaType := ""
	parameters := map[string]string{}
	if contentType := r.Header.Get("Content-Type"); contentType != "" {
		var err error
		mediaType, parameters, err = mime.ParseMediaType(contentType)
		if err != nil {
			return request, errors.New("invalid Content-Type")
		}
	}
	switch {
	case mediaType == "application/x-www-form-urlencoded":
		request.BodyFormat = "form"
		form, err := url.ParseQuery(string(raw))
		if err != nil {
			return request, errors.New("invalid form body")
		}
		request.Form = form
	case mediaType == "multipart/form-data":
		request.BodyFormat = "form"
		request.Form = map[string][]string{}
		boundary := parameters["boundary"]
		if boundary == "" {
			return request, errors.New("multipart boundary is required")
		}
		reader := multipart.NewReader(bytes.NewReader(raw), boundary)
		for count := 0; ; count++ {
			part, err := reader.NextPart()
			if err == io.EOF {
				break
			}
			if err != nil {
				return request, errors.New("invalid multipart body")
			}
			if count >= 200 {
				part.Close()
				return request, errors.New("multipart body exceeds 200 parts")
			}
			if part.FileName() == "" && part.FormName() != "" {
				value, err := io.ReadAll(part)
				if err != nil {
					part.Close()
					return request, errors.New("invalid multipart field")
				}
				request.Form[part.FormName()] = append(request.Form[part.FormName()], string(value))
			}
			part.Close()
		}
	case mediaType == "application/json" || strings.HasSuffix(mediaType, "+json") || mediaType == "" && json.Valid(raw):
		request.BodyFormat = "json"
		if len(bytes.TrimSpace(raw)) == 0 {
			request.BodyPresent = false
			break
		}
		decoder := json.NewDecoder(bytes.NewReader(raw))
		decoder.UseNumber()
		if err := decoder.Decode(&request.JSONBody); err != nil {
			return request, errors.New("invalid JSON request body")
		}
		if err := decoder.Decode(new(any)); err != io.EOF {
			return request, errors.New("request body must contain one JSON value")
		}
		request.Body, _ = request.JSONBody.(map[string]any)
	default:
		request.BodyFormat = "text"
	}
	return request, nil
}

func playgroundRequest(body any, format, raw string, form map[string][]string) (usecase.Request, error) {
	request := usecase.Request{BodyFormat: format, RawBody: raw, Form: form, JSONBody: body, BodyPresent: body != nil || raw != "" || len(form) > 0}
	if format == "" {
		request.BodyFormat = "json"
	}
	switch request.BodyFormat {
	case "json":
		if raw != "" {
			d := json.NewDecoder(strings.NewReader(raw))
			d.UseNumber()
			if err := d.Decode(&request.JSONBody); err != nil {
				return request, errors.New("playground raw_body must contain valid JSON")
			}
			if err := d.Decode(new(any)); err != io.EOF {
				return request, errors.New("playground raw_body must contain one JSON value")
			}
		}
		request.Body, _ = request.JSONBody.(map[string]any)
	case "form":
		request.JSONBody = nil
		request.BodyPresent = len(form) > 0
	case "text":
		request.JSONBody = nil
		request.BodyPresent = raw != ""
	default:
		return request, errors.New("body_mode must be json, form, or text")
	}
	return request, nil
}
