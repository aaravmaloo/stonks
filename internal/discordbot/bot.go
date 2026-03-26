package discordbot

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"stanks/internal/cli"
	"stanks/internal/config"
	"stanks/internal/game"

	"github.com/bwmarrin/discordgo"
	"github.com/google/uuid"
)

var ErrNoSession = errors.New("no discord session found")

const (
	signupModalID = "stanks:signup"
	loginModalID  = "stanks:login"

	colorPrimary = 0x1F8B4C
	colorDanger  = 0xD83C3E
	colorInfo    = 0x2B6CB0
)

type Bot struct {
	log      *slog.Logger
	session  *discordgo.Session
	client   *cli.Client
	store    *Store
	guildID  string
	commands []*discordgo.ApplicationCommand
}

type stocksPayload struct {
	Stocks []game.StockView `json:"stocks"`
}

type leaderboardPayload struct {
	Rows []game.LeaderboardRow `json:"rows"`
}

type idPayload struct {
	ID int64 `json:"id"`
}

type businessEmployeesPayload struct {
	Employees []businessEmployee `json:"employees"`
}

type candidatesPayload struct {
	Candidates []employeeCandidate `json:"candidates"`
}

type businessEmployee struct {
	ID                   int64     `json:"id"`
	FullName             string    `json:"full_name"`
	Role                 string    `json:"role"`
	Trait                string    `json:"trait"`
	RevenuePerTickMicros int64     `json:"revenue_per_tick_micros"`
	RiskBps              int32     `json:"risk_bps"`
	CreatedAt            time.Time `json:"created_at"`
}

type employeeCandidate struct {
	ID                   int64  `json:"id"`
	FullName             string `json:"full_name"`
	Role                 string `json:"role"`
	Trait                string `json:"trait"`
	HireCostMicros       int64  `json:"hire_cost_micros"`
	RevenuePerTickMicros int64  `json:"revenue_per_tick_micros"`
	RiskBps              int32  `json:"risk_bps"`
}

func New(cfg config.DiscordBotConfig, logger *slog.Logger, store *Store) (*Bot, error) {
	if logger == nil {
		logger = slog.Default()
	}
	session, err := discordgo.New("Bot " + cfg.BotToken)
	if err != nil {
		return nil, err
	}
	session.Identify.Intents = discordgo.IntentsGuilds | discordgo.IntentsDirectMessages

	b := &Bot{
		log:      logger,
		session:  session,
		client:   cli.NewClient(cfg.APIBaseURL),
		store:    store,
		guildID:  cfg.GuildID,
		commands: commandDefinitions(),
	}
	session.AddHandler(b.onInteraction)
	return b, nil
}

