package utils

import (
	"fmt"
	"strconv"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ses"
	"github.com/matcornic/hermes/v2"
)

const (
	// Replace sender@example.com with your "From" address.
	// This address must be verified with Amazon SES.
	Sender = "noreply@hackmit.org"

	// The subject line for the email.
	Subject = "Amazon SES Test (AWS SDK for Go)"

	// The character encoding for the email.
	CharSet = "UTF-8"
)

func SendConfirmationEmail(recipient string, code int) {
	h := hermes.Hermes{
		// Optional Theme
		// Theme: new(Default)
		Product: hermes.Product{
			// Appears in header & footer of e-mails
			Name: "Hermes",
			Link: "https://example-hermes.com/",
			// Optional product logo
			Logo: "http://www.duchess-france.org/wp-content/uploads/2016/01/gopher.png",
		},
	}

	email := hermes.Email{
		Body: hermes.Body{
			Intros: []string{
				"Welcome to the HackMIT playground! We're very excited to have you this weekend.",
			},
			Actions: []hermes.Action{
				{
					Instructions: "Please copy your invite code:",
					// Make sure this is 6 digits (zero-pad on left)
					InviteCode: strconv.Itoa(code),
				},
			},
			Outros: []string{
				"Need help, or have questions? Just reply to this email, we'd love to help.",
			},
		},
	}

	html, _ := h.GenerateHTML(email)
	plainText, _ := h.GeneratePlainText(email)

	// 2. send email to person
	sess, _ := session.NewSession(&aws.Config{
		Region: aws.String("us-east-1")},
	)

	// Create an SES session.
	svc := ses.New(sess)

	// Assemble the email.
	input := &ses.SendEmailInput{
		Destination: &ses.Destination{
			CcAddresses: []*string{},
			ToAddresses: []*string{
				aws.String(recipient),
			},
		},
		Message: &ses.Message{
			Body: &ses.Body{
				Html: &ses.Content{
					Charset: aws.String(CharSet),
					Data:    aws.String(html),
				},
				Text: &ses.Content{
					Charset: aws.String(CharSet),
					Data:    aws.String(plainText),
				},
			},
			Subject: &ses.Content{
				Charset: aws.String(CharSet),
				Data:    aws.String(Subject),
			},
		},
		Source: aws.String(Sender),
	}

	// Attempt to send the email.
	_, err := svc.SendEmail(input)

	if err != nil {
		fmt.Println(err)
	}
}
