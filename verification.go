// Verification system module

package main

import (
	"fmt"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/enescakir/emoji"
)

type VerificationConfigFormField struct {
	Label       string
	Type        string
	Placeholder string
	MaxLength   *int
	MinLength   *int
}

type VerificationConfig struct {
	InitialRole      string
	WelcomeMessage   string
	VerifyButtonText string

	FormTitle             string
	FormFields            []VerificationConfigFormField
	FormSubmitChannel     string
	FormSubmitUserMessage string
	FormEmbedDescription  string

	ApproveButtonText string
	DenyButtonText    string
	BanButtonText     string

	ApprovedRole                string
	ApprovedAnnouncementMessage string
	ApprovedAnnouncementChannel string
	ApprovedFormChannel         string

	DenyDmMessage string
}

type VerificationModule struct {
	Discord *Discord
	Config  *VerificationConfig
}

const (
	ColorRed        int = 15548997
	ColorGreen      int = 5763719
	ColorDarkOrange int = 11027200
)

func NewVerificationModule(config *VerificationConfig) *VerificationModule {
	return &VerificationModule{
		Config: config,
	}
}

func (m *VerificationModule) Register(bot *Bot) {
	Logf("Registering module")

	m.Discord = bot.Discord
	m.Discord.AddHandler(func(_ *discordgo.Session, event *discordgo.GuildMemberAdd) {
		err := m.OnGuildMemberAdd(event)
		if err != nil {
			Logf("Error: %s", ErrorToStr(err))
		}
	})
	m.Discord.AddHandler(func(_ *discordgo.Session, event *discordgo.MessageCreate) {
		err := m.OnMessageCreate(event)
		if err != nil {
			Logf("Error: %s", ErrorToStr(err))
		}
	})
	m.Discord.AddHandler(func(_ *discordgo.Session, event *discordgo.InteractionCreate) {
		err := m.OnInteractionCreate(event)
		if err != nil {
			Logf("Error: %s", ErrorToStr(err))
		}
	})
}

func (m *VerificationModule) OnGuildMemberAdd(member *discordgo.GuildMemberAdd) error {
	err := m.Discord.GuildMemberRoleAdd(member.GuildID, member.User.ID, m.Config.InitialRole)
	if err != nil {
		return WrapError(err)
	}

	Logf("Auto assigned role %v to user %v on join", m.Config.InitialRole, member.Member.DisplayName())
	return nil
}

func (m *VerificationModule) OnMessageCreate(message *discordgo.MessageCreate) error {
	if strings.EqualFold(message.Content, "!SpawnVerifyButton") {
		messageData := &discordgo.MessageSend{
			Content: m.Config.WelcomeMessage,
			Components: []discordgo.MessageComponent{
				discordgo.ActionsRow{
					Components: []discordgo.MessageComponent{
						&discordgo.Button{
							Label:    m.Config.VerifyButtonText,
							Style:    discordgo.PrimaryButton,
							CustomID: "VerifyButton",
						},
					},
				},
			},
		}

		_, err := m.Discord.ChannelMessageSendComplex(message.ChannelID, messageData)
		if err != nil {
			return WrapError(err)
		}

		err = m.Discord.ChannelMessageDelete(message.ChannelID, message.ID)
		if err != nil {
			return WrapError(err)
		}
	}

	return nil
}

