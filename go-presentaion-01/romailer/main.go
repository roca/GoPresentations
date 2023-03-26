package main

import (
	"crypto/tls"
	"flag"
	"fmt"
	"log"
	"net/http"

	"regeneron.com/romailer/api/controllers"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/gorilla/mux"
)

var useSamlProfile bool = false

var svc *dynamodb.DynamoDB

// TableName ...
var TableName string = "email_logs"

func initSession() {
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{Transport: tr}

	awsConfig := &aws.Config{
		Region:     aws.String("us-east-1"),
		HTTPClient: client,
	}

	if useSamlProfile {
		awsConfig.Credentials = credentials.NewSharedCredentials("", "saml")
		fmt.Println("Using local SAML profile")
	}

	sess, err := session.NewSession(awsConfig)
	if err != nil {
		fmt.Printf("Could not establish session: %v\n", err)
	}

	// Create DynamoDB client
	svc = dynamodb.New(sess)
	fmt.Println(svc.ClientInfo.ServiceName)

}

// START OMIT
func main() {
	flag.BoolVar(&useSamlProfile, "saml", false, "User sample profile [true|false]")
	flag.Parse()
	initSession()
	router := mux.NewRouter()
	controller := controllers.Controller{}
	router.HandleFunc("/mail", controller.SendMail(TableName, svc)).Methods("POST")
	router.HandleFunc("/mail", controller.GetMessages(TableName, svc)).Methods("GET")
	router.HandleFunc("/mail/{id}", controller.GetMessage(TableName, svc)).Methods("GET")
	router.HandleFunc("/mail/{id}", controller.DeleteMessage(TableName, svc)).Methods("DELETE")

	sh := http.StripPrefix("/swagger-ui/", http.FileServer(http.Dir("./swagger-ui/")))
	router.PathPrefix("/swagger-ui/").Handler(sh)

	log.Fatal(http.ListenAndServe(":8000", router))
}

// END OMIT
