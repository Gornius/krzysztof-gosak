package main

import (
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/bwmarrin/discordgo"
)

func main() {
	authToken := os.Getenv("KRZYSZTOF_GOSAK_TOKEN")
	dg, err := discordgo.New("Bot " + authToken)
	if err != nil {
		panic(err)
	}

	dg.AddHandler(messageCreate)
	dg.Identify.Intents = discordgo.IntentsGuildMessages

	err = dg.Open()
	if err != nil {
		panic(err)
	}
	defer dg.Close()

	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-sc
}

func messageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	if m.Author.ID == s.State.User.ID {
		return
	}

	if strings.Contains(strings.ToLower(m.Content), "można") {
		s.ChannelMessageSendReply(m.ChannelID, "Można. Gdyby to było złe to Bóg by inaczej świat stworzył.", m.Reference())
	}
}
