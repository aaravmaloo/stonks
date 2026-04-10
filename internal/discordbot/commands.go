package discordbot

import "github.com/bwmarrin/discordgo"

func commandDefinitions() []*discordgo.ApplicationCommand {
	scopeChoices := []*discordgo.ApplicationCommandOptionChoice{
		{Name: "global", Value: "global"},
		{Name: "friends", Value: "friends"},
	}
	sideChoices := []*discordgo.ApplicationCommandOptionChoice{
		{Name: "buy", Value: "buy"},
		{Name: "sell", Value: "sell"},
	}
	visibilityChoices := []*discordgo.ApplicationCommandOptionChoice{
		{Name: "private", Value: "private"},
		{Name: "public", Value: "public"},
	}
	hiringChoices := []*discordgo.ApplicationCommandOptionChoice{
		{Name: "best_value", Value: "best_value"},
		{Name: "high_output", Value: "high_output"},
		{Name: "low_risk", Value: "low_risk"},
	}
	strategyChoices := []*discordgo.ApplicationCommandOptionChoice{
		{Name: "Aggressive", Value: "aggressive"},
		{Name: "Balanced", Value: "balanced"},
		{Name: "Defensive", Value: "defensive"},
	}
	upgradeChoices := []*discordgo.ApplicationCommandOptionChoice{
		{Name: "Marketing", Value: "marketing"},
		{Name: "R&D", Value: "rd"},
		{Name: "Automation", Value: "automation"},
		{Name: "Compliance", Value: "compliance"},
		{Name: "Seats", Value: "seats"},
	}
	machineChoices := []*discordgo.ApplicationCommandOptionChoice{
		{Name: "Assembly Line", Value: "assembly_line"},
		{Name: "Server Rack", Value: "server_rack"},
		{Name: "Lab Equipment", Value: "lab_equipment"},
	}
	fundChoices := []*discordgo.ApplicationCommandOptionChoice{
		{Name: "TECH6X", Value: "TECH6X"},
		{Name: "CORE20", Value: "CORE20"},
		{Name: "VOLT10", Value: "VOLT10"},
		{Name: "DIVMAX", Value: "DIVMAX"},
		{Name: "AIEDGE", Value: "AIEDGE"},
		{Name: "STABLE", Value: "STABLE"},
	}
	rushChoices := []*discordgo.ApplicationCommandOptionChoice{
		{Name: "steady", Value: "steady"},
		{Name: "surge", Value: "surge"},
		{Name: "apex", Value: "apex"},
	}

	commands := []*discordgo.ApplicationCommand{
		{Name: "setup", Description: "Show the Stanks intro and how to start playing"},
		{Name: "signup", Description: "Create your Stanks account"},
		{Name: "login", Description: "Log into your Stanks account"},
		{Name: "logout", Description: "Disconnect your Discord from Stanks"},
		{Name: "dashboard", Description: "Show your full Stanks dashboard"},
		{Name: "world", Description: "Show the current political climate, catalyst, and global markets"},
		{Name: "wallet", Description: "Show your wallet balance and stats"},
		{
			Name:        "rush",
			Description: "Play the streak-and-vault volatility loop",
			Options: []*discordgo.ApplicationCommandOption{
				{Type: discordgo.ApplicationCommandOptionString, Name: "mode", Description: "steady, surge, or apex", Choices: rushChoices},
				{Type: discordgo.ApplicationCommandOptionNumber, Name: "amount", Description: "Amount in stonky to put on the line"},
			},
		},
		{
			Name:        "transfer",
			Description: "Send stonky to another player by username",
			Options: []*discordgo.ApplicationCommandOption{
				{Type: discordgo.ApplicationCommandOptionString, Name: "username", Description: "Recipient username", Required: true},
				{Type: discordgo.ApplicationCommandOptionNumber, Name: "amount", Description: "Amount in stonky", Required: true},
			},
		},
		{Name: "portfolio", Description: "Show your stock positions and P/L"},
		{
			Name:        "stocks",
			Description: "List tradable stocks on the market",
			Options: []*discordgo.ApplicationCommandOption{
				{Type: discordgo.ApplicationCommandOptionBoolean, Name: "all", Description: "Include unlisted stocks too"},
			},
		},
		{
			Name:        "stock",
			Description: "Show details and price chart for a stock",
			Options: []*discordgo.ApplicationCommandOption{
				{Type: discordgo.ApplicationCommandOptionString, Name: "symbol", Description: "Stock symbol (6 letters)", Required: true},
			},
		},
		{
			Name:        "order",
			Description: "Place a buy or sell order",
			Options: []*discordgo.ApplicationCommandOption{
				{Type: discordgo.ApplicationCommandOptionString, Name: "symbol", Description: "Stock symbol", Required: true},
				{Type: discordgo.ApplicationCommandOptionString, Name: "side", Description: "Buy or sell", Required: true, Choices: sideChoices},
				{Type: discordgo.ApplicationCommandOptionNumber, Name: "shares", Description: "Number of shares", Required: true},
			},
		},
		{Name: "funds", Description: "List available mutual funds"},
		{
			Name:        "fund-order",
			Description: "Buy or sell mutual fund shares",
			Options: []*discordgo.ApplicationCommandOption{
				{Type: discordgo.ApplicationCommandOptionString, Name: "fund", Description: "Fund code", Required: true, Choices: fundChoices},
				{Type: discordgo.ApplicationCommandOptionString, Name: "side", Description: "Buy or sell", Required: true, Choices: sideChoices},
				{Type: discordgo.ApplicationCommandOptionInteger, Name: "units", Description: "Number of units", Required: true},
			},
		},
		{
			Name:        "business-create",
			Description: "Create a new business",
			Options: []*discordgo.ApplicationCommandOption{
				{Type: discordgo.ApplicationCommandOptionString, Name: "name", Description: "Business name", Required: true},
				{Type: discordgo.ApplicationCommandOptionString, Name: "visibility", Description: "Visibility", Choices: visibilityChoices},
			},
		},
		{
			Name:        "business",
			Description: "Show detailed business overview",
			Options: []*discordgo.ApplicationCommandOption{
				{Type: discordgo.ApplicationCommandOptionInteger, Name: "business_id", Description: "Business ID", Required: true},
			},
		},
		{
			Name:        "employees",
			Description: "List employees in a business",
			Options: []*discordgo.ApplicationCommandOption{
				{Type: discordgo.ApplicationCommandOptionInteger, Name: "business_id", Description: "Business ID", Required: true},
			},
		},
		{Name: "candidates", Description: "Show available employee candidates"},
		{
			Name:        "hire-many",
			Description: "Hire multiple employees at once",
			Options: []*discordgo.ApplicationCommandOption{
				{Type: discordgo.ApplicationCommandOptionInteger, Name: "business_id", Description: "Business ID", Required: true},
				{Type: discordgo.ApplicationCommandOptionInteger, Name: "count", Description: "How many to hire", Required: true},
				{Type: discordgo.ApplicationCommandOptionString, Name: "strategy", Description: "Hiring strategy", Choices: hiringChoices},
			},
		},
		{
			Name:        "machinery",
			Description: "List or buy machinery for a business",
			Options: []*discordgo.ApplicationCommandOption{
				{Type: discordgo.ApplicationCommandOptionInteger, Name: "business_id", Description: "Business ID", Required: true},
				{Type: discordgo.ApplicationCommandOptionString, Name: "buy", Description: "Machine type to buy", Choices: machineChoices},
			},
		},
		{
			Name:        "loans",
			Description: "Manage business loans",
			Options: []*discordgo.ApplicationCommandOption{
				{Type: discordgo.ApplicationCommandOptionInteger, Name: "business_id", Description: "Business ID", Required: true},
				{Type: discordgo.ApplicationCommandOptionString, Name: "action", Description: "Action", Choices: []*discordgo.ApplicationCommandOptionChoice{
					{Name: "List loans", Value: "list"},
					{Name: "Take loan", Value: "take"},
					{Name: "Repay loan", Value: "repay"},
				}},
				{Type: discordgo.ApplicationCommandOptionNumber, Name: "amount", Description: "Loan amount in stonky (for take/repay)"},
			},
		},
		{
			Name:        "strategy",
			Description: "Set your business strategy",
			Options: []*discordgo.ApplicationCommandOption{
				{Type: discordgo.ApplicationCommandOptionInteger, Name: "business_id", Description: "Business ID", Required: true},
				{Type: discordgo.ApplicationCommandOptionString, Name: "mode", Description: "Strategy mode", Required: true, Choices: strategyChoices},
			},
		},
		{
			Name:        "upgrades",
			Description: "Buy an upgrade for your business",
			Options: []*discordgo.ApplicationCommandOption{
				{Type: discordgo.ApplicationCommandOptionInteger, Name: "business_id", Description: "Business ID", Required: true},
				{Type: discordgo.ApplicationCommandOptionString, Name: "upgrade", Description: "Upgrade type", Required: true, Choices: upgradeChoices},
			},
		},
		{
			Name:        "reserve",
			Description: "Manage business cash reserve",
			Options: []*discordgo.ApplicationCommandOption{
				{Type: discordgo.ApplicationCommandOptionInteger, Name: "business_id", Description: "Business ID", Required: true},
				{Type: discordgo.ApplicationCommandOptionString, Name: "action", Description: "Deposit or withdraw", Required: true, Choices: []*discordgo.ApplicationCommandOptionChoice{
					{Name: "Deposit", Value: "deposit"},
					{Name: "Withdraw", Value: "withdraw"},
				}},
				{Type: discordgo.ApplicationCommandOptionNumber, Name: "amount", Description: "Amount in stonky", Required: true},
			},
		},
		{
			Name:        "ipo",
			Description: "Take your business public via IPO",
			Options: []*discordgo.ApplicationCommandOption{
				{Type: discordgo.ApplicationCommandOptionInteger, Name: "business_id", Description: "Business ID", Required: true},
				{Type: discordgo.ApplicationCommandOptionString, Name: "symbol", Description: "Stock symbol (6 uppercase letters)", Required: true},
				{Type: discordgo.ApplicationCommandOptionNumber, Name: "price", Description: "IPO price in stonky", Required: true},
			},
		},
		{
			Name:        "sell-business",
			Description: "Sell your business to the bank",
			Options: []*discordgo.ApplicationCommandOption{
				{Type: discordgo.ApplicationCommandOptionInteger, Name: "business_id", Description: "Business ID", Required: true},
			},
		},
		{Name: "stakes", Description: "Show the business stakes you own"},
		{
			Name:        "give-stake",
			Description: "Give part of your company to another player by username",
			Options: []*discordgo.ApplicationCommandOption{
				{Type: discordgo.ApplicationCommandOptionInteger, Name: "business_id", Description: "Business ID", Required: true},
				{Type: discordgo.ApplicationCommandOptionString, Name: "username", Description: "Recipient username", Required: true},
				{Type: discordgo.ApplicationCommandOptionNumber, Name: "percent", Description: "Stake percent to transfer", Required: true},
			},
		},
		{
			Name:        "revoke-stakes",
			Description: "Take back part of a player's stake in your business",
			Options: []*discordgo.ApplicationCommandOption{
				{Type: discordgo.ApplicationCommandOptionInteger, Name: "business_id", Description: "Business ID", Required: true},
				{Type: discordgo.ApplicationCommandOptionString, Name: "username", Description: "Username to revoke from", Required: true},
				{Type: discordgo.ApplicationCommandOptionNumber, Name: "percent", Description: "Stake percent to revoke", Required: true},
			},
		},
		{
			Name:        "leaderboard",
			Description: "Show the current leaderboard",
			Options: []*discordgo.ApplicationCommandOption{
				{Type: discordgo.ApplicationCommandOptionString, Name: "scope", Description: "Global or friends", Choices: scopeChoices},
			},
		},
		{
			Name:        "friends",
			Description: "Manage your friends list",
			Options: []*discordgo.ApplicationCommandOption{
				{Type: discordgo.ApplicationCommandOptionString, Name: "action", Description: "Add or remove", Required: true, Choices: []*discordgo.ApplicationCommandOptionChoice{
					{Name: "Add friend", Value: "add"},
					{Name: "Remove friend", Value: "remove"},
				}},
				{Type: discordgo.ApplicationCommandOptionString, Name: "invite_code", Description: "Friend's invite code", Required: true},
			},
		},
	}

	contexts := []discordgo.InteractionContextType{
		discordgo.InteractionContextGuild,
		discordgo.InteractionContextBotDM,
	}
	integrationTypes := []discordgo.ApplicationIntegrationType{
		discordgo.ApplicationIntegrationGuildInstall,
		discordgo.ApplicationIntegrationUserInstall,
	}
	for _, cmd := range commands {
		dmAllowed := true
		cmd.DMPermission = &dmAllowed
		cmd.Contexts = &contexts
		cmd.IntegrationTypes = &integrationTypes
	}
	return commands
}
