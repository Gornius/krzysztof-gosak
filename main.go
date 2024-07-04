package main

import (
	"errors"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/bwmarrin/dgvoice"
	"github.com/bwmarrin/discordgo"
)

const (
	ReadyCheckAutoDeleteTime = 60 * time.Second
	CommandErrorDeleteTime   = 5 * time.Second
)

var voiceStoppers map[string]chan bool = map[string]chan bool{}

func main() {
	authToken := os.Getenv("KRZYSZTOF_GOSAK_TOKEN")
	dg, err := discordgo.New("Bot " + authToken)
	if err != nil {
		panic(err)
	}

	err = dg.Open()
	if err != nil {
		panic(err)
	}
	defer dg.Close()

	dg.AddHandler(func(s *discordgo.Session, m *discordgo.MessageCreate) {
		if m.Author.ID == s.State.User.ID {
			return
		}

		if strings.Contains(strings.ToLower(m.Content), "można") {
			s.ChannelMessageSendReply(m.ChannelID, "Można. Gdyby to było złe to Bóg by inaczej świat stworzył.", m.Reference())
		}
	})

	registerSlashCommands(dg, []*SlashCommand{
		&PingAppCommand,
		&ReadyCheckCommand,
	})

	dg.Identify.Intents = discordgo.IntentsGuildMessages

	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-sc
}

func registerSlashCommands(dg *discordgo.Session, slashCommands []*SlashCommand) {
	for _, command := range slashCommands {
		dg.ApplicationCommandCreate(dg.State.User.ID, "", command.ApplicationCommand)
	}
	dg.AddHandler(func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		commandName := i.ApplicationCommandData().Name
		var foundCommand *SlashCommand
		for _, command := range slashCommands {
			if command.ApplicationCommand.Name == commandName {
				foundCommand = command
				break
			}
		}
		if foundCommand == nil {
			return
		}
		foundCommand.Handler(s, i)
	})
}

type SlashCommand struct {
	ApplicationCommand *discordgo.ApplicationCommand
	Handler            func(s *discordgo.Session, i *discordgo.InteractionCreate)
}

var PingAppCommand = SlashCommand{
	ApplicationCommand: &discordgo.ApplicationCommand{
		Name:        "ping",
		Description: "Pong",
	},
	Handler: func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "Pong",
			},
		})
	},
}

var ReadyCheckCommand = SlashCommand{
	ApplicationCommand: &discordgo.ApplicationCommand{
		Name:        "readycheck",
		Description: "Ping everyone on the voice channel you're in and prepare emojis to click on it",
	},
	Handler: func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		user := getUserFromInteractionCreate(i)
		if user == nil {
			return
		}

		guild, err := s.State.Guild(i.GuildID)
		if err != nil {
			return
		}

		channel, err := getVoiceChannelThatUserIsIn(s, user, guild)
		if err != nil {
			return
		}

		// Get users in channel that user requesting ready check is in
		usersInChannel, err := getUsersInVoiceChannel(s, channel, guild)
		if err != nil {
			sendErrorMessage(s, i, "Couldn't find a channel that you're in at the moment.")
			return
		}
		if len(usersInChannel) < 1 {
			sendErrorMessage(s, i, "There are no people in the channel")
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

func getUserFromInteractionCreate(i *discordgo.InteractionCreate) *discordgo.User {
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

func getVoiceChannelThatUserIsIn(s *discordgo.Session, u *discordgo.User, guild *discordgo.Guild) (*discordgo.Channel, error) {
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

func getUsersInVoiceChannel(s *discordgo.Session, c *discordgo.Channel, guild *discordgo.Guild) ([]*discordgo.User, error) {
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

func sendErrorMessage(s *discordgo.Session, i *discordgo.InteractionCreate, message string) {
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: "⚠️ " + message,
		},
	})
	msg, err := s.InteractionResponse(i.Interaction)
	if err != nil {
		return
	}
	go func() {
		time.Sleep(CommandErrorDeleteTime)
		s.ChannelMessageDelete(msg.ChannelID, msg.ID)
	}()
}
