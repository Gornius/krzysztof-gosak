package components

import (
	"fmt"

	"github.com/bwmarrin/discordgo"
	"github.com/gornius/krzysztof-gosak/slashcommands"
	"github.com/gornius/krzysztof-gosak/utils"
)

var ReadyCheckAcceptButtonHandler = ComponentHandler{
	ComponentID: "ready-check-accept",
	Handler:     getCommonReadyCheckButtonHandler(slashcommands.ReadyCheckStatusAccepted),
}

var ReadyCheckRejectButtonHandler = ComponentHandler{
	ComponentID: "ready-check-reject",
	Handler:     getCommonReadyCheckButtonHandler(slashcommands.ReadyCheckStatusRejected),
}

func getCommonReadyCheckButtonHandler(status slashcommands.ReadyCheckStatus) func(*discordgo.Session, *discordgo.InteractionCreate) {
	return func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		user := utils.GetUserFromInteraction(i.Interaction)
		err := slashcommands.ReadyCheckUpdateStatus(s, i.Message, user.ID, status)
		if err != nil {
			fmt.Println(err)
			return
		}

		newTable, err := slashcommands.ReadyCheckTableDraw(i.Message.ChannelID, i.Message.Interaction.ID)
		if err != nil {
			fmt.Println(err)
			return
		}
		i.Message.Embeds[0].Description = newTable

		newColor, err := slashcommands.ReadyCheckGetEmbedColor(i.Message.ChannelID, i.Message.Interaction.ID)
		if err != nil {
			fmt.Println(err)
			return
		}
		i.Message.Embeds[0].Color = int(newColor)

		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseUpdateMessage,
			Data: &discordgo.InteractionResponseData{
				Content:    i.Message.Content,
				Components: i.Message.Components,
				Embeds:     i.Message.Embeds,
			},
		})
	}
}
