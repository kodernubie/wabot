package svc

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"

	"github.com/adlindo/gocom/config"
	_ "github.com/jackc/pgx"
	"google.golang.org/protobuf/proto"

	"github.com/mdp/qrterminal/v3"
	openai "github.com/sashabaranov/go-openai"
	"go.mau.fi/whatsmeow"
	waProto "go.mau.fi/whatsmeow/binary/proto"
	"go.mau.fi/whatsmeow/store"
	"go.mau.fi/whatsmeow/store/sqlstore"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	waLog "go.mau.fi/whatsmeow/util/log"
)

//--------------------------------------------------------------------

type WAClient struct {
	client   *whatsmeow.Client
	device   *store.Device
	myPrefix string
}

func (o *WAClient) getJID(addr string) (types.JID, error) {

	if addr[0] == '+' {
		addr = addr[1:]
	}

	if !strings.ContainsRune(addr, '@') {

		return types.NewJID(addr, types.DefaultUserServer), nil
	} else {

		recipient, err := types.ParseJID(addr)

		if err != nil {

			fmt.Printf("Invalid JID %s: %v \r\n", addr, err)
			return recipient, fmt.Errorf("Invalid JID %s: %v", addr, err)

		} else if recipient.User == "" {

			fmt.Printf("Invalid JID %s: no server specified \r\n", addr)
			return recipient, fmt.Errorf("Invalid JID %s: no server specified", addr)
		}

		return recipient, nil
	}
}

func (o *WAClient) sendTextByJID(to types.JID, mention *types.JID, msgText string) error {

	var msg *waProto.Message

	if mention == nil {
		msg = &waProto.Message{
			Conversation: proto.String(msgText),
		}
	} else {
		msg = &waProto.Message{
			ExtendedTextMessage: &waProto.ExtendedTextMessage{
				Text: proto.String(msgText),
				ContextInfo: &waProto.ContextInfo{
					MentionedJid: []string{
						mention.ToNonAD().String(),
					},
				},
			},
		}
	}

	_, err := o.client.SendMessage(context.Background(), to, msg)

	return err
}

func pprint(data interface{}) {

	b, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		fmt.Println(err)
	}
	fmt.Print(string(b))
}

func (o *WAClient) getAnswer(question string) (string, error) {

	resp, err := openaiClient.CreateChatCompletion(
		context.Background(),
		openai.ChatCompletionRequest{
			Model:       openai.GPT3Dot5Turbo,
			MaxTokens:   100,
			Temperature: 0.6,
			Messages: []openai.ChatCompletionMessage{
				{
					Role:    openai.ChatMessageRoleUser,
					Content: question,
				},
			},
		},
	)

	if err != nil {
		fmt.Printf("ChatCompletion error: %v\n", err)
		return "", fmt.Errorf("GPT error: %v\n", err)
	}

	return resp.Choices[0].Message.Content, nil
}

func (o *WAClient) Start() {

	clientLog := waLog.Stdout("Client", "INFO", true)
	o.client = whatsmeow.NewClient(o.device, clientLog)

	o.myPrefix = o.device.ID.String()
	o.myPrefix = "@" + o.myPrefix[0:strings.Index(o.myPrefix, ".")]

	o.client.AddEventHandler(func(evt interface{}) {

		switch v := evt.(type) {
		case *events.Message:

			if v.Info.IsFromMe {
				return
			}

			if v.Info.IsGroup {

				if v.Message.ExtendedTextMessage != nil &&
					strings.HasPrefix(*v.Message.ExtendedTextMessage.Text, o.myPrefix) {

					prompt := *v.Message.ExtendedTextMessage.Text
					prompt = prompt[len(o.myPrefix):]

					ans, err := o.getAnswer(prompt)

					if err != nil {
						o.sendTextByJID(v.Info.Chat, &v.Info.Sender, "Maaf programmer saya kurang handal, ada kendala di backend")
						return
					}

					o.sendTextByJID(v.Info.Chat, &v.Info.Sender, ans)
				}
			} else {

				prompt := v.Message.GetConversation()
				ans, err := o.getAnswer(prompt)

				if err != nil {
					o.sendTextByJID(v.Info.Chat, nil, "Maaf programmer saya kurang handal, ada kendala di backend")
					return
				}

				o.sendTextByJID(v.Info.Chat, nil, ans)
			}
		}
	})

	err := o.client.Connect()
	if err != nil {
		fmt.Println("Unable to connect to device ")
		panic(err)
	}
}

//--------------------------------------------------------------------

type WASvc struct {
	container *sqlstore.Container
	clients   map[string]*WAClient
}

var waSvc *WASvc
var waSvcOnce sync.Once
var openaiClient *openai.Client

func GetWASvc() *WASvc {

	waSvcOnce.Do(func() {

		waSvc = &WASvc{
			clients: map[string]*WAClient{},
		}
		waSvc.Start()

		openaiClient = openai.NewClient(config.Get("app.openai.key"))
	})

	return waSvc
}

func (o *WASvc) Start() {

	fmt.Println("WA BOT Starting...")

	var err error
	dbLog := waLog.Stdout("Database", "INFO", true)
	o.container, err = sqlstore.New("pgx", config.Get("app.wa.dburl"), dbLog)

	if err != nil {
		fmt.Println("Unable to get wa db store")
		panic(err)
	}

	deviceList, err := o.container.GetAllDevices()

	if err != nil {
		fmt.Println("Unable to get device list")
		panic(err)
	}

	for no, dev := range deviceList {

		fmt.Println(no, ":==> ", dev.ID)
		if dev.ID != nil {

			newClient := &WAClient{
				device: dev,
			}

			newClient.Start()

			o.clients[dev.ID.String()] = newClient
		}
	}
	fmt.Println("=====>>>> ", len(deviceList))
}

func (o *WASvc) NeWDevice() (string, error) {

	dev := o.container.NewDevice()

	clientLog := waLog.Stdout("Client", "INFO", true)
	client := whatsmeow.NewClient(dev, clientLog)
	client.AddEventHandler(func(evt interface{}) {

		fmt.Println("LOGGG ::: ", evt)
	})

	qrChan, _ := client.GetQRChannel(context.Background())
	err := client.Connect()
	if err != nil {
		fmt.Println("connect to cliet qr error ", err)
		panic(err)
	}

	for evt := range qrChan {
		if evt.Event == "code" {
			fmt.Println("QR code:", evt.Code)

			qrterminal.GenerateHalfBlock(evt.Code, qrterminal.L, os.Stdout)
		} else {
			fmt.Println("Login event:", evt.Event)
		}
	}

	fmt.Println("setelah...")

	return "", nil
}