func (b *Bot) Run(ctx context.Context) error {
	if err := b.session.Open(); err != nil {
		return err
	}
	defer b.session.Close()

	if err := b.syncCommands(); err != nil {
		return err
	}
	_ = b.session.UpdateGameStatus(0, "Stanks | /dashboard")

	b.log.Info("discord bot connected", "guild_id", b.guildID)
	<-ctx.Done()
	return nil
}

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
	commands := []*discordgo.ApplicationCommand{
		{Name: "signup", Description: "Create your Stanks account"},
		{Name: "login", Description: "Log into your Stanks account"},
		{Name: "logout", Description: "Disconnect your Discord account from Stanks"},
		{Name: "dashboard", Description: "Show your Stanks dashboard"},
		{Name: "wallet", Description: "Show your wallet summary"},
		{
			Name:        "stocks",
			Description: "List tradable stocks",
			Options: []*discordgo.ApplicationCommandOption{
				{Type: discordgo.ApplicationCommandOptionBoolean, Name: "all", Description: "Include unlisted stocks"},
			},
		},
		{
			Name:        "stock",
			Description: "Show details for a stock symbol",
			Options: []*discordgo.ApplicationCommandOption{
				{Type: discordgo.ApplicationCommandOptionString, Name: "symbol", Description: "Stock symbol", Required: true},
			},
		},
		{
			Name:        "order",
			Description: "Place a buy or sell order",
			Options: []*discordgo.ApplicationCommandOption{
				{Type: discordgo.ApplicationCommandOptionString, Name: "symbol", Description: "Stock symbol", Required: true},
				{Type: discordgo.ApplicationCommandOptionString, Name: "side", Description: "Buy or sell", Required: true, Choices: sideChoices},
				{Type: discordgo.ApplicationCommandOptionNumber, Name: "shares", Description: "Share quantity", Required: true},
			},
		},
		{
			Name:        "business-create",
			Description: "Create a new business",
			Options: []*discordgo.ApplicationCommandOption{
				{Type: discordgo.ApplicationCommandOptionString, Name: "name", Description: "Business name", Required: true},
				{Type: discordgo.ApplicationCommandOptionString, Name: "visibility", Description: "Business visibility", Choices: visibilityChoices},
			},
		},
		{
			Name:        "business",
			Description: "Show one business",
			Options: []*discordgo.ApplicationCommandOption{
				{Type: discordgo.ApplicationCommandOptionInteger, Name: "business_id", Description: "Business ID", Required: true},
			},
		},
		{Name: "candidates", Description: "Show employee candidates"},
		{
			Name:        "employees",
			Description: "List employees in a business",
			Options: []*discordgo.ApplicationCommandOption{
				{Type: discordgo.ApplicationCommandOptionInteger, Name: "business_id", Description: "Business ID", Required: true},
			},
		},
		{
			Name:        "hire-many",
			Description: "Hire multiple employees for a business",
			Options: []*discordgo.ApplicationCommandOption{
				{Type: discordgo.ApplicationCommandOptionInteger, Name: "business_id", Description: "Business ID", Required: true},
				{Type: discordgo.ApplicationCommandOptionInteger, Name: "count", Description: "How many employees to hire", Required: true},
				{Type: discordgo.ApplicationCommandOptionString, Name: "strategy", Description: "Hiring strategy", Choices: hiringChoices},
			},
		},
		{
			Name:        "leaderboard",
			Description: "Show the current leaderboard",
			Options: []*discordgo.ApplicationCommandOption{
				{Type: discordgo.ApplicationCommandOptionString, Name: "scope", Description: "Global or friends", Choices: scopeChoices},
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

func (b *Bot) syncCommands() error {
	appID := b.session.State.User.ID
	if err := b.syncCommandsForScope(appID, ""); err != nil {
		return err
	}
	if strings.TrimSpace(b.guildID) == "" {
		return nil
	}
	return b.syncCommandsForScope(appID, b.guildID)
}

func (b *Bot) syncCommandsForScope(appID, scope string) error {
	existing, err := b.session.ApplicationCommands(appID, scope)
	if err != nil {
		return err
	}
	want := make(map[string]struct{}, len(b.commands))
	for _, cmd := range b.commands {
		want[cmd.Name] = struct{}{}
	}
	for _, cmd := range existing {
		if _, ok := want[cmd.Name]; ok {
			if err := b.session.ApplicationCommandDelete(appID, scope, cmd.ID); err != nil {
				return err
			}
		}
	}
	for _, cmd := range b.commands {
		if _, err := b.session.ApplicationCommandCreate(appID, scope, cmd); err != nil {
			return err
		}
	}
	return nil
}

func (b *Bot) onInteraction(s *discordgo.Session, i *discordgo.InteractionCreate) {
	switch i.Type {
	case discordgo.InteractionApplicationCommand:
		b.handleCommand(s, i)
	case discordgo.InteractionModalSubmit:
		b.handleModal(s, i)
	}
}

func (b *Bot) handleCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
	data := i.ApplicationCommandData()
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	var err error
	switch data.Name {
	case "signup":
		err = b.openAuthModal(s, i, signupModalID, "Create Stanks Account", true)
	case "login":
		err = b.openAuthModal(s, i, loginModalID, "Log Into Stanks", false)
	case "logout":
		err = b.runDeferred(ctx, s, i, func() error { return b.handleLogout(ctx, s, i) })
	case "dashboard":
		err = b.runDeferred(ctx, s, i, func() error { return b.handleDashboard(ctx, s, i) })
	case "wallet":
		err = b.runDeferred(ctx, s, i, func() error { return b.handleWallet(ctx, s, i) })
	case "stocks":
		err = b.runDeferred(ctx, s, i, func() error { return b.handleStocks(ctx, s, i) })
	case "stock":
		err = b.runDeferred(ctx, s, i, func() error { return b.handleStock(ctx, s, i) })
	case "order":
		err = b.runDeferred(ctx, s, i, func() error { return b.handleOrder(ctx, s, i) })
	case "business-create":
		err = b.runDeferred(ctx, s, i, func() error { return b.handleBusinessCreate(ctx, s, i) })
	case "business":
		err = b.runDeferred(ctx, s, i, func() error { return b.handleBusiness(ctx, s, i) })
	case "candidates":
		err = b.runDeferred(ctx, s, i, func() error { return b.handleCandidates(ctx, s, i) })
	case "employees":
		err = b.runDeferred(ctx, s, i, func() error { return b.handleEmployees(ctx, s, i) })
	case "hire-many":
		err = b.runDeferred(ctx, s, i, func() error { return b.handleHireMany(ctx, s, i) })
	case "leaderboard":
		err = b.runDeferred(ctx, s, i, func() error { return b.handleLeaderboard(ctx, s, i) })
	default:
		err = b.respondImmediateError(s, i, "Unknown command.")
	}
	if err != nil {
		b.log.Error("discord interaction failed", "command", data.Name, "err", err)
		_ = b.respondFallbackError(s, i, "That request failed. Check the bot logs if it keeps happening.")
	}
}

func (b *Bot) handleModal(s *discordgo.Session, i *discordgo.InteractionCreate) {
	data := i.ModalSubmitData()
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	if err := b.beginDeferredResponse(s, i); err != nil {
		b.log.Error("discord modal defer failed", "custom_id", data.CustomID, "err", err)
		return
	}

	var err error
	switch data.CustomID {
	case signupModalID:
		err = b.handleSignupModal(ctx, s, i, modalValues(data.Components))
	case loginModalID:
		err = b.handleLoginModal(ctx, s, i, modalValues(data.Components))
	default:
		err = b.respondEmbed(s, i, errorEmbed("Unknown modal."))
	}
	if err != nil {
		b.log.Error("discord modal failed", "custom_id", data.CustomID, "err", err)
		_ = b.respondEmbed(s, i, errorEmbed("That request failed. Check the bot logs if it keeps happening."))
	}
}

func (b *Bot) openAuthModal(s *discordgo.Session, i *discordgo.InteractionCreate, customID, title string, includeUsername bool) error {
	components := []discordgo.MessageComponent{
		discordgo.ActionsRow{Components: []discordgo.MessageComponent{
			discordgo.TextInput{CustomID: "email", Label: "Email", Style: discordgo.TextInputShort, Placeholder: "you@example.com", Required: true},
		}},
		discordgo.ActionsRow{Components: []discordgo.MessageComponent{
			discordgo.TextInput{CustomID: "password", Label: "Password", Style: discordgo.TextInputShort, Placeholder: "Strong password", Required: true},
		}},
	}
	if includeUsername {
		components = append(components, discordgo.ActionsRow{Components: []discordgo.MessageComponent{
			discordgo.TextInput{CustomID: "username", Label: "Username", Style: discordgo.TextInputShort, Placeholder: "stonkslord", Required: true},
		}})
	}
	return s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseModal,
		Data: &discordgo.InteractionResponseData{
			CustomID:   customID,
			Title:      title,
			Components: components,
		},
	})
}

