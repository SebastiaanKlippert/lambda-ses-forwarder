package main

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"net/mail"
	"os"
	"path"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/ses"
)

var (
	sesClient *ses.SES
	s3Client  *s3.S3
)

func main() {

	// check if a different region has been set for S3, if not default to the Lambda region
	s3Region := os.Getenv("S3_BUCKET_REGION")
	if s3Region == "" {
		s3Region = os.Getenv("AWS_REGION")
	}

	sess, err := session.NewSession()
	if err != nil {
		log.Fatalf("AWS NewSession failed: %s", err)
	}
	sesClient = ses.New(sess)
	s3Client = s3.New(sess, aws.NewConfig().WithRegion(s3Region))

	lambda.Start(forward)
}

func forward(event events.SimpleEmailEvent) error {

	for _, sesMail := range event.Records {

		if err := forwardMail(&sesMail); err != nil {
			return err
		}
	}
	return nil
}

func forwardMail(original *events.SimpleEmailRecord) error {

	// get original message from S3
	s3Mail, err := getFromS3(original)
	if err != nil {
		return err
	}
	defer s3Mail.Close()

	// parse the original message
	parsedMail, err := mail.ReadMessage(s3Mail)
	if err != nil {
		return fmt.Errorf("ReadMessage failed: %s", err)
	}

	// parse forwarder FROM and TO addresses
	addrTo, err := mail.ParseAddress(os.Getenv("FORWARD_TO"))
	if err != nil {
		return fmt.Errorf("ParseAddress failed for FORWARD_TO: %s", err)
	}

	// parse original From and add it to the FORWARD_FROM address
	orgFrom, _ := mail.ParseAddress(parsedMail.Header.Get("From"))
	if orgFrom == nil {
		orgFrom = &mail.Address{}
	}
	// FORWARD_FROM may contain %s to include the original sender name
	from := fmt.Sprintf(os.Getenv("FORWARD_FROM"), orgFrom.Name)
	addrFrom, err := mail.ParseAddress(from)
	if err != nil {
		return fmt.Errorf("ParseAddress failed for FORWARD_FROM: %s", err)
	}

	// compose new message in buffer
	newMail := new(bytes.Buffer)

	// add all original headers except address headers
	for h := range parsedMail.Header {
		if h == "To" || h == "From" || h == "Bcc" || h == "Reply-To" {
			continue
		}
		fmt.Fprintf(newMail, "%s: %s\r\n", h, parsedMail.Header.Get(h))
	}

	// set from and to
	fmt.Fprintf(newMail, "From: %s\r\n", addrFrom.String())
	fmt.Fprintf(newMail, "To: %s\r\n", addrTo.String())

	// reply-to is the original sender or original reply-to
	rt := parsedMail.Header.Get("Reply-To")
	if rt == "" {
		rt = parsedMail.Header.Get("From")
	}
	fmt.Fprintf(newMail, "Reply-To: %s\r\n", rt)

	// set body
	newMail.WriteString("\r\n")
	_, err = newMail.ReadFrom(parsedMail.Body)
	if err != nil {
		return fmt.Errorf("reading mail body failed: %s", err)
	}

	// send mail
	rawMail := &ses.SendRawEmailInput{
		RawMessage: &ses.RawMessage{
			Data: newMail.Bytes(),
		},
	}

	_, err = sesClient.SendRawEmail(rawMail)
	if err != nil {
		return fmt.Errorf("SES SendRawEmail failed: %s", err)
	}

	return nil
}

func getFromS3(original *events.SimpleEmailRecord) (io.ReadCloser, error) {

	key := path.Join(os.Getenv("S3_PREFIX"), original.SES.Mail.MessageID)

	obj, err := s3Client.GetObject(&s3.GetObjectInput{
		Bucket: aws.String(os.Getenv("S3_BUCKET")),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, fmt.Errorf("S3 GetObject failed: %s", err)
	}
	return obj.Body, nil
}