func (m *VerificationModule) OnInteractionCreate(interaction *discordgo.InteractionCreate) error {
	if interaction.Type == discordgo.InteractionMessageComponent {
		data := interaction.MessageComponentData()
		if data.CustomID == "VerifyButton" {
			err := m.SendVerifyFormModal(interaction.Interaction)
			if err != nil {
				return err
			}
		} else if strings.HasPrefix(data.CustomID, "VerificationApproveButton|") {
			err := m.VerificationApproveButtonClick(interaction.Interaction)
			if err != nil {
				return err
			}
		} else if strings.HasPrefix(data.CustomID, "VerificationDenyButton|") {
			err := m.VerificationDenyButtonClick(interaction.Interaction)
			if err != nil {
				return err
			}
		} else if strings.HasPrefix(data.CustomID, "VerificationBanButton|") {
			err := m.VerificationBanButtonClick(interaction.Interaction)
			if err != nil {
				return err
			}
		} else if strings.HasPrefix(data.CustomID, "VerificationBanConfirmYesButton|") {
			err := m.VerificationBanConfirmYesButtonClick(interaction.Interaction)
			if err != nil {
				return err
			}
		} else if strings.HasPrefix(data.CustomID, "VerificationBanConfirmNoButton|") {
			err := m.VerificationBanConfirmNoButtonClick(interaction.Interaction)
			if err != nil {
				return err
			}
		}
	}

	if interaction.Type == discordgo.InteractionModalSubmit {
		data := interaction.ModalSubmitData()
		if data.CustomID == "VerifyFormModal" {
			err := m.SubmitVerifyForm(interaction.Interaction)
			if err != nil {
				return err
			}
		} else if strings.HasPrefix(data.CustomID, "VerificationDenyModal") {
			err := m.VerificationDenyModalSubmit(interaction.Interaction)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (m *VerificationModule) SendVerifyFormModal(interaction *discordgo.Interaction) error {
	components := []discordgo.MessageComponent{}

	for _, formField := range m.Config.FormFields {
		textInput := discordgo.TextInput{
			CustomID:    formField.Label,
			Label:       formField.Label,
			Style:       discordgo.TextInputShort,
			Required:    true,
			Placeholder: formField.Placeholder,
			MaxLength:   1000,
		}

		if formField.Type == "ParagraphInput" {
			textInput.Style = discordgo.TextInputParagraph
		} else if formField.Type == "ShortInput" {
			textInput.Style = discordgo.TextInputShort
		}

		if formField.MaxLength != nil {
			textInput.MaxLength = *formField.MaxLength
		}
		if formField.MinLength != nil {
			textInput.MinLength = *formField.MinLength
		}

		row := discordgo.ActionsRow{
			Components: []discordgo.MessageComponent{
				textInput,
			},
		}
		components = append(components, row)
	}

	err := m.Discord.InteractionRespond(interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseModal,
		Data: &discordgo.InteractionResponseData{
			CustomID:   "VerifyFormModal",
			Title:      m.Config.FormTitle,
			Components: components,
		},
	})
	if err != nil {
		return WrapError(err)
	}
	return nil
}

func (m *VerificationModule) SubmitVerifyForm(interaction *discordgo.Interaction) error {
	var err error

	Logf("Verification form submitted by user %v (%v)", interaction.Member.DisplayName(), interaction.Member.User.ID)

	// Acknowledge the interaction
	err = m.Discord.InteractionRespond(interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Flags: discordgo.MessageFlagsEphemeral,
		},
	})
	if err != nil {
		return WrapError(err)
	}

	// Craft the staff room message
	embedDescription := m.Config.FormEmbedDescription
	embedDescription = strings.ReplaceAll(embedDescription, "$USER", interaction.Member.Mention())

	embed := &discordgo.MessageEmbed{
		Type: discordgo.EmbedTypeRich,
		Author: &discordgo.MessageEmbedAuthor{
			Name:    interaction.Member.DisplayName(),
			IconURL: interaction.Member.User.AvatarURL(""),
		},
		Description: embedDescription,
	}

	modalData := interaction.ModalSubmitData()
	for _, component := range modalData.Components {
		textInput := component.(*discordgo.ActionsRow).Components[0].(*discordgo.TextInput)
		// CustomID is the field question
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
			Name:   textInput.CustomID,
			Value:  textInput.Value,
			Inline: false,
		})
	}

	userCreateTime, _ := discordgo.SnowflakeTimestamp(interaction.Member.User.ID)
	embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
		Name:  emoji.PageFacingUp.String() + " User ID",
		Value: interaction.Member.User.ID,
	})
	embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
		Name:  emoji.ThreeThirty.String() + " Account created",
		Value: userCreateTime.Local().Format("2006-01-02 15:04:05"),
	})

	userID := interaction.Member.User.ID
	messageData := &discordgo.MessageSend{
		Embeds: []*discordgo.MessageEmbed{embed},
		Components: []discordgo.MessageComponent{
			discordgo.ActionsRow{
				Components: []discordgo.MessageComponent{
					&discordgo.Button{
						Label: "Approve",
						Emoji: &discordgo.ComponentEmoji{
							Name: emoji.ThumbsUp.String(),
						},
						Style:    discordgo.SuccessButton,
						CustomID: "VerificationApproveButton|" + userID,
					},
					&discordgo.Button{
						Label: "Deny",
						Emoji: &discordgo.ComponentEmoji{
							Name: emoji.ThumbsDown.String(),
						},
						Style:    discordgo.SecondaryButton,
						CustomID: "VerificationDenyButton|" + userID,
					},
					&discordgo.Button{
						Label: "Ban",
						Emoji: &discordgo.ComponentEmoji{
							Name: emoji.Hammer.String(),
						},
						Style:    discordgo.DangerButton,
						CustomID: "VerificationBanButton|" + userID,
					},
				},
			},
		},
	}
	_, err = m.Discord.ChannelMessageSendComplex(m.Config.FormSubmitChannel, messageData)
	if err != nil {
		return WrapError(err)
	}

	// Provide action feedback
	userMessage := m.Config.FormSubmitUserMessage
	userMessage = strings.ReplaceAll(userMessage, "$USER", interaction.Member.Mention())
	_, err = m.Discord.FollowupMessageCreate(interaction, true, &discordgo.WebhookParams{
		Content: userMessage,
		Flags:   discordgo.MessageFlagsEphemeral,
	})
	if err != nil {
		return WrapError(err)
	}

	return nil
}