func (b *Bot) handleSignupModal(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate, values map[string]string) error {
	email := strings.TrimSpace(values["email"])
	password := strings.TrimSpace(values["password"])
	username := strings.TrimSpace(values["username"])

	session, err := b.client.Signup(ctx, email, password, username)
	if err != nil {
		return b.respondError(s, i, trimAPIError(err))
	}
	userID, err := interactionUserID(i)
	if err != nil {
		return err
	}
	if err := b.store.SaveSession(ctx, userID, session.User.Email, session.AccessToken); err != nil {
		return err
	}
	return b.respondEmbed(s, i, successEmbed("Account Ready", "Your Stanks account is live and linked to this Discord user.", []*discordgo.MessageEmbedField{
		{Name: "Email", Value: session.User.Email, Inline: true},
		{Name: "Username", Value: username, Inline: true},
	}))
}

func (b *Bot) handleLoginModal(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate, values map[string]string) error {
	email := strings.TrimSpace(values["email"])
	password := strings.TrimSpace(values["password"])

	session, err := b.client.Login(ctx, email, password)
	if err != nil {
		return b.respondError(s, i, trimAPIError(err))
	}
	userID, err := interactionUserID(i)
	if err != nil {
		return err
	}
	if err := b.store.SaveSession(ctx, userID, session.User.Email, session.AccessToken); err != nil {
		return err
	}
	return b.respondEmbed(s, i, successEmbed("Logged In", "Your Discord account is now connected to Stanks.", []*discordgo.MessageEmbedField{
		{Name: "Email", Value: session.User.Email, Inline: true},
	}))
}

func (b *Bot) handleLogout(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	userID, err := interactionUserID(i)
	if err != nil {
		return err
	}
	if err := b.store.DeleteSession(ctx, userID); err != nil {
		return err
	}
	return b.respondEmbed(s, i, infoEmbed("Logged Out", "This Discord account is no longer linked to a Stanks session.", nil))
}

