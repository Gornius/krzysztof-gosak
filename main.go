package main

import (
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/gornius/krzysztof-gosak/slashcommands"
)

const (
	ReadyCheckAutoDeleteTime = 60 * time.Second
	CommandErrorDeleteTime   = 5 * time.Second
)

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

	slashcommands.RegisterSlashCommands(dg, []*slashcommands.SlashCommand{
		&slashcommands.PingAppCommand,
		&slashcommands.ReadyCheckCommand,
	})

	dg.Identify.Intents = discordgo.IntentsGuildMessages

	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-sc
}
