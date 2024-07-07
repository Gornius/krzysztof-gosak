package components

import (
	"fmt"

	"github.com/bwmarrin/discordgo"
	"github.com/gornius/krzysztof-gosak/slashcommands"
	"github.com/gornius/krzysztof-gosak/utils"
)

var ReadyCheckAcceptButtonHandler = ComponentHandler{
	ComponentID: "ready-check-accept",
	Handler: func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		user := utils.GetUserFromInteraction(i.Interaction)
		slashcommands.ReadyCheckUpdateStatus(s, i.Message, user.ID, slashcommands.ReadyCheckStatusAccepted)
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseDeferredMessageUpdate,
			Data: &discordgo.InteractionResponseData{
				Content: "TEST",
			},
		})
	},
}

var ReadyCheckRejectButtonHandler = ComponentHandler{
	ComponentID: "ready-check-reject",
	Handler: func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		user := utils.GetUserFromInteraction(i.Interaction)
		err := slashcommands.ReadyCheckUpdateStatus(s, i.Message, user.ID, slashcommands.ReadyCheckStatusRejected)
		if err != nil {
			fmt.Println("err")
			return
		}
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseDeferredMessageUpdate,
			Data: &discordgo.InteractionResponseData{
				Content: "TEST",
			},
		})
	},
}
