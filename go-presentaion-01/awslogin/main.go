package main

import (
	"awslogin/client"
	"awslogin/htmlPage"
	"crypto/tls"
	"encoding/base64"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"gopkg.in/ini.v1"

	"github.com/antchfx/xmlquery"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sts"
)

const (
	idpEntryUrl = "https://pingfed.regeneron.com/idp/startSSO.ping?PartnerSpId=urn%3Aamazon%3Awebservices"
	region      = "us-east-1"
)

func check(e error) {
	if e != nil {
		panic(e)
	}
}

func myCloser(w io.Closer) {
	err := w.Close()
	if err != nil {
		log.Panicf("Couldn't close io.Closer: %v", err)
	}
}

func getCredentials(user client.User, roleToAssume *string, profileHeader string) (expiration time.Time) {
	// Programmatically get the SAML assertion
	// Opens the initial IdP url and follows all of the HTTP302 redirects, and
	// gets the resulting login page

	http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	resp, err := http.Get(idpEntryUrl)
	check(err)
	defer myCloser(resp.Body)
	cookies := resp.Cookies()

	// Parse the response and extract all the necessary values
	// in order to build a data payload of all of the form values the IdP expects
	var awsConsoleLoginForm htmlPage.Form
	awsConsoleLoginForm = htmlPage.ExtractForm(resp.Body)
	awsConsoleLoginForm.Inputs["pf.username"] = []string{user.Username}
	awsConsoleLoginForm.Inputs["pf.pass"] = []string{string(user.Password)}
	awsConsoleLoginForm.Inputs["pf.ok"] = []string{"clicked"}
	data := url.Values{}
	for key, val := range awsConsoleLoginForm.Inputs {
		data.Add(key, val[0])
	}

	// Performs a new POST submission of the IdP login form with the above post data and cookies
	httpClient := &http.Client{}
	req2, err := http.NewRequest(awsConsoleLoginForm.Method, awsConsoleLoginForm.Base+awsConsoleLoginForm.Action[1:], strings.NewReader(data.Encode()))
	check(err)
	req2.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	req2.Header.Add("Content-Length", strconv.Itoa(len(data.Encode())))
	for _, cookie := range cookies {
		req2.AddCookie(cookie)
	}
	resp2, err := httpClient.Do(req2)
	check(err)
	defer myCloser(resp2.Body)

	// Decode the response and extract the SAML assertion
	var awsConsoleRolesForm htmlPage.Form
	awsConsoleRolesForm = htmlPage.ExtractForm(resp2.Body)
	samlSAMLResponseInput := awsConsoleRolesForm.Inputs["SAMLResponse"]
	saml := ""
	if samlSAMLResponseInput != nil {
		saml = samlSAMLResponseInput[0]
	} else {
		fmt.Println("Username or Password was incorrect! Please try again:)")
		os.Exit(0)
	}
	decodedSaml, _ := base64.StdEncoding.DecodeString(saml)

	// Decoded SAML is in XML format
	// We'll use xpath to extract out the roles
	root, err := xmlquery.Parse(strings.NewReader(string(decodedSaml)))
	check(err)

	nodeRoles := xmlquery.Find(root, "//saml:Attribute[ends-with(@Name,'Attributes/Role')]/saml:AttributeValue")
	var awsroles []string

	// Note the format of the attribute value should be role_arn,principal_arn
	// but lots of blogs list it as principal_arn,role_arn so let's reverse
	// them if needed
	for _, node := range nodeRoles {
		nodeText := node.InnerText()
		chunks := strings.Split(nodeText, ",")
		if strings.Contains(chunks[0], "saml-provider") {
			chunks[0], chunks[1] = chunks[1], chunks[0]
		}
		awsroles = append(awsroles, chunks[0]+","+chunks[1])
	}

	if awsroles == nil {
		log.Fatalf("No roles returned for user %s\n", user.Username)
	}

	// If I have more than one role, ask the user which one they want,
	// otherwise just proceed
	selectedRoleIndex := 0
	if len(awsroles) > 1 {
		if *roleToAssume == "" {

			fmt.Printf("\nPlease choose the role you would like to assume:\n\n")
			for i, role := range awsroles {
				fmt.Printf("[%d] %s\n", i, strings.Split(strings.Split(role, ",")[0], "/")[1])
			}
			_, err := fmt.Scan(&selectedRoleIndex)
			if err != nil {
				log.Fatalf("Error getting scanning for role chosen: %v", err)
			}
		} else {
			for i, role := range awsroles {
				if *roleToAssume == strings.Split(strings.Split(role, ",")[0], "/")[1] {
					selectedRoleIndex = i
				}
			}

		}
	}

	// Basic sanity check of input
	fmt.Printf("Your selected role will be '%s'\n\n", strings.Split(strings.Split(awsroles[selectedRoleIndex], ",")[0], "/")[1])
	roleArn := strings.Split(awsroles[selectedRoleIndex], ",")[0]
	principalArn := strings.Split(awsroles[selectedRoleIndex], ",")[1]
	// set roleToAssume here so it survives when the session expires and the user entered the role interactively
	*roleToAssume = strings.Split(roleArn, "/")[1]

	sess := session.Must(session.NewSession(&aws.Config{
		Region: aws.String("us-east-1"),
	}))
	svc := sts.New(sess)

	samlInput := sts.AssumeRoleWithSAMLInput{
		PrincipalArn:  &principalArn,
		RoleArn:       &roleArn,
		SAMLAssertion: &saml,
	}

	// Use the assertion to get an AWS STS token using Assume Role with SAML
	samlOutPut, err := svc.AssumeRoleWithSAML(&samlInput)
	if err != nil {
		panic(err)
	}

	accessKeyID := *samlOutPut.Credentials.AccessKeyId
	secretAccessKey := *samlOutPut.Credentials.SecretAccessKey
	sessionToken := *samlOutPut.Credentials.SessionToken
	expirationTime := *samlOutPut.Credentials.Expiration

	// Write the AWS STS token into the AWS credential file
	path := user.AwsCredentialsFilePath()
	cfg, err := ini.Load(path)
	if err != nil {
		fmt.Printf("Fail to read aws credentials at %s: %v", path, err)
		os.Exit(1)
	}

	cfg.Section(profileHeader).Key("output").SetValue("json")
	cfg.Section(profileHeader).Key("region").SetValue(region)
	cfg.Section(profileHeader).Key("aws_access_key_id").SetValue(accessKeyID)
	cfg.Section(profileHeader).Key("aws_secret_access_key").SetValue(secretAccessKey)
	cfg.Section(profileHeader).Key("aws_session_token").SetValue(sessionToken)

	err = cfg.SaveTo(path)
	if err != nil {
		fmt.Printf("Failed to save aws credentials file at %s: %v", path, err)
		os.Exit(1)
	}

	fmt.Printf("The AWS profile [%s] has been updated with new credentials\n", profileHeader)
	fmt.Println("accessKeyId:      ", accessKeyID)
	fmt.Println("secretAccessKey:  ", secretAccessKey)
	fmt.Println("sessionToken:     ", sessionToken)
	fmt.Println("Expires at:       ", expirationTime)

	return expirationTime
}

