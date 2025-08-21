package flow

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/webitel/flow_manager/gen/ai_bots"
	"github.com/webitel/flow_manager/gen/workflow"
	"github.com/webitel/flow_manager/model"
	"google.golang.org/protobuf/types/known/structpb"
	"maps"
	"net/http"
)

type GeminiPart struct {
	Text    string `json:"text"`
	Through bool   `json:"through"`
}

type GeminiMessage struct {
	Parts []GeminiPart `json:"parts"`
	Role  string       `json:"role"`
}

type BotTimeout struct {
	Actions model.Applications `json:"_"`
	Sec     int32              `json:"sec"`
}

type BotParams struct {
	Profile struct {
		Id int64 `json:"id"`
	} `json:"profile"`
	Timeout       map[string]BotTimeout `json:"timeout"`
	Functions     []*BotFunction        `json:"functions"`
	TranscribeVar string                `json:"transcribeVar"`
	//TranscribeModel  string                `json:"transcribeModel"`
	//TranscribeClient string                `json:"transcribeClient"`
	Variables    map[string]string `json:"variables"`
	StartMessage string            `json:"startMessage"`
	Model        string            `json:"model"`
}

type GeminiVad struct {
	Enabled                  bool   `json:"enabled"`
	StartOfSpeechSensitivity string `json:"startOfSpeechSensitivity"`
	EndOfSpeechSensitivity   string `json:"endOfSpeechSensitivity"`
	PrefixPaddingMs          int32  `json:"prefixPaddingMs"`
	SilenceDurationMs        int32  `json:"silenceDurationMs"`
}

type Gemini struct {
	BotParams
	Rate              string        `json:"rate"`
	SystemInstruction GeminiMessage `json:"systemInstruction"`
	Prompt            string        `json:"prompt"`
	VoiceName         string        `json:"voiceName"`
	MediaResolution   string        `json:"mediaResolution"`
	Temperature       float32       `json:"temperature"`
	Language          string        `json:"language"`
	Vad               *GeminiVad    `json:"vad"`
	SessionResumption *struct {
		Handle      string `json:"handle"`
		Transparent bool   `json:"transparent"`
	} `json:"sessionResumption"`
}

type BotFunction struct {
	Actions     model.Applications `json:"_"`
	Behavior    string             `json:"behavior"`
	Name        string             `json:"name"`
	Description string             `json:"description"`
	Parameters  any                `json:"parameters"`
}

func (r *router) withApplications(apps ApplicationHandlers) Router {
	r2 := *r
	r2.apps = maps.Clone(r.apps)

	for k, v := range apps {
		r2.apps[k] = v
	}

	return &r2
}

type BotResult struct {
	Exit         bool           `json:"exit"`
	WillContinue bool           `json:"willContinue"`
	Response     map[string]any `json:"response"`
	Scheduling   string         `json:"scheduling"`
	Content      string         `json:"content"`
	Tts          *struct {
		Model    string `json:"model"`
		Text     string `json:"text"`
		StopTalk bool   `json:"stopTalk"`
	} `json:"tts"`
}

type Embed struct {
	Profile struct {
		Id int64 `json:"id"`
	} `json:"profile"`
	Query     string  `json:"query"`
	Threshold float32 `json:"threshold"`
	Limit     int32   `json:"limit"`
	Set       string
}

func (r *router) embed(ctx context.Context, scope *Flow, conn model.Connection, args interface{}) (model.Response, *model.AppError) {
	var argv Embed
	if err := scope.Decode(args, &argv); err != nil {
		return model.CallResponseError, err
	}

	embed, err := r.fm.AiBots.Embed().GetContent(ctx, &ai_bots.GetContentRequest{
		DomainId:  conn.DomainId(),
		ProfileId: argv.Profile.Id,
		Query:     argv.Query,
		Threshold: argv.Threshold,
		Limit:     argv.Limit,
	})
	if err != nil {
		return model.CallResponseError, model.NewAppError("embed", "bot.embed.get_content", nil, err.Error(), http.StatusInternalServerError)
	}

	// The content from the embedding service is likely raw text, not base64.
	// Encoding it here might be unexpected for the user.
	// If the content is indeed binary, this is correct. Otherwise, this line might be removed.
	encodedContent := base64.StdEncoding.EncodeToString([]byte(embed.Content))

	return conn.Set(ctx, model.Variables{
		argv.Set: encodedContent,
	})
}

