package utils

import (
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ses"
	"github.com/matcornic/hermes/v2"
)

const (
	// Sender
	Sender = "Blueprint <noreply@hackmit.org>"

	// The subject line for the email.
	Subject = "Blueprint Playground Confirmation"

	// The character encoding for the email.
	CharSet = "UTF-8"

	ReplyTo = "blueprint@hackmit.org"
)

func SendConfirmationEmail(recipient string, code int, name string) {
	paddedCode := fmt.Sprintf("%06d", code)

	h := hermes.Hermes{
		Theme: new(hermes.Default),
		Product: hermes.Product{
			Name: "Blueprint",
			Link: "https://blueprint.hackmit.org",
			Logo: "https://blueprint-playground-2021.s3.amazonaws.com/utils/logo.png",
		},
	}

	email := hermes.Email{
		Body: hermes.Body{
			Name: name,
			Intros: []string{
				"Welcome to the Blueprint playground! We're very excited to have you this weekend.",
			},
			Actions: []hermes.Action{
				{
					Instructions: "Please copy your invite code:",
					InviteCode:   paddedCode,
				},
			},
			Outros: []string{
				"Any questions? Email blueprint@hackmit.org",
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
		ReplyToAddresses: []*string{aws.String(ReplyTo)},
		Source:           aws.String(Sender),
	}

	// Attempt to send the email.
	_, err := svc.SendEmail(input)

	if err != nil {
		fmt.Println(err)
	}
}