func (b *Bot) handleDashboard(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	token, _, err := b.requireSession(ctx, s, i)
	if err != nil {
		return err
	}
	raw, err := b.client.Dashboard(ctx, token)
	if err != nil {
		return b.respondAuthAwareError(ctx, s, i, err)
	}
	out, err := decodeInto[game.Dashboard](raw)
	if err != nil {
		return err
	}

	fields := []*discordgo.MessageEmbedField{
		{Name: "Balance", Value: formatMicros(out.BalanceMicros), Inline: true},
		{Name: "Net Worth", Value: formatMicros(out.NetWorthMicros), Inline: true},
		{Name: "Peak", Value: formatMicros(out.PeakNetWorthMicros), Inline: true},
	}
	if out.ActiveBusinessID != nil {
		fields = append(fields, &discordgo.MessageEmbedField{Name: "Active Business ID", Value: strconv.FormatInt(*out.ActiveBusinessID, 10), Inline: true})
	}

	description := "No businesses yet."
	if len(out.Businesses) > 0 {
		lines := make([]string, 0, min(len(out.Businesses), 5))
		for idx, business := range out.Businesses {
			if idx >= 5 {
				break
			}
			lines = append(lines, fmt.Sprintf("`#%d` **%s** | rev/tick %s | employees %d/%d", business.ID, business.Name, formatMicros(business.RevenuePerTickMicros), business.EmployeeCount, business.EmployeeLimit))
		}
		description = strings.Join(lines, "\n")
	}
	return b.respondEmbed(s, i, infoEmbed("Dashboard", description, fields))
}

func (b *Bot) handleWallet(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	token, email, err := b.requireSession(ctx, s, i)
	if err != nil {
		return err
	}
	raw, err := b.client.WalletSummary(ctx, token)
	if err != nil {
		return b.respondAuthAwareError(ctx, s, i, err)
	}
	out, err := decodeInto[game.WalletSummary](raw)
	if err != nil {
		return err
	}
	fields := []*discordgo.MessageEmbedField{
		{Name: "Email", Value: email, Inline: true},
		{Name: "Balance", Value: formatMicros(out.BalanceMicros), Inline: true},
		{Name: "Peak Net Worth", Value: formatMicros(out.PeakNetWorthMicros), Inline: true},
	}
	if out.ActiveBusinessID != nil {
		fields = append(fields, &discordgo.MessageEmbedField{Name: "Active Business ID", Value: strconv.FormatInt(*out.ActiveBusinessID, 10), Inline: true})
	}
	return b.respondEmbed(s, i, infoEmbed("Wallet", "Current account balance and progression.", fields))
}

func (b *Bot) handleStocks(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	all := boolOption(i.ApplicationCommandData().Options, "all")
	raw, err := b.client.ListStocks(ctx, "", all)
	if err != nil {
		return b.respondError(s, i, trimAPIError(err))
	}
	out, err := decodeInto[stocksPayload](raw)
	if err != nil {
		return err
	}
	if len(out.Stocks) == 0 {
		return b.respondEmbed(s, i, infoEmbed("Stocks", "No stocks are available right now.", nil))
	}

	lines := make([]string, 0, min(len(out.Stocks), 12))
	for idx, stock := range out.Stocks {
		if idx >= 12 {
			break
		}
		status := "private"
		if stock.ListedPublic {
			status = "public"
		}
		lines = append(lines, fmt.Sprintf("`%s` **%s** | %s | %s", strings.TrimSpace(stock.Symbol), stock.DisplayName, formatMicros(stock.CurrentPriceMicros), status))
	}
	fields := []*discordgo.MessageEmbedField{
		{Name: "Count", Value: strconv.Itoa(len(out.Stocks)), Inline: true},
		{Name: "Mode", Value: ternary(all, "all", "public only"), Inline: true},
	}
	return b.respondEmbed(s, i, infoEmbed("Stock List", strings.Join(lines, "\n"), fields))
}

func (b *Bot) handleStock(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	symbol := strings.ToUpper(strings.TrimSpace(stringOption(i.ApplicationCommandData().Options, "symbol", "")))
	raw, err := b.client.StockDetail(ctx, "", symbol)
	if err != nil {
		return b.respondError(s, i, trimAPIError(err))
	}
	out, err := decodeInto[game.StockDetail](raw)
	if err != nil {
		return err
	}
	description := fmt.Sprintf("**%s**\nCurrent price: %s", out.DisplayName, formatMicros(out.CurrentPriceMicros))
	if len(out.Series) > 0 {
		points := make([]string, 0, min(len(out.Series), 5))
		start := max(0, len(out.Series)-5)
		for _, point := range out.Series[start:] {
			points = append(points, fmt.Sprintf("%s %s", point.TickAt.UTC().Format("Jan 02 15:04"), formatMicros(point.PriceMicros)))
		}
		description += "\n\nRecent ticks:\n" + strings.Join(points, "\n")
	}
	return b.respondEmbed(s, i, infoEmbed("Stock Detail", description, []*discordgo.MessageEmbedField{
		{Name: "Symbol", Value: strings.TrimSpace(out.Symbol), Inline: true},
		{Name: "Listed", Value: ternary(out.ListedPublic, "yes", "no"), Inline: true},
	}))
}

