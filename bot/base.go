package bot

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/bwmarrin/discordgo"
)

type Whitelist struct {
	ExclusiveJoin []string `json:"ExclusiveJoin"`
}

type GistUpdatePayload struct {
	Files map[string]map[string]string `json:"files"`
}

type GistGetPayload struct {
	Files map[string]map[string]any `json:"files"`
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

// Function to get the current content of a Gist
func GetGistContent(gistID, token string) (*Whitelist, error) {
	client := &http.Client{}
	req, err := http.NewRequest("GET", fmt.Sprintf("https://api.github.com/gists/%s", gistID), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Add("Authorization", "token "+token)
	req.Header.Add("X-GitHub-Api-Version", "2022-11-28")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// Read the response body into a byte array
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var payload GistGetPayload
	err = json.Unmarshal(body, &payload)
	if err != nil {
		fmt.Println("Error decoding whitelist:", err)
		return nil, err
	}

	// Assuming the file containing the whitelist is named "whitelist.json"
	whitelistContent, exists := payload.Files["whitelist.json"]
	if !exists {
		return nil, fmt.Errorf("whitelist.json not found in the gist")
	}

	var whitelist Whitelist
	err = json.Unmarshal([]byte(whitelistContent["content"].(string)), &whitelist)
	if err != nil {
		fmt.Println("Error decoding whitelist:", err)
		return nil, err
	}

	return &whitelist, nil
}

func UpdateGist(gistID, token string, whitelist *Whitelist) error {
	client := &http.Client{}

	// Marshal the whitelist into JSON
	whitelistContent, err := json.Marshal(whitelist)
	if err != nil {
		return err
	}

	// Prepare the Gist update payload
	updatePayload := GistUpdatePayload{
		Files: map[string]map[string]string{
			"whitelist.json": {"content": string(whitelistContent)},
		},
	}
	payloadBytes, err := json.Marshal(updatePayload)
	if err != nil {
		return err
	}

	reqBody := bytes.NewBuffer(payloadBytes)
	req, err := http.NewRequest("PATCH", fmt.Sprintf("https://api.github.com/gists/%s", gistID), reqBody)
	if err != nil {
		return err
	}
	req.Header.Add("Authorization", "token "+token)
	req.Header.Add("Content-Type", "application/json")

	_, err = client.Do(req)
	return err
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

	gistID := "34573185a0c6f2dbda53109ba1d006c4" // Replace with your actual Gist ID
	token := os.Getenv("GITHUB_TOKEN")           // Ensure GITHUB_TOKEN is set in your environment variables

	// Fetch current whitelist from GitHub Gist
	whitelist, err := GetGistContent(gistID, token)
	if err != nil {
		fmt.Println("Error fetching whitelist from Gist:", err)
		return
	}

	// Check if the user ID is already in the whitelist
	alreadyExists := false
	for _, id := range whitelist.ExclusiveJoin {
		if id == userId {
			alreadyExists = true
			break
		}
	}

	// Add the new user ID to the whitelist if it's not already there
	if !alreadyExists {
		whitelist.ExclusiveJoin = append(whitelist.ExclusiveJoin, userId)
	}

	// Update the Gist if a new ID was added
	if !alreadyExists {
		err = UpdateGist(gistID, token, whitelist)
		if err != nil {
			fmt.Println("Error updating whitelist Gist:", err)
			return
		}
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
