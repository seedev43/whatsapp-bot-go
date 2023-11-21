/*
###################################
# Name: Mywa BOT                  #
# Version: 1.0.1                  #
# Developer: Amirul Dev           #
# Library: waSocket               #
# Contact: 085157489446           #
###################################
# Thanks to:
# Vnia
*/
package lib

import (
	"bytes"
	"context"
	"fmt"
	"gowabot/system/dto"
	"image/jpeg"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/amiruldev20/waSocket"
	waProto "github.com/amiruldev20/waSocket/binary/proto"
	"github.com/amiruldev20/waSocket/types"
	"github.com/amiruldev20/waSocket/types/events"
	"github.com/nickalie/go-webpbin"

	"google.golang.org/protobuf/proto"
)

type renz struct {
	sock *waSocket.Client
	Msg  *events.Message
}

func NewSimp(Cli *waSocket.Client, m *events.Message) *renz {
	return &renz{
		sock: Cli,
		Msg:  m,
	}
}

/* parse jid */
func (m *renz) parseJID(arg string) (types.JID, bool) {
	if arg[0] == '+' {
		arg = arg[1:]
	}
	if !strings.ContainsRune(arg, '@') {
		return types.NewJID(arg, types.DefaultUserServer), true
	} else {
		recipient,
			err := types.ParseJID(arg)
		if err != nil {
			fmt.Printf("Invalid JID %s: %v\n", arg, err)
			return recipient, false
		} else if recipient.User == "" {
			fmt.Printf("Invalid JID %s: no server specified\n", arg)
			return recipient, false
		}
		return recipient,
			true
	}
}

/* send react */
func (m *renz) React(react string) {
	_,
		err := m.sock.SendMessage(context.Background(), m.Msg.Info.Chat, m.sock.BuildReaction(m.Msg.Info.Chat, m.Msg.Info.Sender, m.Msg.Info.ID, react))
	if err != nil {
		return
	}
}

/* send message */
func (m *renz) SendMsg(jid types.JID, teks string) {
	_,
		err := m.sock.SendMessage(context.Background(), jid, &waProto.Message{Conversation: proto.String(teks)})
	if err != nil {
		return
	}
}

/* send sticker */
func (m *renz) SendSticker(jid types.JID, data []byte, extra ...dto.ExtraSend) {
	var contextInfo *waProto.ContextInfo
	var req dto.ExtraSend
	if len(extra) > 1 {
		log.Println("only one extra parameter may be provided to SendMessage")
		return
	} else if len(extra) == 1 {
		req = extra[0]
	}

	if req.Reply {
		// Isi contextInfo jika Reply adalah true
		contextInfo = &waProto.ContextInfo{
			Expiration:    proto.Uint32(86400),
			StanzaId:      &m.Msg.Info.ID,
			Participant:   proto.String(m.Msg.Info.Sender.String()),
			QuotedMessage: m.Msg.Message,
		}
	}

	randomJpgImg := "./temp/" + GenerateRandomString(5) + ".jpg"
	randomWebpImg := "./temp/" + GenerateRandomString(5) + ".webp"
	if err := os.WriteFile(randomJpgImg, data, 0600); err != nil {
		log.Printf("Failed to save image: %v", err)
		return
	}

	log.Printf("Saved image in %s", randomJpgImg)

	imgbyte, err := os.ReadFile(randomJpgImg)
	if err != nil {
		fmt.Println("Error reading file:", err)
		return
	}

	decodeImg, err := jpeg.Decode(bytes.NewReader(imgbyte))
	if err != nil {
		fmt.Println("Error decoding file:", err)
		return
	}

	fmt.Println("convert jpg to webp...")
	f, err := os.Create(randomWebpImg)

	if err != nil {
		log.Println(err)
		return
	}

	if err := webpbin.Encode(f, decodeImg); err != nil {
		f.Close()
		log.Println(err)
		return
	}

	if err := f.Close(); err != nil {
		log.Println(err)
		return
	}

	fmt.Println("Success convert to webp")
	webpByte, err := os.ReadFile(randomWebpImg)
	if err != nil {
		fmt.Println("Error reading file:", err)
		return
	}

	fmt.Println("Sending webp as sticker...")

	uploadImg, err := m.sock.Upload(context.Background(), webpByte, waSocket.MediaImage)

	if err != nil {
		log.Println(err)
		return
	}

	_, err = m.sock.SendMessage(context.Background(), m.Msg.Info.Chat, &waProto.Message{
		StickerMessage: &waProto.StickerMessage{
			Url:           proto.String(uploadImg.URL),
			FileSha256:    uploadImg.FileSHA256,
			FileEncSha256: uploadImg.FileEncSHA256,
			MediaKey:      uploadImg.MediaKey,
			Mimetype:      proto.String(http.DetectContentType(webpByte)),
			DirectPath:    proto.String(uploadImg.DirectPath),
			FileLength:    proto.Uint64(uint64(len(webpByte))),
			ContextInfo:   contextInfo,
			// FirstFrameSidecar: webpByte,
			// PngThumbnail:      webpByte,
		},
	})

	// delete file image
	err = os.Remove(randomJpgImg)
	err = os.Remove(randomWebpImg)

	if err != nil {
		log.Println(err)
		return
	}
}