func (r *router) botReturnResult(ctx context.Context, scope *Flow, conn model.Connection, args interface{}) (model.Response, *model.AppError) {
	var argv BotResult

	argsArr, ok := args.([]any)
	if !ok || len(argsArr) < 2 {
		return model.CallResponseError, model.NewAppError("botReturnResult", "bot.return.args", nil, "invalid arguments", http.StatusBadRequest)
	}

	if err := scope.Decode(argsArr[0], &argv); err != nil {
		return model.CallResponseError, err
	}

	content := argv.Content
	if contentBytes, err := base64.StdEncoding.DecodeString(argv.Content); err == nil {
		content = string(contentBytes)
	}

	scope.SetCancel()
	chanResult, ok := argsArr[1].(chan workflow.BotExecuteResponse)
	if !ok {
		return model.CallResponseError, model.NewAppError("botReturnResult", "bot.return.channel", nil, "result channel is of wrong type", http.StatusInternalServerError)
	}

	result := workflow.BotExecuteResponse{
		WillContinue: argv.WillContinue,
		Scheduling:   argv.Scheduling,
		Exit:         argv.Exit,
	}
	if argv.Response == nil {
		argv.Response = make(map[string]any)
	}

	if content != "" {
		argv.Response["output"] = content
	}

	var err error
	result.Response, err = structpb.NewStruct(argv.Response)
	if err != nil {
		return model.CallResponseError, model.NewAppError("botReturnResult", "bot.return.struct", nil, err.Error(), http.StatusInternalServerError)
	}

	if argv.Tts != nil {
		result.Tts = &workflow.BotExecuteResponse_TTS{
			Model:    argv.Tts.Model,
			Text:     argv.Tts.Text,
			StopTalk: argv.Tts.StopTalk,
		}
	}

	go func() {
		select {
		case chanResult <- result:
		case <-ctx.Done():
			return
		}
	}()

	return model.CallResponseOK, nil
}

