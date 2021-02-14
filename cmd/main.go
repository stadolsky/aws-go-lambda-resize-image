package main

import (
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"

	"gopkg.in/gographics/imagick.v2/imagick"
)

type Event struct {
	Region      string      `json:"region"`
	InBucket    string      `json:"in_bucket"`
	InImageKey  string      `json:"in_image_key"`
	OutBucket   string      `json:"out_bucket"`
	OutImageKey string      `json:"out_image_key"`
	Resolution  uint        `json:"resolution"` // px
	OutFormat   imageFormat `json:"out_format"` // jpg, png
}

type imageFormat string

const (
	imageFormatJPG imageFormat = "jpg"
	imageFormatPNG imageFormat = "png"
)

func main() {
	lambda.Start(HandleLambdaEvent)
}

func HandleLambdaEvent(event Event) error {
	sess, err := session.NewSession(&aws.Config{Region: aws.String(event.Region)})
	if err != nil {
		return fmt.Errorf("failed to create NewSession for SDK: %w", err)
	}

	svc := s3.New(sess)

	input := s3.GetObjectInput{
		Bucket: aws.String(event.InBucket),
		Key:    aws.String(event.InImageKey),
	}

	objectOutput, err := svc.GetObject(&input)
	if err != nil {
		return fmt.Errorf("failed to get object `%s` in bucket `%s`: %w", event.InImageKey, event.InBucket, err)
	}

	defer objectOutput.Body.Close()

	// Read the chunk
	originalImageData, err := ioutil.ReadAll(objectOutput.Body)
	newImageData, err := resize(originalImageData, event.Resolution, event.OutFormat)
	if err != nil {
		return fmt.Errorf("fialde to create new resided image: %w", err)
	}

	_, err = svc.PutObject(&s3.PutObjectInput{
		Body:   bytes.NewReader(newImageData),
		Bucket: aws.String(event.OutBucket),
		Key:    aws.String(event.OutImageKey),
	})

	if err != nil {
		return fmt.Errorf("failed to save image with keyobject `%s` in bucket `%s`: %w", event.OutImageKey, event.OutBucket, err)
	}

	return nil
}

func (f imageFormat) isValid() bool {
	switch f {
	case imageFormatJPG:
		return true
	case imageFormatPNG:
		return true
	}

	return false
}

func validateEvent(event Event) error {
	if event.Region == "" {
		return errors.New("event param `Region` is required")
	}

	if event.InBucket == "" {
		return errors.New("event param `InBucket` is required")
	}

	if event.InImageKey == "" {
		return errors.New("event param `InImageKey` is required")
	}

	if event.OutBucket == "" {
		return errors.New("event param `OutBucket` is required")
	}

	if event.OutImageKey == "" {
		return errors.New("event param `OutImageKey` is required")
	}

	if event.OutFormat == "" {
		return errors.New("event param `OutFormat` is required")
	}

	if !event.OutFormat.isValid() {
		return errors.New(fmt.Sprintf("event param `OutFormat` value `%s` is invalid. allowed values are `%s`, `%s`",
			event.OutFormat, imageFormatPNG, imageFormatJPG,
		))
	}

	if event.Resolution == 0 {
		return errors.New("event param `Resolution` must not be zero")
	}

	return nil
}

func resize(data []byte, resolution uint, format imageFormat) ([]byte, error) {
	imagick.Initialize()
	// Schedule cleanup
	defer imagick.Terminate()

	var err error

	mw := imagick.NewMagickWand()

	err = mw.ReadImageBlob(data)
	if err != nil {
		return nil, err
	}

	// Get original image size
	width := mw.GetImageWidth()
	height := mw.GetImageHeight()

	// Calculate New Image Size
	var newWidth, newHeight uint
	if width > height {
		newHeight = resolution
		newWidth = newHeight * width / height
	} else {
		newWidth = resolution
		newHeight = newWidth * height / width
	}

	// Resize the image using the Lanczos filter
	// The blur factor is a float, where > 1 is blurry, < 1 is sharp
	err = mw.ResizeImage(newWidth, newHeight, imagick.FILTER_LANCZOS, 1)
	if err != nil {
		return nil, err
	}

	// Set the compression quality to 95 (high quality = low compression)
	err = mw.SetImageCompressionQuality(95)
	if err != nil {
		return nil, err
	}

	// Convert into pointed format
	if err := mw.SetFormat(string(format)); err != nil {
		return nil, err
	}

	return mw.GetImageBlob(), nil
}
