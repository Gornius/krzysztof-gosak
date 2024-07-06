package slashcommands

import "github.com/bwmarrin/discordgo"

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