/* send image */
func (m *renz) SendImg(jid types.JID, value interface{}) {
	var uploadImg waSocket.UploadResponse
	var imgByte []byte
	randomJpgImg := "./temp/" + GenerateRandomString(5) + ".jpg"

	if imageUrl, ok := value.(string); ok {
		_, err := url.ParseRequestURI(imageUrl)
		if err != nil {
			log.Println("Invalid url")
			return
		}

		if !IsValidImageURL(imageUrl) {
			log.Println("Invalid image url")
			return
		}

		res, err := http.Get(imageUrl)
		if err != nil {
			log.Println("Failed to fetch image:", err)
			return
		}
		defer res.Body.Close() // Pastikan menutup body setelah selesai

		file, err := os.Create(randomJpgImg)
		if err != nil {
			log.Println("Failed to create file:", err)
			return
		}
		defer file.Close() // Pastikan menutup file setelah selesai

		// menghindari membaca seluruh respons HTTP body (dari res.Body) ke dalam memori secara keseluruhan
		// source: chatgpt :v
		_, err = io.Copy(file, res.Body)
		if err != nil {
			log.Println("Failed to copy image:", err)
			return
		}

		imgByte, err = os.ReadFile(randomJpgImg)
		if err != nil {
			log.Println("Cannot read file image:", err)
			return
		}

		uploadImg, err = m.sock.Upload(context.Background(), imgByte, waSocket.MediaImage)
		if err != nil {
			log.Println("Failed to upload file:", err)
			return
		}

	}

	_, err := m.sock.SendMessage(context.Background(), m.Msg.Info.Chat, &waProto.Message{
		ImageMessage: &waProto.ImageMessage{
			JpegThumbnail: imgByte, // blm work
			Url:           proto.String(uploadImg.URL),
			DirectPath:    proto.String(uploadImg.DirectPath),
			MediaKey:      uploadImg.MediaKey,
			Mimetype:      proto.String(http.DetectContentType(imgByte)),
			FileEncSha256: uploadImg.FileEncSHA256,
			FileSha256:    uploadImg.FileSHA256,
			FileLength:    proto.Uint64(uint64(len(imgByte))),
		},
	})

	if err != nil {
		log.Println(err)
		return
	}

	if err := os.Remove(randomJpgImg); err != nil {
		log.Println("Failed to remove temporary file:", err)
		return
	}

}

/* send reply */
func (m *renz) Reply(teks string) {
	_,
		err := m.sock.SendMessage(context.Background(), m.Msg.Info.Chat, &waProto.Message{
		ExtendedTextMessage: &waProto.ExtendedTextMessage{
			Text: proto.String(teks),
			ContextInfo: &waProto.ContextInfo{
				Expiration:    proto.Uint32(86400),
				StanzaId:      &m.Msg.Info.ID,
				Participant:   proto.String(m.Msg.Info.Sender.String()),
				QuotedMessage: m.Msg.Message,
			},
		},
	})
	if err != nil {
		return
	}
}

/* send replyAsSticker */
func (m *renz) ReplyAsSticker(data []byte) {
	m.SendSticker(m.Msg.Info.Chat, data, dto.ExtraSend{Reply: true})
}

