package call

import (
	dialogflow "cloud.google.com/go/dialogflow/apiv2"
	"context"
	"fmt"
	"github.com/webitel/flow_manager/flow"
	"github.com/webitel/flow_manager/model"
	"github.com/webitel/flow_manager/voice/fs"
	"google.golang.org/api/option"
	dialogflowpb "google.golang.org/genproto/googleapis/cloud/dialogflow/v2"
	"google.golang.org/grpc"
	"io"
	"log"
	"time"
)

type Voice int

type Client struct {
	cli         *dialogflow.SessionsClient
	sessionPath string
	stream      dialogflowpb.Sessions_StreamingDetectIntentClient
	r           *Router
}

var projectID = "wbt-test-baunjn"

func NewCli(ctx context.Context, r *Router, sessionID string) (*Client, error) {
	sessionClient, err := dialogflow.NewSessionsClient(ctx, option.WithCredentialsFile("/home/igor/Documents/wbt-test-baunjn-b759d424d63a.json"))
	if err != nil {
		return nil, err
	}

	sessionPath := fmt.Sprintf("projects/%s/agent/sessions/%s", projectID, sessionID)

	streamer, err := sessionClient.StreamingDetectIntent(ctx)
	if err != nil {
		return nil, err
	}
	cli := &Client{
		cli:         sessionClient,
		sessionPath: sessionPath,
		stream:      streamer,
		r:           r,
	}

	audioConfig := dialogflowpb.InputAudioConfig{
		AudioEncoding:   dialogflowpb.AudioEncoding_AUDIO_ENCODING_LINEAR_16,
		SampleRateHertz: 16000,
		LanguageCode:    "uk",
		SingleUtterance: false,
	}

	queryAudioInput := dialogflowpb.QueryInput_AudioConfig{AudioConfig: &audioConfig}

	queryInput := dialogflowpb.QueryInput{Input: &queryAudioInput}

	request := dialogflowpb.StreamingDetectIntentRequest{Session: sessionPath, QueryInput: &queryInput}
	err = streamer.Send(&request)
	if err != nil {
		fmt.Println(err.Error())
		return nil, err
	}

	return cli, nil
}

func (c *Client) Close() {
	c.cli.Close()
}

func (c *Client) Send(audioBytes []byte) error {
	request := dialogflowpb.StreamingDetectIntentRequest{InputAudio: audioBytes}
	return c.stream.Send(&request)
}

func (c *Client) Recive() {
	defer fmt.Println("CLOSE RECIVER")

	var qr *dialogflowpb.QueryResult

	for {
		response, err := c.stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			fmt.Println(err.Error())
			break
		}

		recognitionResult := response.GetRecognitionResult()
		transcript := recognitionResult.GetTranscript()
		log.Printf("Recognition transcript: %s\n", transcript)

		if response.GetQueryResult() != nil {
			qr = response.GetQueryResult()
		}

	}
	if qr != nil {
		fmt.Println("INTENT ", qr.Intent.Name)
	}
}

func (r *Router) voice(ctx context.Context, scope *flow.Flow, call model.Call, args interface{}) (model.Response, *model.AppError) {
	var timeout int
	if err2 := r.Decode(scope, args, &timeout); err2 != nil {
		return nil, err2
	}

	//call.Voice(ctx, 100)
	client, err := grpc.Dial("10.10.10.25:50051", grpc.WithInsecure(), grpc.WithBlock(), grpc.WithTimeout(2*time.Second))
	if err != nil {
		fmt.Println(err.Error())
		return model.CallResponseError, nil
	}
	apiCLi := fs.NewApiClient(client)

	stream, err := apiCLi.VoiceStream(ctx)
	if err != nil {
		fmt.Println(err.Error())
		return model.CallResponseError, nil
	}

	stream.Send(&fs.VoiceStreamRequest{
		Data: &fs.VoiceStreamRequest_Request{
			Request: &fs.VoiceStreamRequest_Init{
				Id: call.Id(),
			},
		},
	})
	cli, err := NewCli(ctx, r, call.Id())
	if err != nil {
		fmt.Println(err.Error())
		return model.CallResponseError, nil
	}
	defer cli.Close()

	go cli.Recive()

	var msg *fs.VoiceStreamResponse
	count := 0
	for {
		msg, err = stream.Recv()
		if err != nil {
			fmt.Println(err.Error())
			break

		}

		r, _ := msg.Data.(*fs.VoiceStreamResponse_Chunk_)
		//fmt.Println("receive ", len(r.Chunk.Content))

		err = stream.Send(&fs.VoiceStreamRequest{
			Data: &fs.VoiceStreamRequest_Chunk_{
				Chunk: &fs.VoiceStreamRequest_Chunk{
					Content: r.Chunk.Content,
				},
			},
		})
		if err != nil {
			fmt.Println(err.Error())
			break

		}
		if r != nil {
			err = cli.Send(r.Chunk.Content)
			count++
			if err != nil {
				fmt.Println(err.Error())
				break
			}
		}

		if count > 3000 {
			cli.stream.CloseSend()
			break
		}
	}
	stream.CloseSend()
	fmt.Println(count)
	time.Sleep(time.Second * 2)
	return call.Hangup(ctx, "")
}
