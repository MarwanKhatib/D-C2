package main

import "github.com/bwmarrin/discordgo"

// getOrCreateCategory ensures a category with the given name exists and returns its ID.
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

// getOrCreateDeviceChannel ensures a text channel exists under the given category for this device.
func getOrCreateDeviceChannel(s *discordgo.Session, categoryID, channelName string) (string, error) {
	channels, err := s.GuildChannels(guildID)
	if err != nil {
		return "", err
	}
	for _, ch := range channels {
		if ch.Type == discordgo.ChannelTypeGuildText && ch.ParentID == categoryID && ch.Name == channelName {
			return ch.ID, nil
		}
	}
	ch, err := s.GuildChannelCreateComplex(guildID, discordgo.GuildChannelCreateData{
		Name:     channelName,
		Type:     discordgo.ChannelTypeGuildText,
		ParentID: categoryID,
	})
	if err != nil {
		return "", err
	}
	return ch.ID, nil
}