func (b *Bot) handleOrder(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	token, _, err := b.requireSession(ctx, s, i)
	if err != nil {
		return err
	}
	data := i.ApplicationCommandData()
	symbol := strings.ToUpper(strings.TrimSpace(stringOption(data.Options, "symbol", "")))
	side := strings.ToLower(strings.TrimSpace(stringOption(data.Options, "side", "")))
	shares := numberOption(data.Options, "shares", 0)
	units, err := game.SharesToUnits(shares)
	if err != nil {
		return b.respondError(s, i, err.Error())
	}
	raw, err := b.client.PlaceOrder(ctx, token, symbol, side, uuid.NewString(), units)
	if err != nil {
		return b.respondAuthAwareError(ctx, s, i, err)
	}
	out, err := decodeInto[game.OrderResult](raw)
	if err != nil {
		return err
	}
	return b.respondEmbed(s, i, successEmbed("Order Filled", fmt.Sprintf("%s %.4f shares of `%s`.", strings.Title(side), shares, symbol), []*discordgo.MessageEmbedField{
		{Name: "Order ID", Value: strconv.FormatInt(out.OrderID, 10), Inline: true},
		{Name: "Price", Value: formatMicros(out.PriceMicros), Inline: true},
		{Name: "Notional", Value: formatMicros(out.NotionalMicros), Inline: true},
		{Name: "Fee", Value: formatMicros(out.FeeMicros), Inline: true},
		{Name: "New Balance", Value: formatMicros(out.BalanceMicros), Inline: true},
	}))
}

func (b *Bot) handleBusinessCreate(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	token, _, err := b.requireSession(ctx, s, i)
	if err != nil {
		return err
	}
	data := i.ApplicationCommandData()
	name := strings.TrimSpace(stringOption(data.Options, "name", ""))
	visibility := strings.TrimSpace(stringOption(data.Options, "visibility", "private"))
	raw, err := b.client.CreateBusiness(ctx, token, name, visibility, uuid.NewString())
	if err != nil {
		return b.respondAuthAwareError(ctx, s, i, err)
	}
	out, err := decodeInto[idPayload](raw)
	if err != nil {
		return err
	}
	return b.respondEmbed(s, i, successEmbed("Business Created", fmt.Sprintf("**%s** is live.", name), []*discordgo.MessageEmbedField{
		{Name: "Business ID", Value: strconv.FormatInt(out.ID, 10), Inline: true},
		{Name: "Visibility", Value: visibility, Inline: true},
	}))
}

func (b *Bot) handleBusiness(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	token, _, err := b.requireSession(ctx, s, i)
	if err != nil {
		return err
	}
	businessID := int64(integerOption(i.ApplicationCommandData().Options, "business_id", 0))
	raw, err := b.client.BusinessState(ctx, token, businessID)
	if err != nil {
		return b.respondAuthAwareError(ctx, s, i, err)
	}
	out, err := decodeInto[game.BusinessView](raw)
	if err != nil {
		return err
	}
	fields := []*discordgo.MessageEmbedField{
		{Name: "Employees", Value: fmt.Sprintf("%d / %d", out.EmployeeCount, out.EmployeeLimit), Inline: true},
		{Name: "Revenue/Tick", Value: formatMicros(out.RevenuePerTickMicros), Inline: true},
		{Name: "Operating Costs", Value: formatMicros(out.OperatingCostsMicros), Inline: true},
		{Name: "Strategy", Value: out.Strategy, Inline: true},
		{Name: "Brand", Value: fmt.Sprintf("%.2f%%", float64(out.BrandBps)/100), Inline: true},
		{Name: "Operational Health", Value: fmt.Sprintf("%.2f%%", float64(out.OperationalHealthBps)/100), Inline: true},
	}
	if strings.TrimSpace(out.StockSymbol) != "" {
		fields = append(fields, &discordgo.MessageEmbedField{Name: "Stock", Value: strings.TrimSpace(out.StockSymbol), Inline: true})
	}
	if strings.TrimSpace(out.LastEvent) != "" {
		fields = append(fields, &discordgo.MessageEmbedField{Name: "Last Event", Value: out.LastEvent, Inline: false})
	}
	return b.respondEmbed(s, i, infoEmbed(fmt.Sprintf("Business #%d", out.ID), out.Name, fields))
}

