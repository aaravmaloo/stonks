package discordbot

import (
	"context"
	"fmt"
	"math"
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

	lines := []string{
		fmt.Sprintf("Name:        %s", out.Name),
		fmt.Sprintf("Visibility:  %s", out.Visibility),
		fmt.Sprintf("Listed:      %t", out.IsListed),
		fmt.Sprintf("Your Stake:  %.2f%%", float64(out.OwnedStakeBps)/100),
		fmt.Sprintf("Region:      %s", out.PrimaryRegion),
		fmt.Sprintf("Story Arc:   %s", out.NarrativeArc),
		fmt.Sprintf("Story Focus: %s", out.NarrativeFocus),
		fmt.Sprintf("Pressure:    %.2f%%", float64(out.NarrativePressureBps)/100),
		fmt.Sprintf("Cycle:       %s (%d ticks, %.2f%%)", out.CyclePhase, out.CycleTicksRemaining, float64(out.CycleImpactBps)/100),
		fmt.Sprintf("Strategy:    %s", out.Strategy),
		fmt.Sprintf("Employees:   %d / %d", out.EmployeeCount, out.EmployeeLimit),
		fmt.Sprintf("Machinery:   %d", out.MachineryCount),
		fmt.Sprintf("Upgrades:    mkt=%d rd=%d auto=%d comp=%d", out.MarketingLevel, out.RDLevel, out.AutomationLevel, out.ComplianceLevel),
		fmt.Sprintf("Brand:       %.2f%%", float64(out.BrandBps)/100),
		fmt.Sprintf("Op Health:   %.2f%%", float64(out.OperationalHealthBps)/100),
		fmt.Sprintf("Reserve:     %s stonky", fmtMicrosExact(out.CashReserveMicros)),
		fmt.Sprintf("Revenue/tick:%s stonky", fmtMicrosExact(out.RevenuePerTickMicros)),
		fmt.Sprintf("Salary/tick: %s stonky", fmtMicrosExact(out.EmployeeSalaryMicros)),
		fmt.Sprintf("Maint/tick:  %s stonky", fmtMicrosExact(out.MaintenanceMicros)),
		fmt.Sprintf("Mach output: %s stonky", fmtMicrosExact(out.MachineryOutputMicros)),
		fmt.Sprintf("Mach upkeep: %s stonky", fmtMicrosExact(out.MachineryUpkeepMicros)),
		fmt.Sprintf("Loan debt:   %s stonky", fmtMicrosExact(out.LoanOutstandingMicros)),
	}
	if strings.TrimSpace(out.StockSymbol) != "" {
		lines = append(lines, fmt.Sprintf("Stock:       %s", strings.TrimSpace(out.StockSymbol)))
	}
	if strings.TrimSpace(out.LastEvent) != "" {
		lines = append(lines, fmt.Sprintf("Last event:  %s", out.LastEvent))
	}

	eb := NewEmbed().Title(fmt.Sprintf("Business #%d", out.ID)).Color(colorBusiness).Desc(codeBlock(lines...))
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

	lines := []string{
		fmt.Sprintf("%-4s %-18s %-10s %-12s %12s %12s %8s", "ID", "NAME", "ROLE", "TRAIT", "HIRE COST", "REV/TICK", "RISK"),
	}
	for _, c := range candidates[start:end] {
		lines = append(lines, fmt.Sprintf("%-4d %-18s %-10s %-12s %12s %12s %7.2f%%",
			c.ID, truncateText(c.FullName, 18), truncateText(c.Role, 10), truncateText(c.Trait, 12), fmtMicrosExact(c.HireCostMicros), fmtMicrosExact(c.RevenuePerTickMicros), float64(c.RiskBps)/100))
	}

	eb := NewEmbed().Title("Candidates").Color(colorBusiness).
		Desc(codeBlock(lines...)).
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

	lines := []string{
		fmt.Sprintf("%-4s %-18s %-10s %-12s %12s %8s %-16s", "ID", "NAME", "ROLE", "TRAIT", "REV/TICK", "RISK", "HIRED"),
	}
	for _, e := range employees[start:end] {
		lines = append(lines, fmt.Sprintf("%-4d %-18s %-10s %-12s %12s %7.2f%% %-16s",
			e.ID, truncateText(e.FullName, 18), truncateText(e.Role, 10), truncateText(e.Trait, 12), fmtMicrosExact(e.RevenuePerTickMicros), float64(e.RiskBps)/100, e.CreatedAt.Local().Format("2006-01-02 15:04")))
	}

	eb := NewEmbed().Title("Employees").Color(colorBusiness).
		Desc(codeBlock(lines...)).
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
	})
	if cost, ok := int64FromMapKeys(raw, "total_cost_micros"); ok {
		balance, _ := int64FromMapKeys(raw, "new_balance_micros", "balance_micros", "remaining_balance_micros")
		fields = append(fields, spendSummaryFields(cost, 0, balance)...)
	}
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
			{Key: "new_level", Label: "New Level"},
		})
		if cost, ok := int64FromMapKeys(raw, "cost_micros"); ok {
			balance, _ := int64FromMapKeys(raw, "new_balance_micros", "balance_micros")
			fields = append(fields, spendSummaryFields(cost, 0, balance)...)
		}
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

	lines := []string{
		fmt.Sprintf("%-16s %12s %12s", "TYPE", "OUTPUT", "UPKEEP"),
	}
	for _, item := range items {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		lines = append(lines, fmt.Sprintf("%-16s %12s %12s",
			fmt.Sprint(m["machine_type"]),
			formatMaybeMicros(m["output_micros"]),
			formatMaybeMicros(m["upkeep_micros"]),
		))
	}

	eb := NewEmbed().Title("Machinery").Color(colorBusiness).
		Desc(codeBlock(lines...)).
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
			{Key: "outstanding_micros", Label: "Outstanding", Micros: true},
		})
		if repaid, ok := int64FromMapKeys(raw, "repaid_micros"); ok {
			balance, _ := int64FromMapKeys(raw, "balance_micros")
			fields = append(fields, spendSummaryFields(repaid, 0, balance)...)
		}
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
		lines := []string{
			fmt.Sprintf("%-12s %12s", "OUTSTAND", "DUE"),
		}
		for _, item := range items {
			m, ok := item.(map[string]any)
			if !ok {
				continue
			}
			lines = append(lines, fmt.Sprintf("%-12s %12s",
				formatMaybeMicros(m["outstanding_micros"]),
				formatMaybeMicros(m["due_amount_micros"]),
			))
		}
		eb := NewEmbed().Title("Loans").Color(colorBusiness).
			Desc(codeBlock(lines...)).
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
	})
	if cost, ok := int64FromMapKeys(raw, "cost_micros"); ok {
		balance, _ := int64FromMapKeys(raw, "balance_micros")
		fields = append(fields, spendSummaryFields(cost, 0, balance)...)
	}
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

