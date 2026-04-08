package discordbot

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"stanks/internal/game"

	"github.com/bwmarrin/discordgo"
	"github.com/google/uuid"
)

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

	eb := NewEmbed().Title(fmt.Sprintf("Business #%d", out.ID)).Color(colorBusiness).
		Desc(out.Name).
		Field("Employees", fmt.Sprintf("%d / %d", out.EmployeeCount, out.EmployeeLimit), true).
		Field("Revenue/Tick", fmtStonky(out.RevenuePerTickMicros), true).
		Field("Operating Costs", fmtStonky(out.OperatingCostsMicros), true).
		Field("Strategy", out.Strategy, true).
		Field("Brand", progressBar(out.BrandBps, 10000, 10), true).
		Field("Health", progressBar(out.OperationalHealthBps, 10000, 10), true).
		Field("Marketing", upgradeBar(out.MarketingLevel, 10), true).
		Field("R&D", upgradeBar(out.RDLevel, 10), true).
		Field("Automation", upgradeBar(out.AutomationLevel, 10), true).
		Field("Compliance", upgradeBar(out.ComplianceLevel, 10), true).
		Field("Cash Reserve", fmtStonky(out.CashReserveMicros), true).
		Field("Loans Outstanding", fmtStonky(out.LoanOutstandingMicros), true)

	if strings.TrimSpace(out.StockSymbol) != "" {
		eb.Field("Stock", strings.TrimSpace(out.StockSymbol), true)
	}
	if strings.TrimSpace(out.LastEvent) != "" {
		eb.Field("Last Event", out.LastEvent, false)
	}

	return b.respondEmbedWithComponents(s, i, eb.Build(), []discordgo.MessageComponent{businessActionButtons(out.ID)})
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

	return b.renderCandidatePage(s, i, out.Candidates, 0)
}

func (b *Bot) renderCandidatePage(s *discordgo.Session, i *discordgo.InteractionCreate, candidates []employeeCandidate, page int) error {
	totalPages := (len(candidates) + pageSize - 1) / pageSize
	if page < 0 {
		page = 0
	}
	if page >= totalPages {
		page = totalPages - 1
	}

	start := page * pageSize
	end := start + pageSize
	if end > len(candidates) {
		end = len(candidates)
	}

	lines := make([]string, 0, end-start)
	for _, c := range candidates[start:end] {
		lines = append(lines, fmt.Sprintf("`#%d` **%s** | %s | cost %s | rev %s | risk %.2f%%",
			c.ID, c.FullName, c.Role, fmtStonky(c.HireCostMicros), fmtStonky(c.RevenuePerTickMicros), float64(c.RiskBps)/100))
	}

	eb := NewEmbed().Title("Candidates").Color(colorBusiness).
		Desc(strings.Join(lines, "\n")).
		Field("Shown", strconv.Itoa(end-start), true).
		Field("Total", strconv.Itoa(len(candidates)), true)

	components := []discordgo.MessageComponent{paginationRow("candidates", page, totalPages)}
	return b.respondEmbedWithComponents(s, i, eb.Build(), components)
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

	return b.renderEmployeePage(s, i, out.Employees, 0, businessID)
}