func (m *VerificationModule) VerificationApproveButtonClick(interaction *discordgo.Interaction) error {
	var err error

	// CustomID contains the original user ID
	userID := strings.Split(interaction.MessageComponentData().CustomID, "|")[1]

	Logf("Verification of user %v approved by staff %v (%v)", userID, interaction.Member.DisplayName(), interaction.Member.User.ID)

	// Acknowledge the interaction
	err = m.Discord.InteractionRespond(interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Flags: discordgo.MessageFlagsEphemeral,
		},
	})
	if err != nil {
		return WrapError(err)
	}

	// Add new role to the user, remove old role
	err = m.Discord.GuildMemberRoleRemove(interaction.GuildID, userID, m.Config.InitialRole)
	if err != nil {
		Logf("Warning: Failed to remove initial role from user %v: %v", userID, err)
	}
	err = m.Discord.GuildMemberRoleAdd(interaction.GuildID, userID, m.Config.ApprovedRole)
	if err != nil {
		Logf("Warning: Failed to add approved role to user %v: %v", userID, err)
	}

	// Send form copy to another channel
	embeds := interaction.Message.Embeds
	embeds[0].Fields = embeds[0].Fields[:len(embeds[0].Fields)-2]
	approvedFormMessage := &discordgo.MessageSend{
		Embeds: interaction.Message.Embeds,
	}
	_, err = m.Discord.ChannelMessageSendComplex(m.Config.ApprovedFormChannel, approvedFormMessage)
	if err != nil {
		return WrapError(err)
	}

	// Send announcement message
	announcementMessage := m.Config.ApprovedAnnouncementMessage
	announcementMessage = strings.ReplaceAll(announcementMessage, "$USER", "<@"+userID+">")
	_, err = m.Discord.ChannelMessageSend(m.Config.ApprovedAnnouncementChannel, announcementMessage)
	if err != nil {
		return WrapError(err)
	}

	// Provide action feedback
	_, err = m.Discord.FollowupMessageCreate(interaction, true, &discordgo.WebhookParams{
		Content: "Verification approved",
		Flags:   discordgo.MessageFlagsEphemeral,
	})
	if err != nil {
		return WrapError(err)
	}

	// Edit the bot message
	embeds = interaction.Message.Embeds
	embeds[0].Color = ColorGreen
	embeds[0].Footer = &discordgo.MessageEmbedFooter{
		Text:    "Approved by " + fmt.Sprintf("%s (%s)", interaction.Member.DisplayName(), interaction.Member.User.ID),
		IconURL: interaction.Member.AvatarURL(""),
	}
	messageEdit := &discordgo.MessageEdit{
		ID:         interaction.Message.ID,
		Channel:    interaction.Message.ChannelID,
		Embeds:     &embeds,
		Components: &[]discordgo.MessageComponent{},
	}
	_, err = m.Discord.ChannelMessageEditComplex(messageEdit)
	if err != nil {
		return WrapError(err)
	}

	return nil
}

