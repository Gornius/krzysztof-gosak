package components

import "github.com/bwmarrin/discordgo"

type ComponentHandler struct {
	ComponentID string
	Handler     func(s *discordgo.Session, i *discordgo.InteractionCreate)
}

func RegisterComponentInteractionHandlers(s *discordgo.Session, handlers []*ComponentHandler) {
	s.AddHandler(func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		if i.Interaction.Type != discordgo.InteractionMessageComponent {
			return
		}
		for _, h := range handlers {
			if i.MessageComponentData().CustomID == h.ComponentID {
				h.Handler(s, i)
			}
		}
	})
}
