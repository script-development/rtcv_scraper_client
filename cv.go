package main

import (
	"encoding/json"
	"errors"
	"strconv"
	"strings"
)

// StrippedCVWithOriginal contains the stripped cv and the original bytes
// Using this we can unmarshal the cv and then marshal it again without losing any data
// Without this we will loos all info and only have the fields within the stripped cv
type StrippedCVWithOriginal struct {
	StrippedCV
	JSONBytes []byte
}

// UnmarshalJSON unmarshals the cv into StrippedCV and saves the bytes also to JSONBytes
func (a *StrippedCVWithOriginal) UnmarshalJSON(b []byte) error {
	// Copy the bytes of the cv into a.JSONBytes
	a.JSONBytes = append([]byte{}, b...)
	return json.Unmarshal(b, &a.StrippedCV)
}

// MarshalJSON returns the cv that was inputted to UnmarshalJSON
func (a StrippedCVWithOriginal) MarshalJSON() ([]byte, error) {
	return a.JSONBytes, nil
}

// StrippedCV is a stripped down version of the RT-CV CV model that only contains the fields we check in this tool
type StrippedCV struct {
	ReferenceNumber string                  `json:"referenceNumber"`
	PersonalDetails StrippedPersonalDetails `json:"personalDetails"`
}

// StrippedPersonalDetails contains a stripped version of the personal details of a RT-CV cv.
// We only have the fields from the RT-CV we use for checking if the cv is valid
type StrippedPersonalDetails struct {
	Zip string `json:"zip"`
}

func (cv *StrippedCV) checkRefNr() error {
	if cv.ReferenceNumber == "" {
		return errors.New("referenceNumber cannot be empty")
	}
	return nil
}

// ErrInvalidZip is returned when the zip code is not valid
var ErrInvalidZip = errors.New("personalDetails.zip has a invalid zip code")

func (cv *StrippedCV) checkMustHaveValidZip() error {
	zip := strings.TrimSpace(cv.PersonalDetails.Zip)
	if zip == "" {
		return errors.New("personalDetails.zip required")
	}

	zipLen := len(zip)
	if zipLen != 4 && zipLen != 6 {
		return ErrInvalidZip
	}
	_, err := strconv.Atoi(zip[:4])
	if err != nil {
		return ErrInvalidZip
	}
	return nil
}