func (m *VerificationModule) VerificationDenyButtonClick(interaction *discordgo.Interaction) error {
	userID := strings.Split(interaction.MessageComponentData().CustomID, "|")[1]

	err := m.Discord.InteractionRespond(interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseModal,
		Data: &discordgo.InteractionResponseData{
			CustomID: "VerificationDenyModal|" + userID,
			Title:    "Deny verification",
			Components: []discordgo.MessageComponent{
				discordgo.ActionsRow{
					Components: []discordgo.MessageComponent{
						discordgo.TextInput{
							Label:    "Reason",
							Style:    discordgo.TextInputParagraph,
							Required: false,
						},
					},
				},
			},
		}})

	if err != nil {
		return WrapError(err)
	}

	return nil
}

func (m *VerificationModule) VerificationDenyModalSubmit(interaction *discordgo.Interaction) error {
	var err error

	modalData := interaction.ModalSubmitData()
	userID := strings.Split(modalData.CustomID, "|")[1]
	reasonText := modalData.Components[0].(*discordgo.ActionsRow).Components[0].(*discordgo.TextInput).Value

	Logf("Verification of user %v denied by staff %v (%v)", userID, interaction.Member.DisplayName(), interaction.Member.User.ID)

	// Acknowledge the interaction
	err = m.Discord.InteractionRespond(interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Flags: discordgo.MessageFlagsEphemeral,
		},
	})
	if err != nil {
		return WrapError(err)
	}

	// DM the denied user
	dmChannel, err := m.Discord.UserChannelCreate(userID)
	if err != nil {
		Logf("Warning: Could not DM user %v with deny reason: %v", userID, err)
	} else {
		denyMessage := m.Config.DenyDmMessage
		denyMessage = strings.ReplaceAll(denyMessage, "$USER", "<@"+userID+">")
		denyMessage = strings.ReplaceAll(denyMessage, "$STAFF", interaction.Member.User.Mention())
		denyMessage = strings.ReplaceAll(denyMessage, "$REASON", reasonText)
		_, err = m.Discord.ChannelMessageSend(dmChannel.ID, denyMessage)
		if err != nil {
			Logf("Warning: Could not DM user %v: %v", userID, err)
		}

	}

	// Provide action feedback
	_, err = m.Discord.FollowupMessageCreate(interaction, true, &discordgo.WebhookParams{
		Content: "Verification denied",
		Flags:   discordgo.MessageFlagsEphemeral,
	})
	if err != nil {
		return WrapError(err)
	}

	// Edit the bot message
	embeds := interaction.Message.Embeds
	embeds[0].Color = ColorDarkOrange
	embeds[0].Footer = &discordgo.MessageEmbedFooter{
		Text:    "Denied by " + fmt.Sprintf("%s (%s)", interaction.Member.DisplayName(), interaction.Member.User.ID),
		IconURL: interaction.Member.AvatarURL(""),
	}
	messageEdit := &discordgo.MessageEdit{
		ID:         interaction.Message.ID,
		Channel:    interaction.Message.ChannelID,
		Embeds:     &embeds,
		Components: &[]discordgo.MessageComponent{},
	}
	_, err = m.Discord.ChannelMessageEditComplex(messageEdit)
	if err != nil {
		return WrapError(err)
	}

	return nil
}

