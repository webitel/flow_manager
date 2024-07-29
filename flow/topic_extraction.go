package flow

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/webitel/flow_manager/model"
	"io"
	"net/http"
	"time"
)

type TopicMessage struct {
	Sender  string `json:"sender"`
	Message string `json:"message"`
}

type TopicExtractionRequest struct {
	PossibleTopics []string       `json:"possible_topics"`
	Messages       []TopicMessage `json:"messages"`
}

type TopicExtractionResponse struct {
	Topics     []string `json:"topics"`
	Confidence float32  `json:"confidence"`
}

type TopicExtraction struct {
	Connection     string         `json:"connection"`
	PossibleTopics []string       `json:"possibleTopics"`
	Limit          int            `json:"limit"`
	Messages       []TopicMessage `json:"messages"`

	Score        string `json:"score"`
	DefinedTopic string `json:"definedTopic"`
}

func (r *router) topicExtraction(ctx context.Context, scope *Flow, c model.Connection, args interface{}) (model.Response, *model.AppError) {
	var req *http.Request
	var resp *http.Response

	argv := TopicExtraction{
		Limit: 4,
	}

	if err := scope.Decode(args, &argv); err != nil {
		return nil, err
	}

	if len(argv.Connection) == 0 {
		return model.CallResponseError, ErrorRequiredParameter("topicExtraction", "connection")
	}

	if len(argv.PossibleTopics) == 0 {
		return model.CallResponseError, ErrorRequiredParameter("topicExtraction", "possibleTopics")
	}

	data := TopicExtractionRequest{
		PossibleTopics: argv.PossibleTopics,
	}

	switch conn := c.(type) {
	case model.Call:
		msg := conn.SpeechMessages(argv.Limit)
		if len(msg) == 0 {
			break
		}
		data.Messages = make([]TopicMessage, 0, len(msg))
		for _, v := range msg {
			if v.Question != "" {
				data.Messages = append(data.Messages, TopicMessage{
					Message: v.Question,
					Sender:  "operator",
				})
			}

			argv.Messages = append(data.Messages, TopicMessage{
				Message: v.Answer,
				Sender:  "user",
			})
		}
	case model.Conversation:
		msg := conn.LastMessages(argv.Limit)
		if len(msg) == 0 {
			break
		}
		data.Messages = make([]TopicMessage, 0, len(msg))
		for _, v := range msg {
			m := TopicMessage{
				Message: v.Text,
			}
			if v.User == "" {
				m.Sender = "operator"
			} else {
				m.Sender = "user"
			}
			data.Messages = append(data.Messages, m)
		}
	default:
		if len(argv.Messages) != 0 {
			data.Messages = argv.Messages
		}
	}

	if len(data.Messages) == 0 {
		return model.CallResponseError, ErrorRequiredParameter("topicExtraction", "messages")
	}

	var bodyRequest, bodyResponse []byte
	var err error
	var result TopicExtractionResponse
	bodyRequest, err = json.Marshal(data)
	if err != nil {
		return model.CallResponseError, Error("topicExtraction", err)
	}

	req, err = http.NewRequest("POST", argv.Connection, bytes.NewBuffer(bodyRequest))
	if err != nil {
		return model.CallResponseError, Error("topicExtraction", err)
	}

	req.Header.Add("Content-Type", "application/json")
	client := &http.Client{
		Timeout: time.Second * 10,
	}
	resp, err = client.Do(req)
	if err != nil {
		return model.CallResponseError, Error("topicExtraction", err)
	}

	defer resp.Body.Close()

	bodyResponse, err = io.ReadAll(resp.Body)
	if err != nil {
		return model.CallResponseError, Error("topicExtraction", err)
	}

	err = json.Unmarshal(bodyResponse, &result)
	if err != nil {
		return model.CallResponseError, Error("topicExtraction", err)
	}

	set := make(model.Variables)
	if argv.Score != "" {
		set[argv.Score] = fmt.Sprintf("%v", result.Confidence)
	}
	if argv.DefinedTopic != "" {
		if len(result.Topics) == 0 {
			set[argv.DefinedTopic] = ""
		} else {
			set[argv.DefinedTopic] = result.Topics[0]
		}

	}

	return c.Set(ctx, set)
}
