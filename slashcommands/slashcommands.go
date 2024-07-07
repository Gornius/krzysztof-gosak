package slashcommands

import "github.com/bwmarrin/discordgo"

type SlashCommand struct {
	ApplicationCommand *discordgo.ApplicationCommand
	Handler            func(s *discordgo.Session, i *discordgo.InteractionCreate)
}

func RegisterSlashCommands(dg *discordgo.Session, slashCommands []*SlashCommand) {
	for _, command := range slashCommands {
		dg.ApplicationCommandCreate(dg.State.User.ID, "", command.ApplicationCommand)
	}
	dg.AddHandler(func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		if i.Interaction.Type != discordgo.InteractionApplicationCommand {
			return
		}
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
