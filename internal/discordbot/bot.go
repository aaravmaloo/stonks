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
)

var ErrNoSession = errors.New("no discord session found")

const (
	signupModalID = "stanks:signup"
	loginModalID  = "stanks:login"
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

type stakesPayload struct {
	Stakes []game.StakeView `json:"stakes"`
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
	_, err := b.session.ApplicationCommandBulkOverwrite(appID, scope, b.commands)
	return err
}

func (b *Bot) onInteraction(s *discordgo.Session, i *discordgo.InteractionCreate) {
	switch i.Type {
	case discordgo.InteractionApplicationCommand:
		b.handleCommand(s, i)
	case discordgo.InteractionModalSubmit:
		b.handleModal(s, i)
	case discordgo.InteractionMessageComponent:
		b.handleComponent(s, i)
	}
}

func (b *Bot) handleCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
	data := i.ApplicationCommandData()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	var err error
	switch data.Name {
	case "setup":
		err = b.runDeferred(ctx, s, i, func() error { return b.handleSetup(ctx, s, i) })
	case "signup":
		err = b.openAuthModal(s, i, signupModalID, "Create Stanks Account", true)
	case "login":
		err = b.openAuthModal(s, i, loginModalID, "Log Into Stanks", false)
	case "logout":
		err = b.runDeferredPrivate(ctx, s, i, func() error { return b.handleLogout(ctx, s, i) })
	case "dashboard":
		err = b.runDeferred(ctx, s, i, func() error { return b.handleDashboard(ctx, s, i) })
	case "world":
		err = b.runDeferred(ctx, s, i, func() error { return b.handleWorld(ctx, s, i) })
	case "wallet":
		err = b.runDeferred(ctx, s, i, func() error { return b.handleWallet(ctx, s, i) })
	case "transfer":
		err = b.runDeferredPrivate(ctx, s, i, func() error { return b.handleTransfer(ctx, s, i) })
	case "portfolio":
		err = b.runDeferredPrivate(ctx, s, i, func() error { return b.handlePortfolio(ctx, s, i) })
	case "stocks":
		err = b.runDeferred(ctx, s, i, func() error { return b.handleStocks(ctx, s, i) })
	case "stock":
		err = b.runDeferred(ctx, s, i, func() error { return b.handleStock(ctx, s, i) })
	case "order":
		err = b.runDeferredPrivate(ctx, s, i, func() error { return b.handleOrder(ctx, s, i) })
	case "funds":
		err = b.runDeferred(ctx, s, i, func() error { return b.handleFunds(ctx, s, i) })
	case "fund-order":
		err = b.runDeferredPrivate(ctx, s, i, func() error { return b.handleFundOrder(ctx, s, i) })
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
	case "machinery":
		err = b.runDeferred(ctx, s, i, func() error { return b.handleMachinery(ctx, s, i) })
	case "loans":
		err = b.runDeferred(ctx, s, i, func() error { return b.handleLoans(ctx, s, i) })
	case "strategy":
		err = b.runDeferred(ctx, s, i, func() error { return b.handleStrategy(ctx, s, i) })
	case "upgrades":
		err = b.runDeferred(ctx, s, i, func() error { return b.handleUpgrades(ctx, s, i) })
	case "reserve":
		err = b.runDeferred(ctx, s, i, func() error { return b.handleReserve(ctx, s, i) })
	case "ipo":
		err = b.runDeferred(ctx, s, i, func() error { return b.handleIPO(ctx, s, i) })
	case "sell-business":
		err = b.runDeferred(ctx, s, i, func() error { return b.handleSellBusiness(ctx, s, i) })
	case "stakes":
		err = b.runDeferred(ctx, s, i, func() error { return b.handleStakes(ctx, s, i) })
	case "give-stake":
		err = b.runDeferred(ctx, s, i, func() error { return b.handleGiveStake(ctx, s, i) })
	case "revoke-stakes":
		err = b.runDeferred(ctx, s, i, func() error { return b.handleRevokeStake(ctx, s, i) })
	case "leaderboard":
		err = b.runDeferred(ctx, s, i, func() error { return b.handleLeaderboard(ctx, s, i) })
	case "friends":
		err = b.runDeferredPrivate(ctx, s, i, func() error { return b.handleFriends(ctx, s, i) })
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
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := b.beginDeferredResponse(s, i, true); err != nil {
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

func (b *Bot) handleComponent(s *discordgo.Session, i *discordgo.InteractionCreate) {
	data := i.MessageComponentData()
	parts := parseCustomID(data.CustomID)
	if len(parts) == 0 {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := b.beginDeferredResponse(s, i, false); err != nil {
		b.log.Error("discord component defer failed", "custom_id", data.CustomID, "err", err)
		return
	}

	var err error
	switch parts[0] {
	case "setup":
		err = b.handleSetupButton(ctx, s, i, parts)
	case "nav":
		err = b.handleNavButton(ctx, s, i, parts)
	case "refresh":
		err = b.handleRefreshButton(ctx, s, i, parts)
	case "refresh_stock":
		err = b.handleRefreshStockButton(ctx, s, i, parts)
	case "refresh_biz":
		err = b.handleRefreshBizButton(ctx, s, i, parts)
	case "quickbuy":
		if len(parts) >= 2 {
			err = b.respondEmbed(s, i, infoEmbed("Quick Buy", fmt.Sprintf("Use `/order symbol:%s side:buy shares:<amount>` to buy.", parts[1]), nil))
		}
	case "quicksell":
		if len(parts) >= 2 {
			err = b.respondEmbed(s, i, infoEmbed("Quick Sell", fmt.Sprintf("Use `/order symbol:%s side:sell shares:<amount>` to sell.", parts[1]), nil))
		}
	case "biz_employees":
		err = b.handleBizEmployeesButton(ctx, s, i, parts)
	case "biz_machinery":
		err = b.handleBizMachineryButton(ctx, s, i, parts)
	case "biz_loans":
		err = b.handleBizLoansButton(ctx, s, i, parts)
	case "page":
		err = b.handlePageButton(ctx, s, i, parts)
	default:
		err = b.respondEmbed(s, i, errorEmbed("Unknown action."))
	}
	if err != nil {
		b.log.Error("discord component failed", "custom_id", data.CustomID, "err", err)
		_ = b.respondEmbed(s, i, errorEmbed("That action failed."))
	}
}

func (b *Bot) handleNavButton(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate, parts []string) error {
	if len(parts) < 2 {
		return nil
	}
	switch parts[1] {
	case "portfolio":
		return b.handlePortfolio(ctx, s, i)
	case "stocks":
		token, _, err := b.requireSession(ctx, s, i)
		if err != nil {
			return err
		}
		raw, err := b.client.ListStocks(ctx, token, false)
		if err != nil {
			return b.respondAuthAwareError(ctx, s, i, err)
		}
		out, err := decodeInto[stocksPayload](raw)
		if err != nil {
			return err
		}
		return b.renderStockPage(s, i, out.Stocks, 0, false)
	case "funds":
		return b.handleFunds(ctx, s, i)
	case "world":
		return b.handleWorld(ctx, s, i)
	case "stakes":
		return b.handleStakes(ctx, s, i)
	}
	return nil
}

func (b *Bot) handleRefreshButton(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate, parts []string) error {
	if len(parts) < 2 {
		return nil
	}
	if parts[1] == "dashboard" {
		return b.handleDashboard(ctx, s, i)
	}
	return nil
}

func (b *Bot) handleRefreshStockButton(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate, parts []string) error {
	if len(parts) < 2 {
		return nil
	}
	symbol := parts[1]
	raw, err := b.client.StockDetail(ctx, "", symbol)
	if err != nil {
		return b.respondError(s, i, trimAPIError(err))
	}
	out, err := decodeInto[game.StockDetail](raw)
	if err != nil {
		return err
	}

	desc := fmt.Sprintf("**%s**\nCurrent price: %s", out.DisplayName, fmtStonky(out.CurrentPriceMicros))
	if len(out.Series) > 0 {
		prices := make([]int64, len(out.Series))
		for idx, p := range out.Series {
			prices[idx] = p.PriceMicros
		}
		desc += "\n\nPrice trend: `" + sparkline(prices) + "`"

		recent := out.Series
		if len(recent) > 5 {
			recent = recent[len(recent)-5:]
		}
		points := make([]string, 0, len(recent))
		for _, point := range recent {
			points = append(points, fmt.Sprintf("%s %s", point.TickAt.UTC().Format("Jan 02 15:04"), fmtStonky(point.PriceMicros)))
		}
		desc += "\n\n" + strings.Join(points, "\n")
	}

	eb := NewEmbed().Title("Stock | "+strings.TrimSpace(out.Symbol)).Color(colorMarket).
		Desc(desc).
		Field("Symbol", strings.TrimSpace(out.Symbol), true).
		Field("Listed", ternary(out.ListedPublic, "yes", "no"), true)

	return b.respondEmbedWithComponents(s, i, eb.Build(), []discordgo.MessageComponent{stockActionButtons(strings.TrimSpace(out.Symbol))})
}

func (b *Bot) handleRefreshBizButton(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate, parts []string) error {
	if len(parts) < 2 {
		return nil
	}
	token, _, err := b.requireSession(ctx, s, i)
	if err != nil {
		return err
	}
	businessID, _ := strconv.ParseInt(parts[1], 10, 64)
	raw, err := b.client.BusinessState(ctx, token, businessID)
	if err != nil {
		return b.respondAuthAwareError(ctx, s, i, err)
	}
	out, err := decodeInto[game.BusinessView](raw)
	if err != nil {
		return err
	}

	eb := NewEmbed().Title(fmt.Sprintf("Business #%d", out.ID)).Color(colorBusiness).
		Desc(out.Name).
		Field("Employees", fmt.Sprintf("%d / %d", out.EmployeeCount, out.EmployeeLimit), true).
		Field("Revenue/Tick", fmtStonky(out.RevenuePerTickMicros), true).
		Field("Operating Costs", fmtStonky(out.OperatingCostsMicros), true).
		Field("Salary/Tick", fmtStonky(out.EmployeeSalaryMicros), true).
		Field("Maint/Tick", fmtStonky(out.MaintenanceMicros), true).
		Field("Strategy", out.Strategy, true).
		Field("Brand", progressBar(out.BrandBps, 10000, 10), true).
		Field("Health", progressBar(out.OperationalHealthBps, 10000, 10), true)

	return b.respondEmbedWithComponents(s, i, eb.Build(), []discordgo.MessageComponent{businessActionButtons(out.ID)})
}

func (b *Bot) handleBizEmployeesButton(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate, parts []string) error {
	if len(parts) < 2 {
		return nil
	}
	token, _, err := b.requireSession(ctx, s, i)
	if err != nil {
		return err
	}
	businessID, _ := strconv.ParseInt(parts[1], 10, 64)
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
	return b.renderEmployeePage(s, i, out.Employees, 0, businessID)
}

func (b *Bot) handleBizMachineryButton(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate, parts []string) error {
	if len(parts) < 2 {
		return nil
	}
	token, _, err := b.requireSession(ctx, s, i)
	if err != nil {
		return err
	}
	businessID, _ := strconv.ParseInt(parts[1], 10, 64)
	raw, err := b.client.ListBusinessMachinery(ctx, token, businessID)
	if err != nil {
		return b.respondAuthAwareError(ctx, s, i, err)
	}
	machines, ok := raw["machines"]
	if !ok {
		return b.respondEmbed(s, i, infoEmbed("Machinery", fmt.Sprintf("Business `%d` has no machinery.", businessID), nil))
	}
	items, ok := machines.([]any)
	if !ok || len(items) == 0 {
		return b.respondEmbed(s, i, infoEmbed("Machinery", fmt.Sprintf("Business `%d` has no machinery.", businessID), nil))
	}
	lines := make([]string, 0, len(items))
	for _, item := range items {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		lines = append(lines, fmt.Sprintf("`%s` | output %s | upkeep %s",
			fmt.Sprint(m["machine_type"]),
			formatMaybeMicros(m["output_micros"]),
			formatMaybeMicros(m["upkeep_micros"]),
		))
	}
	return b.respondEmbed(s, i, infoEmbed("Machinery", strings.Join(lines, "\n"), []*discordgo.MessageEmbedField{
		{Name: "Business ID", Value: strconv.FormatInt(businessID, 10), Inline: true},
	}))
}

func (b *Bot) handleBizLoansButton(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate, parts []string) error {
	if len(parts) < 2 {
		return nil
	}
	token, _, err := b.requireSession(ctx, s, i)
	if err != nil {
		return err
	}
	businessID, _ := strconv.ParseInt(parts[1], 10, 64)
	raw, err := b.client.ListBusinessLoans(ctx, token, businessID)
	if err != nil {
		return b.respondAuthAwareError(ctx, s, i, err)
	}
	loans, ok := raw["loans"]
	if !ok {
		return b.respondEmbed(s, i, infoEmbed("Loans", fmt.Sprintf("Business `%d` has no outstanding loans.", businessID), nil))
	}
	items, ok := loans.([]any)
	if !ok || len(items) == 0 {
		return b.respondEmbed(s, i, infoEmbed("Loans", fmt.Sprintf("Business `%d` has no outstanding loans.", businessID), nil))
	}
	lines := make([]string, 0, len(items))
	for _, item := range items {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		lines = append(lines, fmt.Sprintf("Outstanding: %s | Due: %s",
			formatMaybeMicros(m["outstanding_micros"]),
			formatMaybeMicros(m["due_amount_micros"]),
		))
	}
	return b.respondEmbed(s, i, infoEmbed("Loans", strings.Join(lines, "\n"), []*discordgo.MessageEmbedField{
		{Name: "Business ID", Value: strconv.FormatInt(businessID, 10), Inline: true},
	}))
}

func (b *Bot) handlePageButton(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate, parts []string) error {
	if len(parts) < 3 {
		return nil
	}
	namespace := parts[1]
	page, _ := strconv.Atoi(parts[2])

	switch namespace {
	case "stocks":
		token, _, err := b.requireSession(ctx, s, i)
		if err != nil {
			return err
		}
		all := false
		if len(parts) >= 4 {
			all = parts[3] == "true"
		}
		raw, err := b.client.ListStocks(ctx, token, all)
		if err != nil {
			return b.respondAuthAwareError(ctx, s, i, err)
		}
		out, err := decodeInto[stocksPayload](raw)
		if err != nil {
			return err
		}
		return b.renderStockPage(s, i, out.Stocks, page, all)
	case "candidates":
		raw, err := b.client.ListEmployeeCandidates(ctx, "")
		if err != nil {
			return b.respondError(s, i, trimAPIError(err))
		}
		out, err := decodeInto[candidatesPayload](raw)
		if err != nil {
			return err
		}
		return b.renderCandidatePage(s, i, out.Candidates, page)
	case "employees":
		if len(parts) < 4 {
			return nil
		}
		token, _, err := b.requireSession(ctx, s, i)
		if err != nil {
			return err
		}
		businessID, _ := strconv.ParseInt(parts[3], 10, 64)
		raw, err := b.client.ListBusinessEmployees(ctx, token, businessID)
		if err != nil {
			return b.respondAuthAwareError(ctx, s, i, err)
		}
		out, err := decodeInto[businessEmployeesPayload](raw)
		if err != nil {
			return err
		}
		return b.renderEmployeePage(s, i, out.Employees, page, businessID)
	}
	return nil
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
	if err := b.beginDeferredResponse(s, i, false); err != nil {
		return err
	}
	return fn()
}

func (b *Bot) runDeferredPrivate(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate, fn func() error) error {
	if err := b.beginDeferredResponse(s, i, true); err != nil {
		return err
	}
	return fn()
}

func (b *Bot) beginDeferredResponse(s *discordgo.Session, i *discordgo.InteractionCreate, ephemeral bool) error {
	if i.Type == discordgo.InteractionMessageComponent {
		return s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseDeferredMessageUpdate,
		})
	}
	data := &discordgo.InteractionResponseData{}
	if ephemeral {
		data.Flags = discordgo.MessageFlagsEphemeral
	}
	return s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
		Data: data,
	})
}

func (b *Bot) respondEmbed(s *discordgo.Session, i *discordgo.InteractionCreate, embed *discordgo.MessageEmbed) error {
	embeds := []*discordgo.MessageEmbed{embed}
	_, err := s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
		Embeds: &embeds,
	})
	return err
}

func (b *Bot) respondEmbedWithComponents(s *discordgo.Session, i *discordgo.InteractionCreate, embed *discordgo.MessageEmbed, components []discordgo.MessageComponent) error {
	embeds := []*discordgo.MessageEmbed{embed}
	_, err := s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
		Embeds:     &embeds,
		Components: &components,
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
	return fmtStonky(v)
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
