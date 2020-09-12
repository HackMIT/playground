package utils

import (
	"fmt"

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
	Subject = "HackMIT Playground Confirmation"

	// The character encoding for the email.
	CharSet = "UTF-8"
)

func SendConfirmationEmail(recipient string, code int, name string) {
	paddedCode := fmt.Sprintf("%06d", code)
	h := hermes.Hermes{
		// Optional Theme
		Theme: new(hermes.Default),
		Product: hermes.Product{
			// Appears in header & footer of e-mails
			Name: "HackMIT",
			Link: "https://hackmit.org",
			// Optional product logo
			Logo: "https://hackmit-playground-2020.s3.amazonaws.com/utils/logo.png?fbclid=IwAR17B0II1-CuC3Ix3ZIs2as9jf62dnKydrTMhT4oKXeAHh8CEYmvqwoxs-Q",
		},
	}

	email := hermes.Email{
		Body: hermes.Body{
			Name: name,
			Intros: []string{
				"Welcome to the HackMIT playground! We're very excited to have you this weekend.",
			},
			Actions: []hermes.Action{
				{
					Instructions: "Please copy your invite code:",
					// Make sure this is 6 digits (zero-pad on left)
					InviteCode: paddedCode,
				},
			},
			Outros: []string{
				"Any questions? Email help@hackmit.org",
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
