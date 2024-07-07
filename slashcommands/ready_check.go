package slashcommands

import (
	"bytes"
	"errors"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/gornius/krzysztof-gosak/pkg/dgvoice"
	"github.com/gornius/krzysztof-gosak/utils"
	"github.com/jedib0t/go-pretty/v6/table"
)

const ReadyCheckWaitingTime = 5 * time.Second

type ReadyCheckStatus string

const (
	ReadyCheckStatusWaiting       ReadyCheckStatus = "⌛ WAITING"
	ReadyCheckStatusRejected      ReadyCheckStatus = "⛔ REJECTED"
	ReadyCheckStatusAccepted      ReadyCheckStatus = "✅ ACCEPTED"
	ReadyCheckStatusDidNotRespond ReadyCheckStatus = "⛔ DID NOT RESPOND"
)

type ReadyCheckEmbedColor int

const (
	ReadyCheckEmbedColorWaiting  ReadyCheckEmbedColor = 0xeab308
	ReadyCheckEmbedColorRejected ReadyCheckEmbedColor = 0xdc2626
	ReadyCheckEmbedColorAccepted ReadyCheckEmbedColor = 0x84cc16
)

var voiceStoppers = map[string]chan bool{}

// format: ReadyCheckRows[channelID][interactionID][userID]
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

func ReadyCheckTableDraw(channelID string, interactionID string) (string, error) {
	rows, err := findReadyCheckRows(channelID, interactionID)
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

	return "```\n" + output.String() + "\n```", nil
}

func ReadyCheckUpdateStatus(s *discordgo.Session, message *discordgo.Message, userID string, status ReadyCheckStatus) error {
	row, err := findReadyCheckRow(message.ChannelID, message.Interaction.ID, userID)
	if err != nil {
		return err
	}
	row.Status = status
	return nil
}

func ReadyCheckGetEmbedColor(channelID string, interactionID string) (ReadyCheckEmbedColor, error) {
	rows, err := findReadyCheckRows(channelID, interactionID)
	if err != nil {
		return 0, err
	}
	rowsCount := len(rows)
	readyRowsCount := 0

	for _, row := range rows {
		if row.Status == ReadyCheckStatusAccepted {
			readyRowsCount++
		}
		if row.Status == ReadyCheckStatusRejected || row.Status == ReadyCheckStatusDidNotRespond {
			return ReadyCheckEmbedColorRejected, nil
		}
	}

	if rowsCount == readyRowsCount {
		return ReadyCheckEmbedColorAccepted, nil
	}

	return ReadyCheckEmbedColorWaiting, nil
}

func readyCheckTableInit(channelID string, interactionID string, users []*discordgo.User) error {
	for _, user := range users {
		row := &ReadyCheckRow{
			UserName:      user.Username,
			ChannelID:     channelID,
			InteractionID: interactionID,
			UserID:        user.ID,
			Status:        ReadyCheckStatusWaiting,
		}
		readyCheckRows = append(readyCheckRows, row)
	}

	return nil
}

func readyCheckHandleTimeout(s *discordgo.Session, interaction *discordgo.Interaction, message *discordgo.Message) error {
	time.Sleep(ReadyCheckWaitingTime)

	rows, err := findReadyCheckRows(message.ChannelID, interaction.ID)
	if err != nil {
		return err
	}
	for _, row := range rows {
		if row.Status == ReadyCheckStatusWaiting {
			row.Status = ReadyCheckStatusDidNotRespond
		}
	}

	newEmbedColor, _ := ReadyCheckGetEmbedColor(message.ChannelID, interaction.ID)
	message.Embeds[0].Color = int(newEmbedColor)

	newTable, err := ReadyCheckTableDraw(message.ChannelID, interaction.ID)
	if err != nil {
		return err
	}
	message.Embeds[0].Description = newTable

	s.ChannelMessageEditComplex(&discordgo.MessageEdit{
		Channel:    message.ChannelID,
		ID:         message.ID,
		Embeds:     &message.Embeds,
		Components: &[]discordgo.MessageComponent{},
	})
	readyCheckCleanUpRows(message.ChannelID, interaction.ID)

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
		readyCheckTableText, err := ReadyCheckTableDraw(i.Interaction.ChannelID, i.Interaction.ID)
		if err != nil {
			utils.SendErrorMessage(s, i.Interaction, "Something went wrong")
		}

		// Print ready check message
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: user.Mention() + " has requested a ready check for users: " + strings.Join(usersInChannelMentions, ", ") + ". You have " + ReadyCheckWaitingTime.String() + " to respond.",
				Embeds: []*discordgo.MessageEmbed{
					{
						Title:       "Ready Check",
						Description: readyCheckTableText,
						Color:       int(ReadyCheckEmbedColorWaiting),
					},
				},
				Components: []discordgo.MessageComponent{
					discordgo.ActionsRow{
						Components: []discordgo.MessageComponent{
							discordgo.Button{
								CustomID: "ready-check-accept",
								Label:    "✅ ACCEPT",
								Style:    discordgo.SuccessButton,
							},
							discordgo.Button{
								CustomID: "ready-check-reject",
								Label:    "⛔ REJECT",
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

		// Timeout handle
		go readyCheckHandleTimeout(s, i.Interaction, msg)

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
