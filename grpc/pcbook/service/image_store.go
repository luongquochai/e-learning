package service

import "bytes"

// This role to save uploaded image file somewhere on the server or on the cloud
// To make it generic and easy to change to different types of storage.
type ImageStore interface {
	Save(laptopID string, imageType string, imageData bytes.Buffer) (string, error)
}
