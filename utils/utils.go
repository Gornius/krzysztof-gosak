package utils

import (
	"errors"
	"time"

	"github.com/bwmarrin/discordgo"
)

const CommandErrorDeleteTime = 5 * time.Second

func GetUserFromInteraction(i *discordgo.Interaction) *discordgo.User {
	var user *discordgo.User

	user = i.User
	if user != nil {
		return user
	}

	user = i.Member.User
	if user != nil {
		return user
	}

	return nil
}

func GetVoiceChannelThatUserIsIn(s *discordgo.Session, u *discordgo.User, guild *discordgo.Guild) (*discordgo.Channel, error) {
	var voiceState *discordgo.VoiceState
	if guild.VoiceStates == nil {
		return nil, errors.New("nobody is on any voice channel on that server")
	}
	for _, vs := range guild.VoiceStates {
		if vs.UserID == u.ID {
			voiceState = vs
		}
	}
	if voiceState == nil {
		return nil, errors.New("user is not on any channel")
	}
	channel, err := s.Channel(voiceState.ChannelID)
	if err != nil {
		return nil, err
	}

	return channel, nil
}

func GetUsersInVoiceChannel(s *discordgo.Session, c *discordgo.Channel, guild *discordgo.Guild) ([]*discordgo.User, error) {
	var users []*discordgo.User
	for _, vs := range guild.VoiceStates {
		if c.ID == vs.ChannelID {
			user, err := s.User(vs.UserID)
			if err != nil {
				continue
			}
			users = append(users, user)
		}
	}

	return users, nil
}

func SendErrorMessage(s *discordgo.Session, i *discordgo.Interaction, message string) {
	s.InteractionRespond(i, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: "⚠️ " + message,
		},
	})
	msg, err := s.InteractionResponse(i)
	if err != nil {
		return
	}
	go func() {
		time.Sleep(CommandErrorDeleteTime)
		s.ChannelMessageDelete(msg.ChannelID, msg.ID)
	}()
}