func (b *Bot) renderEmployeePage(s *discordgo.Session, i *discordgo.InteractionCreate, employees []businessEmployee, page int, businessID int64) error {
	totalPages := (len(employees) + pageSize - 1) / pageSize
	if page < 0 {
		page = 0
	}
	if page >= totalPages {
		page = totalPages - 1
	}

	start := page * pageSize
	end := start + pageSize
	if end > len(employees) {
		end = len(employees)
	}

	lines := make([]string, 0, end-start)
	for _, e := range employees[start:end] {
		lines = append(lines, fmt.Sprintf("`#%d` **%s** | %s | rev %s | risk %.2f%%",
			e.ID, e.FullName, e.Role, fmtStonky(e.RevenuePerTickMicros), float64(e.RiskBps)/100))
	}

	eb := NewEmbed().Title("Employees").Color(colorBusiness).
		Desc(strings.Join(lines, "\n")).
		Field("Business ID", strconv.FormatInt(businessID, 10), true).
		Field("Total", strconv.Itoa(len(employees)), true)

	bid := strconv.FormatInt(businessID, 10)
	components := []discordgo.MessageComponent{paginationRow("employees", page, totalPages, bid)}
	return b.respondEmbedWithComponents(s, i, eb.Build(), components)
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

func (b *Bot) handleMachinery(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	token, _, err := b.requireSession(ctx, s, i)
	if err != nil {
		return err
	}
	data := i.ApplicationCommandData()
	businessID := int64(integerOption(data.Options, "business_id", 0))
	buyType := strings.TrimSpace(stringOption(data.Options, "buy", ""))

	if buyType != "" {
		raw, err := b.client.BuyBusinessMachinery(ctx, token, businessID, buyType, uuid.NewString())
		if err != nil {
			return b.respondAuthAwareError(ctx, s, i, err)
		}
		fields := fieldsFromMap(raw, []fieldMapping{
			{Key: "machine_type", Label: "Type"},
			{Key: "cost_micros", Label: "Cost", Micros: true},
			{Key: "balance_micros", Label: "New Balance", Micros: true},
		})
		return b.respondEmbed(s, i, successEmbed("Machinery Purchased", fmt.Sprintf("Bought `%s` for business `%d`.", buyType, businessID), fields))
	}

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

	eb := NewEmbed().Title("Machinery").Color(colorBusiness).
		Desc(strings.Join(lines, "\n")).
		Field("Business ID", strconv.FormatInt(businessID, 10), true).
		Field("Count", strconv.Itoa(len(items)), true)

	return b.respondEmbed(s, i, eb.Build())
}

func (b *Bot) handleLoans(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	token, _, err := b.requireSession(ctx, s, i)
	if err != nil {
		return err
	}
	data := i.ApplicationCommandData()
	businessID := int64(integerOption(data.Options, "business_id", 0))
	action := strings.TrimSpace(stringOption(data.Options, "action", "list"))
	amount := numberOption(data.Options, "amount", 0)

	switch action {
	case "take":
		amountMicros := game.StonkyToMicros(amount)
		raw, err := b.client.TakeBusinessLoan(ctx, token, businessID, amountMicros, uuid.NewString())
		if err != nil {
			return b.respondAuthAwareError(ctx, s, i, err)
		}
		fields := fieldsFromMap(raw, []fieldMapping{
			{Key: "amount_micros", Label: "Amount", Micros: true},
			{Key: "outstanding_micros", Label: "Outstanding", Micros: true},
			{Key: "balance_micros", Label: "New Balance", Micros: true},
		})
		return b.respondEmbed(s, i, successEmbed("Loan Taken", fmt.Sprintf("Took a loan of %s for business `%d`.", fmtStonky(amountMicros), businessID), fields))

	case "repay":
		amountMicros := game.StonkyToMicros(amount)
		raw, err := b.client.RepayBusinessLoan(ctx, token, businessID, amountMicros, uuid.NewString())
		if err != nil {
			return b.respondAuthAwareError(ctx, s, i, err)
		}
		fields := fieldsFromMap(raw, []fieldMapping{
			{Key: "repaid_micros", Label: "Repaid", Micros: true},
			{Key: "outstanding_micros", Label: "Outstanding", Micros: true},
			{Key: "balance_micros", Label: "New Balance", Micros: true},
		})
		return b.respondEmbed(s, i, successEmbed("Loan Repaid", fmt.Sprintf("Repaid %s for business `%d`.", fmtStonky(amountMicros), businessID), fields))

	default:
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
		eb := NewEmbed().Title("Loans").Color(colorBusiness).
			Desc(strings.Join(lines, "\n")).
			Field("Business ID", strconv.FormatInt(businessID, 10), true)
		return b.respondEmbed(s, i, eb.Build())
	}
}

func (b *Bot) handleStrategy(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	token, _, err := b.requireSession(ctx, s, i)
	if err != nil {
		return err
	}
	data := i.ApplicationCommandData()
	businessID := int64(integerOption(data.Options, "business_id", 0))
	mode := strings.TrimSpace(stringOption(data.Options, "mode", ""))

	raw, err := b.client.SetBusinessStrategy(ctx, token, businessID, mode, uuid.NewString())
	if err != nil {
		return b.respondAuthAwareError(ctx, s, i, err)
	}
	fields := fieldsFromMap(raw, []fieldMapping{
		{Key: "strategy", Label: "Strategy"},
		{Key: "previous_strategy", Label: "Previous"},
	})
	return b.respondEmbed(s, i, successEmbed("Strategy Updated", fmt.Sprintf("Business `%d` strategy set to **%s**.", businessID, mode), fields))
}

func (b *Bot) handleUpgrades(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	token, _, err := b.requireSession(ctx, s, i)
	if err != nil {
		return err
	}
	data := i.ApplicationCommandData()
	businessID := int64(integerOption(data.Options, "business_id", 0))
	upgrade := strings.TrimSpace(stringOption(data.Options, "upgrade", ""))

	raw, err := b.client.BuyBusinessUpgrade(ctx, token, businessID, upgrade, uuid.NewString())
	if err != nil {
		return b.respondAuthAwareError(ctx, s, i, err)
	}
	fields := fieldsFromMap(raw, []fieldMapping{
		{Key: "upgrade", Label: "Upgrade"},
		{Key: "new_level", Label: "New Level"},
		{Key: "cost_micros", Label: "Cost", Micros: true},
		{Key: "balance_micros", Label: "New Balance", Micros: true},
	})
	return b.respondEmbed(s, i, successEmbed("Upgrade Purchased", fmt.Sprintf("Bought **%s** upgrade for business `%d`.", upgrade, businessID), fields))
}

func (b *Bot) handleReserve(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	token, _, err := b.requireSession(ctx, s, i)
	if err != nil {
		return err
	}
	data := i.ApplicationCommandData()
	businessID := int64(integerOption(data.Options, "business_id", 0))
	action := strings.TrimSpace(stringOption(data.Options, "action", ""))
	amount := numberOption(data.Options, "amount", 0)
	amountMicros := game.StonkyToMicros(amount)

	var raw map[string]any
	if action == "deposit" {
		raw, err = b.client.BusinessReserveDeposit(ctx, token, businessID, amountMicros, uuid.NewString())
	} else {
		raw, err = b.client.BusinessReserveWithdraw(ctx, token, businessID, amountMicros, uuid.NewString())
	}
	if err != nil {
		return b.respondAuthAwareError(ctx, s, i, err)
	}
	fields := fieldsFromMap(raw, []fieldMapping{
		{Key: "reserve_micros", Label: "Reserve", Micros: true},
		{Key: "balance_micros", Label: "Balance", Micros: true},
	})
	return b.respondEmbed(s, i, successEmbed("Reserve "+strings.Title(action), fmt.Sprintf("%s %s for business `%d`.", strings.Title(action), fmtStonky(amountMicros), businessID), fields))
}

func (b *Bot) handleIPO(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	token, _, err := b.requireSession(ctx, s, i)
	if err != nil {
		return err
	}
	data := i.ApplicationCommandData()
	businessID := int64(integerOption(data.Options, "business_id", 0))
	symbol := strings.ToUpper(strings.TrimSpace(stringOption(data.Options, "symbol", "")))
	price := numberOption(data.Options, "price", 0)
	priceMicros := game.StonkyToMicros(price)

	raw, err := b.client.BusinessIPO(ctx, token, businessID, symbol, priceMicros, uuid.NewString())
	if err != nil {
		return b.respondAuthAwareError(ctx, s, i, err)
	}
	fields := fieldsFromMap(raw, []fieldMapping{
		{Key: "symbol", Label: "Symbol"},
		{Key: "price_micros", Label: "IPO Price", Micros: true},
	})
	return b.respondEmbed(s, i, successEmbed("IPO Complete", fmt.Sprintf("Business `%d` is now public as `%s` at %s.", businessID, symbol, fmtStonky(priceMicros)), fields))
}

func (b *Bot) handleSellBusiness(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	token, _, err := b.requireSession(ctx, s, i)
	if err != nil {
		return err
	}
	businessID := int64(integerOption(i.ApplicationCommandData().Options, "business_id", 0))

	raw, err := b.client.SellBusinessToBank(ctx, token, businessID, uuid.NewString())
	if err != nil {
		return b.respondAuthAwareError(ctx, s, i, err)
	}
	fields := fieldsFromMap(raw, []fieldMapping{
		{Key: "payout_micros", Label: "Payout", Micros: true},
		{Key: "balance_micros", Label: "New Balance", Micros: true},
	})
	return b.respondEmbed(s, i, successEmbed("Business Sold", fmt.Sprintf("Business `%d` has been sold to the bank.", businessID), fields))
}

func formatMaybeMicros(v any) string {
	if v == nil {
		return "-"
	}
	if micros, ok := toInt64(v); ok {
		return fmtStonky(micros)
	}
	return fmt.Sprint(v)
}
