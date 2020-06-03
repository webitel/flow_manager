package dialogflow

import (
	"bufio"
	"context"
	"fmt"
	"google.golang.org/api/option"
	"io"
	"log"
	"os"
	"strings"

	dialogflow "cloud.google.com/go/dialogflow/apiv2"
	dialogflowpb "google.golang.org/genproto/googleapis/cloud/dialogflow/v2"
)

var sessionID = "111"
var projectID = "wbt-test-baunjn"

func Process() (string, error) {
	ctx := context.Background()

	sessionClient, err := dialogflow.NewSessionsClient(ctx, option.WithCredentialsFile("/home/igor/Documents/wbt-test-baunjn-b759d424d63a.json"))
	if err != nil {
		return "", err
	}
	defer sessionClient.Close()

	//if projectID == "" || sessionID == "" {
	//	return "", errors.New(fmt.Sprintf("Received empty project (%s) or session (%s)", projectID, sessionID))
	//}

	sessionPath := fmt.Sprintf("projects/%s/agent/sessions/%s", projectID, sessionID)

	streamer, err := sessionClient.StreamingDetectIntent(ctx)
	if err != nil {
		return "", err
	}

	go func() {
		reader := bufio.NewReader(os.Stdin)
		fmt.Println("Wbt test intent")
		fmt.Println("---------------------")

		for {
			fmt.Print("-> ")
			text, _ := reader.ReadString('\n')
			// convert CRLF to LF
			text = strings.Replace(text, "\n", "", -1)

			if strings.Compare("...", text) == 0 {
				fmt.Println("Poka!")
				return
			} else {

				textInput := dialogflowpb.TextInput{Text: text, LanguageCode: "uk"}
				queryTextInput := dialogflowpb.QueryInput_Text{Text: &textInput}
				queryInput := dialogflowpb.QueryInput{Input: &queryTextInput}

				request := dialogflowpb.StreamingDetectIntentRequest{Session: sessionPath, QueryInput: &queryInput}
				err = streamer.Send(&request)
				if err != nil {
					panic(err.Error())
				}

			}

		}
	}()

	//var queryResult *dialogflowpb.QueryResult

	for {
		response, err := streamer.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			panic(err.Error())
		}

		recognitionResult := response.GetRecognitionResult()
		transcript := recognitionResult.GetTranscript()
		log.Printf("Recognition transcript: %s\n", transcript)

		//queryResult = response.GetQueryResult()
		//
		//fmt.Println(queryResult.FulfillmentText)
		//fmt.Println(queryResult.Intent.Name)
	}

	fmt.Println("EXIT")

	return "", nil
}

func init() {

	Process()
}