func (m *VerificationModule) VerificationBanButtonClick(interaction *discordgo.Interaction) error {
	var err error

	userID := strings.Split(interaction.MessageComponentData().CustomID, "|")[1]

	// Send confirmation message
	err = m.Discord.InteractionRespond(interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: "Are you sure you want to ban the user?",
			Components: []discordgo.MessageComponent{
				discordgo.ActionsRow{
					Components: []discordgo.MessageComponent{
						&discordgo.Button{
							Label:    "Yes, I am sure",
							Style:    discordgo.DangerButton,
							CustomID: "VerificationBanConfirmYesButton|" + userID + "|" + interaction.Message.ChannelID + "|" + interaction.Message.ID,
						},
						&discordgo.Button{
							Label:    "No, cancel",
							Style:    discordgo.SuccessButton,
							CustomID: "VerificationBanConfirmNoButton|" + userID,
						},
					},
				},
			},
			Flags: discordgo.MessageFlagsEphemeral,
		},
	})
	if err != nil {
		return WrapError(err)
	}

	return nil
}

func (m *VerificationModule) VerificationBanConfirmNoButtonClick(interaction *discordgo.Interaction) error {
	var err error

	// Acknowledge the interaction
	err = m.Discord.InteractionRespond(interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseUpdateMessage,
		Data: &discordgo.InteractionResponseData{
			Content: "Ban cancelled",
			Embeds:  []*discordgo.MessageEmbed{},
		},
	})
	if err != nil {
		Logf("Warning: %s", ErrorToStr(err))
	}

	return nil
}

func (m *VerificationModule) VerificationBanConfirmYesButtonClick(interaction *discordgo.Interaction) error {
	var err error
	args := strings.Split(interaction.MessageComponentData().CustomID, "|")
	userID := args[1]

	Logf("Verification of user %v banned by staff %v (%v)", userID, interaction.Member.DisplayName(), interaction.Member.User.ID)

	// Acknowledge the interaction
	err = m.Discord.InteractionRespond(interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredMessageUpdate,
		Data: &discordgo.InteractionResponseData{
			Flags: discordgo.MessageFlagsEphemeral,
		},
	})
	if err != nil {
		return WrapError(err)
	}

	// Ban the user
	err = m.Discord.GuildBanCreateWithReason(interaction.GuildID, userID, fmt.Sprintf("Verification ban by %v (%v)", interaction.Member.DisplayName(), interaction.Member.User.ID), 0)
	if err != nil {
		Logf("Warning: Failed to ban user %v: %v", userID, err)
	}

	// Provide action feedback
	content := "User has been banned"
	_, err = m.Discord.FollowupMessageEdit(interaction, interaction.Message.ID, &discordgo.WebhookEdit{
		Content: &content,
		Embeds:  &[]*discordgo.MessageEmbed{},
	})
	if err != nil {
		Logf("Warning: %s", ErrorToStr(err))
	}

	// Edit the bot message
	messageWithButtons, err := m.Discord.ChannelMessage(args[2], args[3])
	if err != nil {
		return WrapError(err)
	}

	embeds := messageWithButtons.Embeds
	embeds[0].Color = ColorRed
	embeds[0].Footer = &discordgo.MessageEmbedFooter{
		Text:    "Banned by " + fmt.Sprintf("%s (%s)", interaction.Member.DisplayName(), interaction.Member.User.ID),
		IconURL: interaction.Member.AvatarURL(""),
	}
	messageEdit := &discordgo.MessageEdit{
		ID:         messageWithButtons.ID,
		Channel:    messageWithButtons.ChannelID,
		Embeds:     &embeds,
		Components: &[]discordgo.MessageComponent{},
	}
	_, err = m.Discord.ChannelMessageEditComplex(messageEdit)
	if err != nil {
		return WrapError(err)
	}

	return nil
}