/* send adReply */
func (m *renz) ReplyAd(teks string) {
	var isImage = waProto.ContextInfo_ExternalAdReplyInfo_IMAGE
	_, err := m.sock.SendMessage(context.Background(), m.Msg.Info.Chat, &waProto.Message{
		ExtendedTextMessage: &waProto.ExtendedTextMessage{
			Text: proto.String(teks),
			ContextInfo: &waProto.ContextInfo{
				ExternalAdReply: &waProto.ContextInfo_ExternalAdReplyInfo{
					Title:                 proto.String("MywaBOT 2023"),
					Body:                  proto.String("Made with waSocket by Amirul Dev"),
					MediaType:             &isImage,
					ThumbnailUrl:          proto.String("https://telegra.ph/file/eb7261ee8de82f8f48142.jpg"),
					MediaUrl:              proto.String("https://wa.me/stickerpack/amirul.dev"),
					SourceUrl:             proto.String("https://chat.whatsapp.com/ByQt0u0bz4NJfNPEUfDHps"),
					ShowAdAttribution:     proto.Bool(true),
					RenderLargerThumbnail: proto.Bool(true),
				},
				Expiration:    proto.Uint32(86400),
				StanzaId:      &m.Msg.Info.ID,
				Participant:   proto.String(m.Msg.Info.Sender.String()),
				QuotedMessage: m.Msg.Message,
			},
		},
	})
	if err != nil {
		return
	}
}

/* send contact */
func (m *renz) SendContact(jid types.JID, number string, nama string) {
	_,
		err := m.sock.SendMessage(context.Background(), jid, &waProto.Message{
		ContactMessage: &waProto.ContactMessage{
			DisplayName: proto.String(nama),
			Vcard:       proto.String(fmt.Sprintf("BEGIN:VCARD\nVERSION:3.0\nN:%s;;;\nFN:%s\nitem1.TEL;waid=%s:+%s\nitem1.X-ABLabel:Mobile\nEND:VCARD", nama, nama, number, number)),
			ContextInfo: &waProto.ContextInfo{
				StanzaId:      &m.Msg.Info.ID,
				Participant:   proto.String(m.Msg.Info.Sender.String()),
				QuotedMessage: m.Msg.Message,
			},
		},
	})
	if err != nil {
		return
	}
}

/* create channel */
func (m *renz) createChannel(params []string) {
	_,
		err := m.sock.CreateNewsletter(waSocket.CreateNewsletterParams{
		Name: strings.Join(params, " "),
	})
	if err != nil {
		return
	}
}

/* fetch group admin */
func (m *renz) FetchGroupAdmin(Jid types.JID) ([]string, error) {
	var Admin []string
	resp, err := m.sock.GetGroupInfo(Jid)
	if err != nil {
		return Admin, err
	} else {
		for _, group := range resp.Participants {
			if group.IsAdmin || group.IsSuperAdmin {
				Admin = append(Admin, group.JID.String())
			}
		}
	}
	return Admin, nil
}

/* get group admin */
func (m *renz) GetGroupAdmin(jid types.JID, sender string) bool {
	if !m.Msg.Info.IsGroup {
		return false
	}
	admin, err := m.FetchGroupAdmin(jid)
	if err != nil {
		return false
	}
	for _, v := range admin {
		if v == sender {
			return true
		}
	}
	return false
}

/* get link group */
func (m *renz) LinkGc(Jid types.JID, reset bool) string {
	link,
		err := m.sock.GetGroupInviteLink(Jid, reset)

	if err != nil {
		panic(err)
	}
	return link
}

func (m *renz) GetCMD() string {
	extended := m.Msg.Message.GetExtendedTextMessage().GetText()
	text := m.Msg.Message.GetConversation()
	imageMatch := m.Msg.Message.GetImageMessage().GetCaption()
	videoMatch := m.Msg.Message.GetVideoMessage().GetCaption()
	//pollVote := m.Msg.Message.GetPollUpdateMessage().GetVote()
	tempBtnId := m.Msg.Message.GetTemplateButtonReplyMessage().GetSelectedId()
	btnId := m.Msg.Message.GetButtonsResponseMessage().GetSelectedButtonId()
	listId := m.Msg.Message.GetListResponseMessage().GetSingleSelectReply().GetSelectedRowId()
	var command string
	if text != "" {
		command = text
	} else if imageMatch != "" {
		command = imageMatch
	} else if videoMatch != "" {
		command = videoMatch
	} else if extended != "" {
		command = extended
		/*
		   } else if pollVote != "" {
		   command = pollVote
		*/
	} else if tempBtnId != "" {
		command = tempBtnId
	} else if btnId != "" {
		command = btnId
	} else if listId != "" {
		command = listId
	}
	return command
}