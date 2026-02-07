package main

import "github.com/bwmarrin/discordgo"

func getOrCreateCategory(s *discordgo.Session, name string) (string, error) {
	channels, err := s.GuildChannels(guildID)
	if err != nil {
		return "", err
	}
	for _, ch := range channels {
		if ch.Type == discordgo.ChannelTypeGuildCategory && ch.Name == name {
			return ch.ID, nil
		}
	}
	cat, err := s.GuildChannelCreateComplex(guildID, discordgo.GuildChannelCreateData{
		Name: name,
		Type: discordgo.ChannelTypeGuildCategory,
	})
	if err != nil {
		return "", err
	}
	return cat.ID, nil
}