// executeAndGetResponse is a helper to run a sub-flow (fork) for a function or timeout
// and wait for its result via the 'returnResult' application.
func (r *router) executeAndGetResponse(ctx context.Context, scope *Flow, conn model.Connection, forkName string, actions model.Applications) (*workflow.BotExecuteResponse, error) {
	funcResp := make(chan workflow.BotExecuteResponse, 1)
	defer close(funcResp)

	scope.handler = scope.handler.AddApplications(map[string]*Application{
		"returnResult": {
			AllowNoConnect: false,
			Handler:        r.doExecute(r.botReturnResult),
			ArgsParser: func(resultChan chan workflow.BotExecuteResponse) ApplicationArgsParser {
				return func(c model.Connection, args ...any) any {
					// The arguments from the flow are in args[0]. We append our result channel.
					return append(args, resultChan)
				}
			}(funcResp),
		},
		"embed": {
			AllowNoConnect: false,
			Handler:        r.doExecute(r.embed),
		},
	})
	// This router is specific to the callback execution context.

	forkedScope := scope.Fork(forkName, actions)
	Route(ctx, forkedScope, scope.handler)

	// Wait for the result.
	select {
	case response := <-funcResp:
		return &response, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func botOnlyText(conn model.Connection) (bool, *model.AppError) {
	switch conn.Type() {
	case model.ConnectionTypeCall:
		return false, nil
	case model.ConnectionTypeChat:
		return true, nil
	default:
		return false, model.NewRequestError("gemini.bots", fmt.Sprintf("unsupported connection type %d", conn.Type()))
	}
}

func (r *router) gemini(ctx context.Context, scope *Flow, conn model.Connection, args any) (model.Response, *model.AppError) {
	var argv Gemini
	var textBot bool
	var err *model.AppError
	if err = scope.Decode(args, &argv); err != nil {
		return nil, err
	}
	if err = scope.Decode(args, &argv.BotParams); err != nil {
		return nil, err
	}

	textBot, err = botOnlyText(conn)
	if err != nil {
		return model.CallResponseError, err
	}

	channel := "call"
	if textBot {
		channel = "chat"
	}

	jsonFunctions, _ := json.Marshal(argv.Functions)

	initial := &ai_bots.GeminiRequest_Initial{
		TextBot:          textBot,
		Channel:          channel,
		DomainId:         conn.DomainId(),
		FlowConnection:   r.fm.ConnectionString(),
		ProfileId:        argv.Profile.Id,
		CallId:           conn.Id(),
		Functions:        jsonFunctions,
		TranscribeClient: "todo", //argv.TranscribeClient,
		TranscribeModel:  "todo", //argv.TranscribeModel,
		TranscribeVar:    argv.TranscribeVar,

		SystemInstruction: &ai_bots.GeminiRequest_SystemInstruction{
			Role:  argv.SystemInstruction.Role,
			Parts: make([]*ai_bots.GeminiRequest_SystemInstruction_Part, 0, len(argv.SystemInstruction.Parts)),
		},
		VoiceName:       argv.VoiceName,
		Language:        argv.Language,
		Temperature:     argv.Temperature,
		MediaResolution: argv.MediaResolution,
		Model:           argv.Model,
		StartMessage:    argv.StartMessage,
	}

	if argv.SessionResumption != nil {
		initial.SessionResumption = &ai_bots.GeminiRequest_Initial_SessionResumption{
			Handle:      argv.SessionResumption.Handle,
			Transparent: argv.SessionResumption.Transparent,
		}
	}

	for _, v := range argv.SystemInstruction.Parts {
		initial.SystemInstruction.Parts = append(initial.SystemInstruction.Parts, &ai_bots.GeminiRequest_SystemInstruction_Part{
			Thought: v.Through,
			Text:    v.Text,
		})
	}

	if argv.Vad != nil && argv.Vad.Enabled {
		initial.Vad = &ai_bots.GeminiRequest_VAD{
			StartOfSpeechSensitivity: argv.Vad.StartOfSpeechSensitivity,
			EndOfSpeechSensitivity:   argv.Vad.EndOfSpeechSensitivity,
			PrefixPaddingMs:          argv.Vad.PrefixPaddingMs,
			SilenceDurationMs:        argv.Vad.SilenceDurationMs,
		}
	}

	m := args.(map[string]any)
	if err = argv.setupFunctions(m); err != nil {
		return nil, err
	}

	if t, ok := m["timeout"].(map[string]any); argv.Timeout != nil && ok {
		for k, v := range t {
			obj := v.(map[string]any)
			if tf, ok := argv.Timeout[k]; ok {
				tf.Actions, err = parseOutputs("actions", obj)
				argv.Timeout[k] = tf
				initial.Events = append(initial.Events, &ai_bots.GeminiRequest_Initial_TimeoutEvent{
					Name: k,
					Sec:  tf.Sec,
				})
			}
		}
	}

	res, e := r.fm.AiBots.Bot().Gemini(ctx, &ai_bots.GeminiRequest{
		Input: &ai_bots.GeminiRequest_Initial_{
			Initial: initial,
		},
	})
	if e != nil {
		return nil, model.NewInternalError("gemini.connect", e.Error())
	}

	var connection, dialogId string
	switch out := res.Output.(type) {
	case *ai_bots.GeminiResponse_Connected:
		connection = out.Connected.Connection
		dialogId = out.Connected.DialogId
	default:
		return nil, model.NewAppError("gemini", "gemini.gemini", nil, "failed to get connected response", http.StatusInternalServerError)
	}

	return r.bot(ctx, scope, conn, connection, dialogId, argv.BotParams)
}

type OpenAi struct {
	BotParams
	Instructions []struct {
		Text string `json:"text"`
	} `json:"instructions"`
	Voice                   string                                                 `json:"voice"`
	Language                string                                                 `json:"language"`
	InputAudioTranscription *ai_bots.OpenAIRequest_Initial_InputAudioTranscription `json:"inputAudioTranscription"`
	TurnDetection           *struct {
		Threshold         float32 `json:"threshold"`
		PrefixPaddingMs   int32   `json:"prefixPaddingMs"`
		SilenceDurationMs int32   `json:"silenceDurationMs"`
	} `json:"turnDetection"`
	ToolChoice               string  `json:"toolChoice"`
	Temperature              float32 `json:"temperature"`
	MaxResponseOutputTokens  int32   `json:"maxResponseOutputTokens"`
	InputAudioNoiseReduction string  `json:"inputAudioNoiseReduction"`
}

func (r *router) openai(ctx context.Context, scope *Flow, conn model.Connection, args any) (model.Response, *model.AppError) {
	var argv OpenAi
	var textBot bool
	var err *model.AppError
	if err = scope.Decode(args, &argv); err != nil {
		return nil, err
	}
	if err = scope.Decode(args, &argv.BotParams); err != nil {
		return nil, err
	}

	textBot, err = botOnlyText(conn)
	if err != nil {
		return model.CallResponseError, err
	}

	channel := "call"
	if textBot {
		channel = "chat"
	}

	jsonFunctions, _ := json.Marshal(argv.Functions)
	instructions := ""

	var l int
	for i, v := range argv.Instructions {
		if v.Text == "" {
			continue
		}
		if i > 0 && l > 0 && instructions[l-1] != '.' {
			instructions += ". "
		}
		instructions += v.Text
		l += len(instructions)
	}

	initial := &ai_bots.OpenAIRequest_Initial{
		FlowConnection: r.fm.ConnectionString(),
		DomainId:       conn.DomainId(),
		ProfileId:      argv.Profile.Id,
		TextBot:        textBot,
		Channel:        channel,
		CallId:         conn.Id(),
		//TranscribeClient: argv.TranscribeClient,
		//TranscribeModel:  argv.TranscribeModel,
		TranscribeVar: argv.TranscribeVar,
		Functions:     jsonFunctions,
		StartMessage:  argv.StartMessage,
		Model:         argv.Model,

		Instructions:             instructions,
		Language:                 argv.Language,
		Voice:                    argv.Voice,
		InputAudioTranscription:  argv.InputAudioTranscription,
		TurnDetection:            nil,
		ToolChoice:               argv.ToolChoice,
		Temperature:              argv.Temperature,
		MaxResponseOutputTokens:  argv.MaxResponseOutputTokens,
		InputAudioNoiseReduction: argv.InputAudioNoiseReduction,
	}

	if argv.TurnDetection != nil {
		initial.TurnDetection = &ai_bots.OpenAIRequest_Initial_TurnDetection{
			Threshold:         argv.TurnDetection.Threshold,
			PrefixPaddingMs:   argv.TurnDetection.PrefixPaddingMs,
			SilenceDurationMs: argv.TurnDetection.SilenceDurationMs,
		}
	}

	m := args.(map[string]any)
	if err = argv.setupFunctions(m); err != nil {
		return nil, err
	}
	if t, ok := m["timeout"].(map[string]any); argv.Timeout != nil && ok {
		for k, v := range t {
			obj := v.(map[string]any)
			if tf, ok := argv.Timeout[k]; ok {
				tf.Actions, err = parseOutputs("actions", obj)
				argv.Timeout[k] = tf
				initial.Events = append(initial.Events, &ai_bots.OpenAIRequest_Initial_TimeoutEvent{
					Name: k,
					Sec:  tf.Sec,
				})
			}
		}
	}

	res, e := r.fm.AiBots.Bot().OpenAI(ctx, &ai_bots.OpenAIRequest{
		Input: &ai_bots.OpenAIRequest_Initial_{
			Initial: initial,
		},
	})
	if e != nil {
		return nil, model.NewInternalError("bot.openai.connect", e.Error())
	}

	var connection, dialogId string
	switch out := res.Output.(type) {
	case *ai_bots.OpenAIResponse_Connected:
		connection = out.Connected.Connection
		dialogId = out.Connected.DialogId
	default:
		return nil, model.NewAppError("openai", "bot.openai", nil, "failed to get connected response", http.StatusInternalServerError)
	}

	return r.bot(ctx, scope, conn, connection, dialogId, argv.BotParams)
}

func (r *router) bot(ctx context.Context, scope *Flow, conn model.Connection, connection, dialogId string, argv BotParams) (model.Response, *model.AppError) {

	if len(argv.Functions) > 0 || len(argv.Timeout) > 0 {
		cb := func(ctx context.Context, v any) (any, error) {
			req, ok := v.(*workflow.BotExecuteRequest)
			if !ok {
				return nil, errors.New("unknown request type for callback")
			}

			switch data := req.Data.(type) {
			case *workflow.BotExecuteRequest_Timeout_:
				timeout, ok := argv.Timeout[data.Timeout.Event]
				if !ok {
					return nil, model.NewAppError("bot", "bot.timeout", nil, "timeout event not found", http.StatusInternalServerError)
				}
				return r.executeAndGetResponse(ctx, scope, conn, "function-timeout", timeout.Actions)

			case *workflow.BotExecuteRequest_Function_:
				exec := data.Function
				fn, err := argv.getFunctionByName(exec.Name)
				if err != nil {
					conn.Log().Error(err.Error())
					return nil, err
				}

				if exec.Args != nil {
					conn.Set(ctx, exec.Args.AsMap())
				}
				return r.executeAndGetResponse(ctx, scope, conn, fmt.Sprintf("function-%s", fn.Name), fn.Actions)
			default:
				return nil, errors.New("unknown data type in VoiceBotExecuteRequest")
			}
		}
		r.fm.Callback().Register(dialogId, cb)
		defer r.fm.Callback().Unregister(dialogId)
	}

	switch conn.Type() {
	case model.ConnectionTypeCall:
		return conn.(model.Call).Bot(ctx, connection, 16000, dialogId, argv.Variables)
	case model.ConnectionTypeChat:
		//ctx2 := r.fm.AiBots.WithConnection(ctx, connection)
		return conn.(model.Conversation).Bot(ctx, r.fm.AiBots.Converse(), dialogId)
	default:
		return model.CallResponseError, nil
	}
}

func parseOutputs(propertyName string, props map[string]any) (model.Applications, *model.AppError) {
	if apps, ok := props[propertyName].([]any); ok {
		return ArrInterfaceToArrayApplication(apps), nil
	}
	return nil, model.NewAppError("Voice.Parse", "voice.valid.props", nil, fmt.Sprintf("bad arguments %v", props), http.StatusBadRequest)
}

func (bp *BotParams) setupFunctions(m map[string]any) (err *model.AppError) {
	//m := args.(map[string]any)
	if f, ok := m["functions"].([]any); bp.Functions != nil && ok {
		for i, fn := range f {
			fnObj := fn.(map[string]any)
			name, _ := fnObj["name"].(string)
			if bp.Functions[i].Name != name {
				return ErrorRequiredParameter("bot", "function name")
			}
			bp.Functions[i].Actions, err = parseOutputs("actions", fnObj)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (g *BotParams) getFunctionByName(name string) (*BotFunction, error) {
	for _, f := range g.Functions {
		if f.Name == name {
			return f, nil
		}
	}
	return nil, fmt.Errorf("function %s not found", name)
}