func (b *Bot) handleCandidates(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	raw, err := b.client.ListEmployeeCandidates(ctx, "")
	if err != nil {
		return b.respondError(s, i, trimAPIError(err))
	}
	out, err := decodeInto[candidatesPayload](raw)
	if err != nil {
		return err
	}
	if len(out.Candidates) == 0 {
		return b.respondEmbed(s, i, infoEmbed("Candidates", "No employee candidates are available right now.", nil))
	}

	lines := make([]string, 0, min(len(out.Candidates), 10))
	for idx, candidate := range out.Candidates {
		if idx >= 10 {
			break
		}
		lines = append(lines, fmt.Sprintf("`#%d` **%s** | %s | cost %s | rev %s | risk %.2f%%", candidate.ID, candidate.FullName, candidate.Role, formatMicros(candidate.HireCostMicros), formatMicros(candidate.RevenuePerTickMicros), float64(candidate.RiskBps)/100))
	}
	return b.respondEmbed(s, i, infoEmbed("Candidates", strings.Join(lines, "\n"), []*discordgo.MessageEmbedField{
		{Name: "Shown", Value: strconv.Itoa(min(len(out.Candidates), 10)), Inline: true},
		{Name: "Total", Value: strconv.Itoa(len(out.Candidates)), Inline: true},
	}))
}

func (b *Bot) handleEmployees(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	token, _, err := b.requireSession(ctx, s, i)
	if err != nil {
		return err
	}
	businessID := int64(integerOption(i.ApplicationCommandData().Options, "business_id", 0))
	raw, err := b.client.ListBusinessEmployees(ctx, token, businessID)
	if err != nil {
		return b.respondAuthAwareError(ctx, s, i, err)
	}
	out, err := decodeInto[businessEmployeesPayload](raw)
	if err != nil {
		return err
	}
	if len(out.Employees) == 0 {
		return b.respondEmbed(s, i, infoEmbed("Employees", fmt.Sprintf("Business `%d` has no employees yet.", businessID), nil))
	}

	lines := make([]string, 0, min(len(out.Employees), 10))
	for idx, employee := range out.Employees {
		if idx >= 10 {
			break
		}
		lines = append(lines, fmt.Sprintf("`#%d` **%s** | %s | rev %s | risk %.2f%%", employee.ID, employee.FullName, employee.Role, formatMicros(employee.RevenuePerTickMicros), float64(employee.RiskBps)/100))
	}
	return b.respondEmbed(s, i, infoEmbed("Employees", strings.Join(lines, "\n"), []*discordgo.MessageEmbedField{
		{Name: "Business ID", Value: strconv.FormatInt(businessID, 10), Inline: true},
		{Name: "Total", Value: strconv.Itoa(len(out.Employees)), Inline: true},
	}))
}

func (b *Bot) handleHireMany(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	token, _, err := b.requireSession(ctx, s, i)
	if err != nil {
		return err
	}
	data := i.ApplicationCommandData()
	businessID := int64(integerOption(data.Options, "business_id", 0))
	count := integerOption(data.Options, "count", 0)
	strategy := strings.TrimSpace(stringOption(data.Options, "strategy", "best_value"))

	raw, err := b.client.HireEmployeesBulk(ctx, token, businessID, int(count), strategy, uuid.NewString())
	if err != nil {
		return b.respondAuthAwareError(ctx, s, i, err)
	}
	description := fmt.Sprintf("Bulk hiring finished for business `%d`.", businessID)
	fields := fieldsFromMap(raw, []fieldMapping{
		{Key: "strategy", Label: "Strategy"},
		{Key: "requested_count", Label: "Requested"},
		{Key: "hired_count", Label: "Hired"},
		{Key: "employee_count", Label: "Employee Count"},
		{Key: "employee_slots_remaining", Label: "Slots Remaining"},
		{Key: "total_cost_micros", Label: "Total Cost", Micros: true},
		{Key: "new_balance_micros", Label: "New Balance", Micros: true},
	})
	if preview := stringSliceFromMap(raw, "hired_name_preview"); len(preview) > 0 {
		description += "\n\nPreview:\n" + strings.Join(preview, "\n")
	}
	if preview := stringSliceFromMap(raw, "hired_names"); len(preview) > 0 {
		description += "\n\nHired:\n" + strings.Join(preview, "\n")
	}
	return b.respondEmbed(s, i, successEmbed("Hiring Complete", description, fields))
}

