package main

import (
	"bytes"
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
	Resolution  string      `json:"resolution"`
	OutFormat   imageFormat `json:"out_format"`
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
	newImageData, err := resize(originalImageData, event.OutFormat)
	if err != nil {
		return fmt.Errorf("fialde to create new resided image: %w", err)
	}

	_, err = svc.PutObject(&s3.PutObjectInput{
		Body:   bytes.NewReader(newImageData),
		Bucket: aws.String(event.OutBucket),
		Key:    aws.String(event.OutImageKey),
	})

	fmt.Println(fmt.Sprint("New file size: %d", len(newImageData)))

	if err != nil {
		return fmt.Errorf("failed to save image with keyobject `%s` in bucket `%s`: %w", event.OutImageKey, event.OutBucket, err)
	}

	return nil
}

func resize(data []byte, format imageFormat) ([]byte, error) {
	imagick.Initialize()
	// Schedule cleanup
	defer imagick.Terminate()
	var err error

	mw := imagick.NewMagickWand()

	err = mw.ReadImageBlob(data)
	if err != nil {
		return nil, err
	}

	// Get original logo size
	width := mw.GetImageWidth()
	height := mw.GetImageHeight()

	// Calculate half the size
	hWidth := uint(width / 2)
	hHeight := uint(height / 2)

	// Resize the image using the Lanczos filter
	// The blur factor is a float, where > 1 is blurry, < 1 is sharp
	err = mw.ResizeImage(hWidth, hHeight, imagick.FILTER_LANCZOS, 1)
	if err != nil {
		return nil, err
	}

	// Set the compression quality to 95 (high quality = low compression)
	err = mw.SetImageCompressionQuality(95)
	if err != nil {
		return nil, err
	}

	// Convert into JPG
	if err := mw.SetFormat(string(format)); err != nil {
		return nil, err
	}

	return mw.GetImageBlob(), nil
}
