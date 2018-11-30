package main

import (
	"log"

	"github.com/aws/aws-lambda-go/lambda"
)

func main() {

	log.Println("Hi")

	lambda.Start(forward)

}

func forward() error {
	return nil
}