func (b *Bot) handleLeaderboard(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	scope := strings.TrimSpace(stringOption(i.ApplicationCommandData().Options, "scope", "global"))

	var raw map[string]any
	var err error
	if scope == "friends" {
		token, _, tokenErr := b.requireSession(ctx, s, i)
		if tokenErr != nil {
			return tokenErr
		}
		raw, err = b.client.LeaderboardFriends(ctx, token)
		if err != nil {
			return b.respondAuthAwareError(ctx, s, i, err)
		}
	} else {
		raw, err = b.client.LeaderboardGlobal(ctx, "")
		if err != nil {
			return b.respondError(s, i, trimAPIError(err))
		}
	}

	out, err := decodeInto[leaderboardPayload](raw)
	if err != nil {
		return err
	}
	if len(out.Rows) == 0 {
		return b.respondEmbed(s, i, infoEmbed("Leaderboard", "No leaderboard data yet.", nil))
	}
	lines := make([]string, 0, min(len(out.Rows), 10))
	for idx, row := range out.Rows {
		if idx >= 10 {
			break
		}
		lines = append(lines, fmt.Sprintf("`#%d` **%s** | %s", row.Rank, row.Username, formatMicros(row.NetWorthMicros)))
	}
	return b.respondEmbed(s, i, infoEmbed(strings.Title(scope)+" Leaderboard", strings.Join(lines, "\n"), nil))
}

func (b *Bot) requireSession(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) (string, string, error) {
	userID, err := interactionUserID(i)
	if err != nil {
		return "", "", err
	}
	record, err := b.store.GetSession(ctx, userID)
	if err == nil {
		return record.AccessToken, record.Email, nil
	}
	if errors.Is(err, ErrNoSession) {
		return "", "", b.respondError(s, i, "You need to `/signup` or `/login` first.")
	}
	return "", "", err
}

func (b *Bot) respondAuthAwareError(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate, err error) error {
	if isUnauthorizedAPIError(err) {
		if userID, userErr := interactionUserID(i); userErr == nil {
			_ = b.store.DeleteSession(ctx, userID)
		}
		return b.respondError(s, i, "Your Stanks session expired. Run `/login` again.")
	}
	return b.respondError(s, i, trimAPIError(err))
}

func (b *Bot) runDeferred(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate, fn func() error) error {
	if err := b.beginDeferredResponse(s, i); err != nil {
		return err
	}
	return fn()
}

func (b *Bot) beginDeferredResponse(s *discordgo.Session, i *discordgo.InteractionCreate) error {
	return s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Flags: discordgo.MessageFlagsEphemeral,
		},
	})
}

func (b *Bot) respondEmbed(s *discordgo.Session, i *discordgo.InteractionCreate, embed *discordgo.MessageEmbed) error {
	embeds := []*discordgo.MessageEmbed{embed}
	_, err := s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
		Embeds: &embeds,
	})
	return err
}

func (b *Bot) respondError(s *discordgo.Session, i *discordgo.InteractionCreate, message string) error {
	return b.respondEmbed(s, i, errorEmbed(message))
}

func (b *Bot) respondImmediateError(s *discordgo.Session, i *discordgo.InteractionCreate, message string) error {
	return s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Flags:  discordgo.MessageFlagsEphemeral,
			Embeds: []*discordgo.MessageEmbed{errorEmbed(message)},
		},
	})
}

func (b *Bot) respondFallbackError(s *discordgo.Session, i *discordgo.InteractionCreate, message string) error {
	if err := b.respondEmbed(s, i, errorEmbed(message)); err == nil {
		return nil
	}
	return b.respondImmediateError(s, i, message)
}

func successEmbed(title, description string, fields []*discordgo.MessageEmbedField) *discordgo.MessageEmbed {
	return baseEmbed(title, description, colorPrimary, fields)
}

func errorEmbed(message string) *discordgo.MessageEmbed {
	return baseEmbed("Request Failed", message, colorDanger, nil)
}

func infoEmbed(title, description string, fields []*discordgo.MessageEmbedField) *discordgo.MessageEmbed {
	return baseEmbed(title, description, colorInfo, fields)
}

func baseEmbed(title, description string, color int, fields []*discordgo.MessageEmbedField) *discordgo.MessageEmbed {
	return &discordgo.MessageEmbed{
		Title:       title,
		Description: description,
		Color:       color,
		Fields:      fields,
		Footer:      &discordgo.MessageEmbedFooter{Text: "Stanks"},
		Timestamp:   time.Now().UTC().Format(time.RFC3339),
	}
}

