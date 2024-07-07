package slashcommands

import (
	"bytes"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/gornius/krzysztof-gosak/pkg/dgvoice"
	"github.com/gornius/krzysztof-gosak/utils"
	"github.com/jedib0t/go-pretty/v6/table"
)

const ReadyCheckAutoDeleteTime = 60 * time.Second

type ReadyCheckStatus string

const (
	ReadyCheckStatusWaiting  ReadyCheckStatus = "⌛ WAITING"
	ReadyCheckStatusRejected ReadyCheckStatus = "⛔ REJECTED"
	ReadyCheckStatusAccepted ReadyCheckStatus = "✅ ACCEPTED"
)

var voiceStoppers = map[string]chan bool{}

// format: ReadyCheckRows[channelId][interactionId][userId]
type ReadyCheckRow struct {
	ChannelID     string
	InteractionID string
	UserID        string
	UserName      string
	Status        ReadyCheckStatus
}

var readyCheckRows = []*ReadyCheckRow{}

func findReadyCheckRows(channelID string, interactionID string) ([]*ReadyCheckRow, error) {
	rows := []*ReadyCheckRow{}
	for _, row := range readyCheckRows {
		if row.ChannelID == channelID && row.InteractionID == interactionID {
			rows = append(rows, row)
		}
	}

	if len(rows) == 0 {
		return nil, errors.New("no ready check rows found")
	}

	return rows, nil
}

func findReadyCheckRow(channelID string, interactionID string, userID string) (*ReadyCheckRow, error) {
	for _, row := range readyCheckRows {
		if row.ChannelID == channelID && row.InteractionID == interactionID && row.UserID == userID {
			return row, nil
		}
	}

	return nil, errors.New("couldn't find ready check row")
}

func readyCheckTableDraw(channelId string, interactionId string) (string, error) {
	rows, err := findReadyCheckRows(channelId, interactionId)
	if err != nil {
		return "", err
	}

	output := bytes.NewBuffer([]byte{})
	t := table.NewWriter()
	t.SetOutputMirror(output)
	t.AppendHeader(table.Row{"User", "Status"})

	for _, row := range rows {
		t.AppendRow(table.Row{row.UserName, row.Status})
	}
	t.Render()

	return output.String(), nil
}

func ReadyCheckUpdateStatus(s *discordgo.Session, message *discordgo.Message, userId string, status ReadyCheckStatus) error {
	row, err := findReadyCheckRow(message.ChannelID, message.Interaction.ID, userId)
	if err != nil {
		fmt.Printf("err: %v\n", err)
		return err
	}
	row.Status = status

	previousMessageContentPreTable := strings.Split(message.Content, "\n")[0]

	tableString, err := readyCheckTableDraw(message.ChannelID, message.Interaction.ID)
	if err != nil {
		fmt.Printf("err: %v\n", err)
		return err
	}

	_, err = s.ChannelMessageEdit(message.ChannelID, message.ID, previousMessageContentPreTable+"\n```\n"+tableString+"\n```")
	if err != nil {
		fmt.Printf("err: %v\n", err)
		return err
	}

	return nil
}

func readyCheckTableInit(channelId string, interactionId string, users []*discordgo.User) error {
	for _, user := range users {
		row := &ReadyCheckRow{
			UserName:      user.Username,
			ChannelID:     channelId,
			InteractionID: interactionId,
			UserID:        user.ID,
			Status:        ReadyCheckStatusWaiting,
		}
		readyCheckRows = append(readyCheckRows, row)
	}

	return nil
}

func readyCheckCleanUpRows(channelID string, interactionID string) {
	newRows := []*ReadyCheckRow{}
	for _, row := range readyCheckRows {
		if row.ChannelID != channelID && row.InteractionID != interactionID {
			newRows = append(newRows, row)
		}
	}

	readyCheckRows = newRows
}

var ReadyCheckCommand = SlashCommand{
	ApplicationCommand: &discordgo.ApplicationCommand{
		Name:        "readycheck",
		Description: "Ping everyone on the voice channel you're in and prepare emojis to click on it",
	},
	Handler: func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		user := utils.GetUserFromInteraction(i.Interaction)
		if user == nil {
			return
		}

		guild, err := s.State.Guild(i.GuildID)
		if err != nil {
			return
		}

		channel, err := utils.GetVoiceChannelThatUserIsIn(s, user, guild)
		if err != nil {
			utils.SendErrorMessage(s, i.Interaction, "Couldn't find a channel that you're in at the moment.")
			return
		}

		// Get users in channel that user requesting ready check is in
		usersInChannel, err := utils.GetUsersInVoiceChannel(s, channel, guild)
		if err != nil {
			utils.SendErrorMessage(s, i.Interaction, "Couldn't find a channel that you're in at the moment.")
			return
		}
		if len(usersInChannel) < 1 {
			utils.SendErrorMessage(s, i.Interaction, "There are no people in the channel")
			return
		}

		var usersInChannelMentions []string
		for _, u := range usersInChannel {
			usersInChannelMentions = append(usersInChannelMentions, u.Mention())
		}

		readyCheckTableInit(i.Interaction.ChannelID, i.Interaction.ID, usersInChannel)
		readyCheckTableText, err := readyCheckTableDraw(i.Interaction.ChannelID, i.Interaction.ID)
		if err != nil {
			fmt.Println("TODO: Handle error")
		}

		// Print ready check message
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: user.Mention() + " has requested a ready check for users: " + strings.Join(usersInChannelMentions, ", ") + ". This message will auto-delete in " + ReadyCheckAutoDeleteTime.String() + ".\n```\n" + readyCheckTableText + "\n```",
				Components: []discordgo.MessageComponent{
					discordgo.ActionsRow{
						Components: []discordgo.MessageComponent{
							discordgo.Button{
								CustomID: "ready-check-accept",
								Label:    "ACCEPT",
								Style:    discordgo.PrimaryButton,
							},
							discordgo.Button{
								CustomID: "ready-check-reject",
								Label:    "REJECT",
								Style:    discordgo.DangerButton,
							},
						},
					},
				},
			},
		})

		msg, err := s.InteractionResponse(i.Interaction)
		if err != nil {
			return
		}

		// Auto-destroy ready check message
		go func() {
			time.Sleep(ReadyCheckAutoDeleteTime)
			s.ChannelMessageDelete(msg.ChannelID, msg.ID)
			readyCheckCleanUpRows(i.Interaction.ChannelID, i.Interaction.ID)
		}()

		// Play sound
		if _, ok := voiceStoppers[i.GuildID]; ok {
			voiceStoppers[i.GuildID] <- true
			delete(voiceStoppers, i.GuildID)
			time.Sleep(400 * time.Millisecond)
		}

		stopper := make(chan bool)
		voiceStoppers[i.GuildID] = stopper
		vc, err := s.ChannelVoiceJoin(i.GuildID, channel.ID, false, false)
		if err != nil {
			return
		}
		defer func() {
			if stopper == voiceStoppers[i.GuildID] {
				vc.Disconnect()
			}
		}()
		dgvoice.PlayAudioFile(vc, "levelup2.ogg", voiceStoppers[i.GuildID])
	},
}
