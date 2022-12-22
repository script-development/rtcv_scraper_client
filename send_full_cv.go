package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/textproto"

	"github.com/valyala/fasthttp"
)

func parseSendFullCvRequest(req *fasthttp.Request) (func(conn serverConn, resp any) error, *CVMetadata, error) {
	// Parse the multipart form data
	form, err := req.MultipartForm()
	if err != nil {
		return nil, nil, err
	}

	cvFiles, ok := form.File["cv"]
	if !ok {
		return nil, nil, errors.New(`no "cv" form file provided`)
	}

	// Get the cv file
	switch len(cvFiles) {
	case 0:
		return nil, nil, errors.New("no cv provided")
	case 1:
		// Good
	default:
		return nil, nil, errors.New("you can only provide one cv")
	}
	file := cvFiles[0]

	metadataValues, ok := form.Value["metadata"]
	if !ok {
		return nil, nil, errors.New(`no "metadata" form value provided`)
	}

	switch len(metadataValues) {
	case 0:
		return nil, nil, errors.New("no metadata provided")
	case 1:
		// Good
	default:
		return nil, nil, errors.New("you can only provide one metadata value")
	}

	metadata := CVMetadata{}

	err = json.Unmarshal([]byte(metadataValues[0]), &metadata)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid metadata, err: %s", err)
	}

	buff := bytes.NewBuffer(nil)
	creationForm := multipart.NewWriter(buff)

	boundry := creationForm.Boundary()
	defer creationForm.Close()

	err = creationForm.WriteField("metadata", metadataValues[0])
	if err != nil {
		return nil, nil, err
	}
	err = passtroughCVFile(file, creationForm)
	if err != nil {
		return nil, nil, err
	}

	return func(conn serverConn, resp any) error {
		return conn.PostFormData("/api/v1/scraper/scanCVDocument", bytes.NewBuffer(buff.Bytes()), boundry, resp)
	}, &metadata, nil
}

func passtroughCVFile(uploadedFile *multipart.FileHeader, creationForm *multipart.Writer) error {
	h := make(textproto.MIMEHeader)
	h.Set("Content-Disposition", `form-data; name="cv"; filename="cv.pdf"`)
	uploadedCVFileHeader := uploadedFile.Header.Get("Content-Type")
	if uploadedCVFileHeader == "" {
		uploadedCVFileHeader = "application/octet-stream"
	}
	h.Set("Content-Type", uploadedCVFileHeader)
	cvFileWriter, err := creationForm.CreatePart(h)

	if err != nil {
		return err
	}

	f, err := uploadedFile.Open()
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = io.Copy(cvFileWriter, f)
	return err
}

// CVMetadata is the metadata of a CV
type CVMetadata struct {
	ReferenceNumber string          `json:"referenceNumber,omitempty"`
	Link            *string         `json:"link,omitempty"`
	CreatedAt       *string         `json:"createdAt,omitempty"`
	LastChanged     *string         `json:"lastChanged,omitempty"`
	PersonalDetails PersonalDetails `json:"personalDetails"`
}

// PersonalDetails contains the personal details of a CV
type PersonalDetails struct {
	Initials          string `json:"initials,omitempty"`
	FirstName         string `json:"firstName,omitempty"`
	SurNamePrefix     string `json:"surNamePrefix,omitempty"`
	SurName           string `json:"surName,omitempty"`
	DateOfBirth       string `json:"dob,omitempty"`
	Gender            string `json:"gender,omitempty"`
	StreetName        string `json:"streetName,omitempty"`
	HouseNumber       string `json:"houseNumber,omitempty"`
	HouseNumberSuffix string `json:"houseNumberSuffix,omitempty"`
	Zip               string `json:"zip,omitempty"`
	City              string `json:"city,omitempty"`
	Country           string `json:"country,omitempty"`
	PhoneNumber       string `json:"phoneNumber,omitempty"`
	Email             string `json:"email,omitempty"`
}