func (b *Bot) handleStakes(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	token, _, err := b.requireSession(ctx, s, i)
	if err != nil {
		return err
	}
	raw, err := b.client.ListStakes(ctx, token)
	if err != nil {
		return b.respondAuthAwareError(ctx, s, i, err)
	}
	out, err := decodeInto[stakesPayload](raw)
	if err != nil {
		return err
	}
	if len(out.Stakes) == 0 {
		return b.respondEmbed(s, i, infoEmbed("Stakes", "You do not own any business stakes yet.", nil))
	}
	lines := []string{
		fmt.Sprintf("%-4s %-18s %-10s %-8s %10s %10s", "ID", "BUSINESS", "CONTROL", "STAKE", "VALUE", "P/L"),
	}
	for _, stake := range out.Stakes {
		lines = append(lines, fmt.Sprintf("%-4d %-18s %-10s %7.2f%% %10s %10s",
			stake.BusinessID,
			truncateText(stake.BusinessName, 18),
			truncateText(stake.ControllerUsername, 10),
			float64(stake.StakeBps)/100.0,
			fmtMicrosExact(stake.EstimatedValueMicros),
			signedMicrosExact(stake.UnrealizedPLMicros),
		))
	}
	return b.respondEmbed(s, i, NewEmbed().Title("Stakes").Color(colorBusiness).Desc(codeBlock(lines...)).Build())
}

func (b *Bot) handleGiveStake(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	token, _, err := b.requireSession(ctx, s, i)
	if err != nil {
		return err
	}
	data := i.ApplicationCommandData()
	businessID := int64(integerOption(data.Options, "business_id", 0))
	username := strings.TrimSpace(stringOption(data.Options, "username", ""))
	percent := numberOption(data.Options, "percent", 0)
	stakeBps := int32(math.Round(percent * 100))
	raw, err := b.client.TransferBusinessStake(ctx, token, businessID, username, stakeBps, uuid.NewString())
	if err != nil {
		return b.respondAuthAwareError(ctx, s, i, err)
	}
	fields := fieldsFromMap(raw, []fieldMapping{
		{Key: "estimated_value_micros", Label: "Marked Value", Micros: true},
	})
	return b.respondEmbed(s, i, successEmbed("Stake Transferred", fmt.Sprintf("Gave %.2f%% of business `%d` to `%s`.", percent, businessID, username), fields))
}

func (b *Bot) handleRevokeStake(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	token, _, err := b.requireSession(ctx, s, i)
	if err != nil {
		return err
	}
	data := i.ApplicationCommandData()
	businessID := int64(integerOption(data.Options, "business_id", 0))
	username := strings.TrimSpace(stringOption(data.Options, "username", ""))
	percent := numberOption(data.Options, "percent", 0)
	stakeBps := int32(math.Round(percent * 100))
	raw, err := b.client.RevokeBusinessStake(ctx, token, businessID, username, stakeBps, uuid.NewString())
	if err != nil {
		return b.respondAuthAwareError(ctx, s, i, err)
	}
	fields := fieldsFromMap(raw, []fieldMapping{
		{Key: "estimated_value_micros", Label: "Marked Value", Micros: true},
	})
	return b.respondEmbed(s, i, successEmbed("Stake Revoked", fmt.Sprintf("Revoked %.2f%% of business `%d` from `%s`.", percent, businessID, username), fields))
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
