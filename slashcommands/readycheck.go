package slashcommands

import (
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/gornius/krzysztof-gosak/pkg/dgvoice"
	"github.com/gornius/krzysztof-gosak/utils"
)

const ReadyCheckAutoDeleteTime = 1 * time.Minute

var voiceStoppers map[string]chan bool = map[string]chan bool{}

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

		// Print ready check message
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: user.Mention() + " has requested a ready check for users: " + strings.Join(usersInChannelMentions, ", ") + ". This message will auto-delete in " + ReadyCheckAutoDeleteTime.String() + ".",
			},
		})

		// Add interactions to ready check message
		msg, err := s.InteractionResponse(i.Interaction)
		if err != nil {
			return
		}
		s.MessageReactionAdd(msg.ChannelID, msg.ID, "✅")
		s.MessageReactionAdd(msg.ChannelID, msg.ID, "⛔")

		// Auto-destroy ready check message
		go func() {
			time.Sleep(ReadyCheckAutoDeleteTime)
			s.ChannelMessageDelete(msg.ChannelID, msg.ID)
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