func main() {

	var username string
	flag.StringVar(&username, "u", "", "Username")

	var password string
	flag.StringVar(&password, "p", "", "Password")

	var assumedAwsRole string
	flag.StringVar(&assumedAwsRole, "r", "", "AWS role to assume")

	var profileHeader string
	flag.StringVar(&profileHeader, "c", "saml", "AWS credentials profile header name")

	var daemonize bool
	flag.BoolVar(&daemonize, "d", false, "Run in loop, refreshing credentials before expiration")

	var help bool
	flag.BoolVar(&help, "h", false, "Prints usage")

	flag.Parse()

	if help {
		flag.PrintDefaults()
		os.Exit(0)
	}

	user := client.User{}
	user.PasswordReader = &client.PasswordReader{}

	user.Username = username
	if username == "" {
		fmt.Print("Enter Username: ")
		err := user.GetUserName()
		if err != nil {
			log.Fatalf("Error getting username: %v", err)
		}
	}

	user.Password = []byte(password)
	if password == "" {
		fmt.Print("Enter Password: ")
		err := user.GetPassWord()
		if err != nil {
			log.Fatalf("Error getting password: %v", err)
		}
	}
//START OMIT
	expiration := getCredentials(user, &assumedAwsRole, profileHeader)

	if daemonize {
		for {
			expDuration := expiration.Sub(time.Now())
			fmt.Println("Expires in:", expDuration)
			if expDuration < (5 * time.Minute) {
				fmt.Println("Refreshing credentials...")
				expiration = getCredentials(user, &assumedAwsRole, profileHeader)
			}
			time.Sleep(60 * time.Second)
		}
	}
//END OMIT
}
