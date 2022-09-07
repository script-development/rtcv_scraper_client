package main

import "errors"

func checkIfCVHasReferenceNr(cv map[string]interface{}) (referenceNr string, err error) {
	referenceNrInterf, ok := cv["referenceNumber"]
	if !ok {
		return "", errors.New("referenceNumber field does not exists")
	}

	referenceNr, ok = referenceNrInterf.(string)
	if !ok {
		return "", errors.New("referenceNumber must be a string")
	}

	return referenceNr, nil
}
