package main

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"github.com/aws/aws-lambda-go/lambda"
)

func main() {
	lambda.Start(HandleLambdaEvent)
}

type Event struct {
	InBucket    string `json:"in_bucket"`
	InImageKey  string `json:"in_image_key"`
	OutBucket   string `json:"out_bucket"`
	OutImageKey string `json:"out_image_key"`
	Resolution  string `json:"resolution"`
}

type Response struct {
	PDF64 string `json:"pdf"`
}

func HandleLambdaEvent(event Event) (*Response, error) {

	return nil, nil
}
