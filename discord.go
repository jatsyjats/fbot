// Discordgo wrapper

package main

import "github.com/bwmarrin/discordgo"

type Discord struct {
	*discordgo.Session
}

func NewDiscord(token string) (*Discord, error) {
	session, err := discordgo.New("Bot " + token)
	if err != nil {
		return nil, WrapError(err)
	}

	discord := &Discord{
		Session: session,
	}
	return discord, nil
}
