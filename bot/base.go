package bot

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/bwmarrin/discordgo"
)

type Whitelist struct {
	ExclusiveJoin []string `json:"ExclusiveJoin"`
}

var commands = []*discordgo.ApplicationCommand{
	{
		Type:                     discordgo.ChatApplicationCommand,
		Name:                     "whitelist",
		DefaultMemberPermissions: Ptr(int64(discordgo.PermissionAdministrator)),
		DMPermission:             new(bool),
		NSFW:                     new(bool),
		Description:              "Add a user to the whitelist",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "eos_id",
				Description: "The EOS id of the user.",
				Required:    true,
			},
		},
	},
}

func ParseSlashCommand(i *discordgo.InteractionCreate) map[string]interface{} {
	var options = make(map[string]interface{})
	for _, option := range i.ApplicationCommandData().Options {
		options[option.Name] = option.Value
	}

	return options
}

func BotConnect(token string) (*discordgo.Session, error) {

	s, err := discordgo.New("Bot " + token)
	if err != nil {
		return s, fmt.Errorf("Discordgo.New Error: %w", err)
	}

	s.Identify.Intents = discordgo.IntentsAllWithoutPrivileged | discordgo.IntentsGuildMembers
	s.AddHandler(HandleCommands)

	err = s.Open()
	if err != nil {
		return s, fmt.Errorf("failed to open a websocket connection with discord. Likely due to an invalid token. %w", err)
	}

	s.ApplicationCommandBulkOverwrite(s.State.User.ID, "", commands)

	fmt.Println(s.State.User.Username + " bot startup complete!")

	return s, nil
}

func Ptr[T any](v T) *T {
	return &v
}

func HandleCommands(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if i.Type != discordgo.InteractionApplicationCommand {
		return
	}

	switch i.ApplicationCommandData().Name {
	case "whitelist":
		WhitelistCommand(s, i)
	}
}

func WhitelistCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
	// Assuming the user ID is passed as an option to the command
	options := ParseSlashCommand(i)
	if options["eos_id"] == nil {
		fmt.Println(options)
		fmt.Println("No user ID provided")
		return
	}
	userId := options["eos_id"].(string)

	// Load the current whitelist from the file
	var whitelist Whitelist
	file, err := os.OpenFile("whitelist.json", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		fmt.Println("Error opening whitelist file:", err)
		return
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	err = decoder.Decode(&whitelist)
	if err != nil {
		fmt.Println("Error decoding whitelist file:", err)
		return
	}

	// Add the new user ID to the whitelist
	whitelist.ExclusiveJoin = append(whitelist.ExclusiveJoin, userId)

	// Save the updated whitelist back to the file
	file, err = os.Create("whitelist.json")
	if err != nil {
		fmt.Println("Error opening whitelist file for writing:", err)
		return
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	err = encoder.Encode(whitelist)
	if err != nil {
		fmt.Println("Error encoding whitelist to file:", err)
		return
	}

	// Send a confirmation message
	err = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: "User added to the whitelist.",
		},
	})
	if err != nil {
		fmt.Println("Error sending confirmation message:", err)
	}
}
