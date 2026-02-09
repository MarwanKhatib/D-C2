package main

// This file contains helpers for creating and reusing guild categories and
// per-device text channels.

import (
	"log"

	"github.com/bwmarrin/discordgo"
)

// getOrCreateCategory ensures a guild category with the given name exists and
// returns its ID, creating it if necessary.
func getOrCreateCategory(s *discordgo.Session, name string) (string, error) {
	channels, err := s.GuildChannels(guildID)
	if err != nil {
		log.Printf("failed to list guild channels for category %s: %v", name, err)
	} else {
		for _, ch := range channels {
			if ch.Type == discordgo.ChannelTypeGuildCategory && ch.Name == name {
				log.Printf("using existing category %s (%s)", name, ch.ID)
				return ch.ID, nil
			}
		}
	}
	cat, err := s.GuildChannelCreateComplex(guildID, discordgo.GuildChannelCreateData{
		Name: name,
		Type: discordgo.ChannelTypeGuildCategory,
	})
	if err != nil {
		log.Printf("failed to create category %s: %v", name, err)
		return "", err
	}
	log.Printf("created category %s (%s)", name, cat.ID)
	return cat.ID, nil
}

// getOrCreateDeviceChannel ensures a text channel for this device exists under
// the given category and returns its ID.
func getOrCreateDeviceChannel(s *discordgo.Session, categoryID, channelName string) (string, error) {
	channels, err := s.GuildChannels(guildID)
	if err != nil {
		log.Printf("failed to list guild channels for device channel %s: %v", channelName, err)
	} else {
		for _, ch := range channels {
			if ch.Type == discordgo.ChannelTypeGuildText && ch.ParentID == categoryID && ch.Name == channelName {
				log.Printf("using existing device channel %s (%s)", channelName, ch.ID)
				return ch.ID, nil
			}
		}
	}
	ch, err := s.GuildChannelCreateComplex(guildID, discordgo.GuildChannelCreateData{
		Name:     channelName,
		Type:     discordgo.ChannelTypeGuildText,
		ParentID: categoryID,
	})
	if err != nil {
		log.Printf("failed to create device channel %s: %v", channelName, err)
		return "", err
	}
	log.Printf("created device channel %s (%s)", channelName, ch.ID)
	return ch.ID, nil
}
