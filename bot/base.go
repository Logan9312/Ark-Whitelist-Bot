package bot

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
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
				Name:        "action",
				Description: "Add or remove the user from the whitelist.",
				Required:    true,
				Choices: []*discordgo.ApplicationCommandOptionChoice{
					{
						Name:  "add",
						Value: "add",
					},
					{
						Name:  "remove",
						Value: "remove",
					},
				},
			},
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "eos_id",
				Description: "The EOS id of the user.",
				Required:    true,
			},
			{
				Type:         discordgo.ApplicationCommandOptionString,
				Name:         "folder",
				Description:  "The folder the file is in.",
				Required:     true,
				Autocomplete: true,
			},
			{
				Type:         discordgo.ApplicationCommandOptionString,
				Name:         "file",
				Description:  "The file to edit.",
				Required:     true,
				Autocomplete: true,
			},
		},
	},
}

var autoCompleteFile = map[string][]string{}

func BotConnect(token string) (*discordgo.Session, error) {

	s, err := discordgo.New("Bot " + token)
	if err != nil {
		return s, fmt.Errorf("Discordgo.New Error: %w", err)
	}

	s.Identify.Intents = discordgo.IntentsAllWithoutPrivileged
	s.AddHandler(HandleCommands)

	err = s.Open()
	if err != nil {
		return s, fmt.Errorf("failed to open a websocket connection with discord. Likely due to an invalid token. %w", err)
	}

	go updateFolders()

	_, err = s.ApplicationCommandBulkOverwrite(s.State.User.ID, "", commands)
	if err != nil {
		return s, fmt.Errorf("failed to register commands: %w", err)
	}

	return s, nil
}

func FetchRepo() (*git.Repository, string, error) {
	tmpDir, err := os.MkdirTemp("", "whitelist")
	if err != nil {
		fmt.Println("Error creating temp directory:", err)
		return nil, "", err
	}

	repo, err := git.PlainClone(tmpDir, false, &git.CloneOptions{
		URL:      os.Getenv("GITHUB_URL"),
		Progress: os.Stdout,
	})
	if err != nil {
		fmt.Println("Error cloning repository:", err)
		return repo, "", err
	}

	return repo, tmpDir, nil
}

// Function to get the whitelist for a file
func GetWhitelist(folderName, fileName string) (*Whitelist, error) {
	// Clone the repository to a temporary directory

	_, tmpDir, err := FetchRepo()
	defer os.RemoveAll(tmpDir)
	if err != nil {
		return nil, err
	}

	// Read the file content
	fileContent, err := os.ReadFile(filepath.Join(tmpDir, folderName, fileName))
	if err != nil {
		return nil, err
	}

	if len(fileContent) == 0 {
		return &Whitelist{}, nil
	}

	var whitelist Whitelist
	if err := json.Unmarshal(fileContent, &whitelist); err != nil {
		return nil, err
	}

	return &whitelist, nil

}

func UpdateRepo(folderName, fileName string, whitelist *Whitelist) error {
	// Clone the repository to a temporary directory

	repo, tmpDir, err := FetchRepo()
	defer os.RemoveAll(tmpDir)
	if err != nil {
		return err
	}

	// Marshal the whitelist into JSON
	whitelistContent, err := json.MarshalIndent(whitelist, "", "    ")
	if err != nil {
		return err
	}

	fmt.Println(filepath.Join(tmpDir, folderName, fileName))

	if err := os.WriteFile(filepath.Join(tmpDir, folderName, fileName), whitelistContent, 0644); err != nil {
		return err
	}

	// Git operations: add, commit, and push
	worktree, err := repo.Worktree()
	if err != nil {
		return err
	}

	_, err = worktree.Add(".")
	if err != nil {
		return err
	}

	_, err = worktree.Commit("Update whitelist", &git.CommitOptions{
		Author: &object.Signature{
			Name:  "Ark Whitelist Bot",
			Email: "",
			When:  time.Now(),
		},
	})
	if err != nil {
		return err
	}

	auth := &http.BasicAuth{
		Username: os.Getenv("GITHUB_USERNAME"), // Replace with your GitHub username
		Password: os.Getenv("GITHUB_TOKEN"),    // Replace with your GitHub token
	}

	err = repo.Push(&git.PushOptions{
		Auth: auth,
	})
	if err != nil {
		return err
	}

	return nil
}

func ParseSlashCommand(i *discordgo.InteractionCreate) map[string]interface{} {
	var options = make(map[string]interface{})
	for _, option := range i.ApplicationCommandData().Options {
		options[option.Name] = option.Value
	}

	return options
}

func Ptr[T any](v T) *T {
	return &v
}

func HandleCommands(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if i.Type == discordgo.InteractionApplicationCommand {
		switch i.ApplicationCommandData().Name {
		case "whitelist":
			WhitelistCommand(s, i)
		}
	} else if i.Type == discordgo.InteractionApplicationCommandAutocomplete {
		switch i.ApplicationCommandData().Name {
		case "whitelist":
			WhitelistAutoComplete(s, i)
		}
	}

}

func WhitelistCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
	})

	// Assuming the user ID is passed as an option to the command
	options := ParseSlashCommand(i)
	if options["eos_id"] == nil {
		fmt.Println(options)
		fmt.Println("No user ID provided")
		return
	}
	userId := options["eos_id"].(string)

	// Fetch current whitelist from GitHub
	whitelist, err := GetWhitelist(options["folder"].(string), options["file"].(string))
	if err != nil {
		fmt.Println("Error fetching whitelist", err)
		return
	}

	if options["action"] == "add" {
		// Check if the user ID is already in the whitelist
		for _, id := range whitelist.ExclusiveJoin {
			if id == userId {
				_, err := s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
					Embeds: &[]*discordgo.MessageEmbed{
						{
							Title:       "Error",
							Description: "User is already in the whitelist.",
							Color:       0xff0000,
						},
					},
				})
				if err != nil {
					fmt.Println("Error sending confirmation message:", err)
				}
				return
			}
		}

		// Add the new user ID to the whitelist.
		whitelist.ExclusiveJoin = append(whitelist.ExclusiveJoin, userId)

		// Update the whitelist
		err = UpdateRepo(options["folder"].(string), options["file"].(string), whitelist)
		if err != nil {
			fmt.Println("Error updating whitelist:", err)
			return
		}

		// Send a confirmation message
		_, err = s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
			Embeds: &[]*discordgo.MessageEmbed{
				{
					Title:       "Success",
					Description: "User has been added to the whitelist.",
					Color:       0x00bfff, // Deep Sky Blue
				},
			},
		})
		if err != nil {
			fmt.Println("Error sending confirmation message:", err)
		}
	} else if options["action"] == "remove" {
		// Check if the user ID is already in the whitelist
		for n, id := range whitelist.ExclusiveJoin {
			if id == userId {
				// Remove the user ID from the whitelist.
				whitelist.ExclusiveJoin = append(whitelist.ExclusiveJoin[:n], whitelist.ExclusiveJoin[n+1:]...)

				// Update the whitelist
				err = UpdateRepo(options["folder"].(string), options["file"].(string), whitelist)
				if err != nil {
					fmt.Println("Error updating whitelist:", err)
					return
				}

				// Send a confirmation message
				_, err = s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
					Embeds: &[]*discordgo.MessageEmbed{
						{
							Title:       "Success",
							Description: "User has been removed from the whitelist.",
							Color:       0x00bfff, // Deep Sky Blue
						},
					},
				})
				if err != nil {
					fmt.Println("Error sending confirmation message:", err)
				}
				return
			}
		}

		_, err := s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
			Embeds: &[]*discordgo.MessageEmbed{
				{
					Title:       "Error",
					Description: "User is not in the whitelist.",
					Color:       0xff0000,
				},
			},
		})
		if err != nil {
			fmt.Println("Error sending confirmation message:", err)
		}

	} else {
		fmt.Println("Invalid action")
		return
	}

}

func WhitelistAutoComplete(s *discordgo.Session, i *discordgo.InteractionCreate) {
	// Iterate through all options to find the focused one
	var focusedOption *discordgo.ApplicationCommandInteractionDataOption
	for _, option := range i.ApplicationCommandData().Options {
		if option.Focused {
			focusedOption = option
			break
		}
	}

	// Check if a focused option was found and process accordingly
	if focusedOption != nil {
		switch focusedOption.Name {
		case "folder":
			choices := []*discordgo.ApplicationCommandOptionChoice{}
			for folder := range autoCompleteFile {
				choices = append(choices, &discordgo.ApplicationCommandOptionChoice{
					Name:  folder,
					Value: folder,
				})
			}

			err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionApplicationCommandAutocompleteResult,
				Data: &discordgo.InteractionResponseData{
					Choices: choices,
				},
			})
			if err != nil {
				fmt.Println("Error sending autocomplete response:", err)
			}
			return

		case "file":
			options := ParseSlashCommand(i)
			if options["folder"] == nil {
				fmt.Println("No folder provided")
				err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionApplicationCommandAutocompleteResult,
					Data: &discordgo.InteractionResponseData{
						Choices: []*discordgo.ApplicationCommandOptionChoice{
							{
								Name:  "Please select a folder first.",
								Value: "",
							},
						},
					},
				})
				if err != nil {
					fmt.Println("Error sending autocomplete response:", err)
				}
				return
			}

			response := []*discordgo.ApplicationCommandOptionChoice{}
			for _, file := range autoCompleteFile[options["folder"].(string)] {
				response = append(response, &discordgo.ApplicationCommandOptionChoice{
					Name:  file,
					Value: file,
				})
			}

			err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionApplicationCommandAutocompleteResult,
				Data: &discordgo.InteractionResponseData{
					Choices: response,
				},
			})
			if err != nil {
				fmt.Println("Error sending autocomplete response:", err)
			}
		}
	} else {
		// Respond with a generic error or guidance message if no option is focused
		err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionApplicationCommandAutocompleteResult,
			Data: &discordgo.InteractionResponseData{
				Choices: []*discordgo.ApplicationCommandOptionChoice{
					{
						Name:  "Please select an option.",
						Value: "",
					},
				},
			},
		})
		if err != nil {
			fmt.Println("Error sending autocomplete response:", err)
		}
	}
}

// Function to update the autocomplete list, runs every 30 minutes
func updateFolders() {
	for {
		_, tmpDir, err := FetchRepo()
		if err != nil {
			fmt.Println("Error fetching repo:", err)
			return
		}

		defer os.RemoveAll(tmpDir)

		files, err := os.ReadDir(tmpDir)
		if err != nil {
			fmt.Println("Error reading directory:", err)
		}

		for _, folder := range files {
			if folder.Name() == ".git" || !folder.IsDir() {
				continue
			}

			dirPath := filepath.Join(tmpDir, folder.Name())
			subFiles, err := os.ReadDir(dirPath)
			if err != nil {
				fmt.Println("Error reading directory:", err)
				continue
			}

			for _, subFile := range subFiles {
				if !sliceContains(autoCompleteFile[folder.Name()], subFile.Name()) {
					autoCompleteFile[folder.Name()] = append(autoCompleteFile[folder.Name()], subFile.Name())
				}
			}

		}

		time.Sleep(30 * time.Minute)
	}
}

// Helper function to check if a slice contains a specific string.
func sliceContains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