func modalValues(components []discordgo.MessageComponent) map[string]string {
	values := map[string]string{}
	for _, component := range components {
		switch row := component.(type) {
		case discordgo.ActionsRow:
			readActionRowValues(values, row.Components)
		case *discordgo.ActionsRow:
			readActionRowValues(values, row.Components)
		}
	}
	return values
}

func readActionRowValues(out map[string]string, components []discordgo.MessageComponent) {
	for _, child := range components {
		switch input := child.(type) {
		case discordgo.TextInput:
			out[input.CustomID] = input.Value
		case *discordgo.TextInput:
			out[input.CustomID] = input.Value
		}
	}
}

func decodeInto[T any](raw map[string]any) (T, error) {
	var out T
	buf, err := json.Marshal(raw)
	if err != nil {
		return out, err
	}
	err = json.Unmarshal(buf, &out)
	return out, err
}

type fieldMapping struct {
	Key    string
	Label  string
	Micros bool
}

func fieldsFromMap(raw map[string]any, mappings []fieldMapping) []*discordgo.MessageEmbedField {
	fields := make([]*discordgo.MessageEmbedField, 0, len(mappings))
	for _, mapping := range mappings {
		value, ok := raw[mapping.Key]
		if !ok {
			continue
		}
		text := fmt.Sprint(value)
		if mapping.Micros {
			if micros, ok := toInt64(value); ok {
				text = formatMicros(micros)
			}
		}
		fields = append(fields, &discordgo.MessageEmbedField{Name: mapping.Label, Value: text, Inline: true})
	}
	return fields
}

func stringSliceFromMap(raw map[string]any, key string) []string {
	value, ok := raw[key]
	if !ok {
		return nil
	}
	items, ok := value.([]any)
	if !ok {
		return nil
	}
	out := make([]string, 0, len(items))
	for _, item := range items {
		text := strings.TrimSpace(fmt.Sprint(item))
		if text != "" {
			out = append(out, text)
		}
	}
	return out
}

func trimAPIError(err error) string {
	if err == nil {
		return ""
	}
	msg := strings.TrimSpace(err.Error())
	if !strings.Contains(msg, "api status") {
		return msg
	}
	idx := strings.Index(msg, ":")
	if idx < 0 || idx == len(msg)-1 {
		return msg
	}
	body := strings.TrimSpace(msg[idx+1:])
	var payload struct {
		Error string `json:"error"`
	}
	if json.Unmarshal([]byte(body), &payload) == nil && strings.TrimSpace(payload.Error) != "" {
		return payload.Error
	}
	return body
}

func isUnauthorizedAPIError(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "api status 401") || strings.Contains(msg, "invalid token")
}

func formatMicros(v int64) string {
	return fmt.Sprintf("%.2f stonky", game.MicrosToStonky(v))
}

func stringOption(options []*discordgo.ApplicationCommandInteractionDataOption, name, fallback string) string {
	for _, option := range options {
		if option.Name == name {
			return option.StringValue()
		}
	}
	return fallback
}

func boolOption(options []*discordgo.ApplicationCommandInteractionDataOption, name string) bool {
	for _, option := range options {
		if option.Name == name {
			return option.BoolValue()
		}
	}
	return false
}

func integerOption(options []*discordgo.ApplicationCommandInteractionDataOption, name string, fallback int64) int64 {
	for _, option := range options {
		if option.Name == name {
			return option.IntValue()
		}
	}
	return fallback
}

func numberOption(options []*discordgo.ApplicationCommandInteractionDataOption, name string, fallback float64) float64 {
	for _, option := range options {
		if option.Name == name {
			return option.FloatValue()
		}
	}
	return fallback
}

func interactionUserID(i *discordgo.InteractionCreate) (string, error) {
	if i == nil {
		return "", fmt.Errorf("missing interaction")
	}
	if i.Member != nil && i.Member.User != nil && strings.TrimSpace(i.Member.User.ID) != "" {
		return i.Member.User.ID, nil
	}
	if i.User != nil && strings.TrimSpace(i.User.ID) != "" {
		return i.User.ID, nil
	}
	return "", fmt.Errorf("missing interaction user")
}

func toInt64(v any) (int64, bool) {
	switch n := v.(type) {
	case int64:
		return n, true
	case int:
		return int64(n), true
	case float64:
		return int64(n), true
	case json.Number:
		out, err := n.Int64()
		return out, err == nil
	case string:
		out, err := strconv.ParseInt(strings.TrimSpace(n), 10, 64)
		return out, err == nil
	default:
		return 0, false
	}
}

func ternary[T any](cond bool, yes, no T) T {
	if cond {
		return yes
	}
	return no
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
