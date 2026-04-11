package whatsappbot

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"go.mau.fi/whatsmeow/types"
	"stanks/internal/game"
)

type businessEmployeesPayload struct {
	Employees []businessEmployee `json:"employees"`
}

type candidatesPayload struct {
	Candidates []employeeCandidate `json:"candidates"`
}

type businessEmployee struct {
	ID                   int64 `json:"id"`
	FullName             string `json:"full_name"`
	Role                 string `json:"role"`
}

type employeeCandidate struct {
	ID                   int64 `json:"id"`
	FullName             string `json:"full_name"`
	Role                 string `json:"role"`
	HireCostMicros       int64 `json:"hire_cost_micros"`
}

type stakesPayload struct {
	Stakes []game.StakeView `json:"stakes"`
}

func (b *Bot) handleBusinessCreate(ctx context.Context, chat, sender types.JID, args []string) error {
	token, _, err := b.requireSession(ctx, chat, sender)
	if err != nil {
		return nil
	}
	if len(args) < 1 {
		return b.replyText(ctx, chat, "Usage: `!business-create <name> [visibility]`")
	}
	name := args[0]
	vis := "private"
	if len(args) > 1 {
		vis = args[1]
	}

	raw, errResp := b.api.CreateBusiness(ctx, token, name, vis, "")
	if errResp != nil {
		return b.replyText(ctx, chat, "Error: "+trimAPIError(errResp))
	}
	bizID := int64(raw["id"].(float64))
	return b.replyText(ctx, chat, fmt.Sprintf("Business created successfully! ID: %d", bizID))
}

func (b *Bot) handleBusiness(ctx context.Context, chat, sender types.JID, args []string) error {
	token, _, err := b.requireSession(ctx, chat, sender)
	if err != nil {
		return nil
	}
	if len(args) < 1 {
		return b.replyText(ctx, chat, "Usage: `!business <business_id>`")
	}
	id, _ := strconv.ParseInt(args[0], 10, 64)

	raw, errResp := b.api.BusinessState(ctx, token, id)
	if errResp != nil {
		return b.replyText(ctx, chat, "Error: "+trimAPIError(errResp))
	}

	out, err := decodeInto[game.BusinessView](raw)
	if err != nil {
		return b.replyText(ctx, chat, "Parse error.")
	}

	msg := fmt.Sprintf(`*Business #%d: %s*
Strategy: %s
Employees: %d / %d
Revenue/Tick: %s
Operating Costs: %s
Reserve: %s`,
		out.ID, out.Name, out.Strategy,
		out.EmployeeCount, out.EmployeeLimit,
		fmtStonky(out.RevenuePerTickMicros),
		fmtStonky(out.OperatingCostsMicros),
		fmtStonky(out.CashReserveMicros))

	return b.replyText(ctx, chat, msg)
}

func (b *Bot) handleCandidates(ctx context.Context, chat, sender types.JID, args []string) error {
	token, _, err := b.requireSession(ctx, chat, sender)
	if err != nil {
		return nil
	}
	raw, errResp := b.api.ListEmployeeCandidates(ctx, token)
	if errResp != nil {
		return b.replyText(ctx, chat, "Error: "+trimAPIError(errResp))
	}

	out, err := decodeInto[candidatesPayload](raw)
	if err != nil || len(out.Candidates) == 0 {
		return b.replyText(ctx, chat, "No candidates available right now.")
	}

	sb := strings.Builder{}
	sb.WriteString("*Employee Candidates*\n\n")
	for i, c := range out.Candidates {
		if i >= 10 {
			break
		}
		sb.WriteString(fmt.Sprintf("%d. %s (%s) - Hire: %s\n", c.ID, c.FullName, c.Role, fmtStonky(c.HireCostMicros)))
	}
	return b.replyText(ctx, chat, strings.TrimSpace(sb.String()))
}

func (b *Bot) handleEmployees(ctx context.Context, chat, sender types.JID, args []string) error {
	token, _, err := b.requireSession(ctx, chat, sender)
	if err != nil {
		return nil
	}
	if len(args) < 1 {
		return b.replyText(ctx, chat, "Usage: `!employees <business_id>`")
	}
	id, _ := strconv.ParseInt(args[0], 10, 64)

	raw, errResp := b.api.ListBusinessEmployees(ctx, token, id)
	if errResp != nil {
		return b.replyText(ctx, chat, "Error: "+trimAPIError(errResp))
	}

	out, err := decodeInto[businessEmployeesPayload](raw)
	if err != nil || len(out.Employees) == 0 {
		return b.replyText(ctx, chat, "No employees found.")
	}

	sb := strings.Builder{}
	sb.WriteString("*Employees*\n\n")
	for _, e := range out.Employees {
		sb.WriteString(fmt.Sprintf("%d. %s (%s)\n", e.ID, e.FullName, e.Role))
	}
	return b.replyText(ctx, chat, strings.TrimSpace(sb.String()))
}

func (b *Bot) handleHireMany(ctx context.Context, chat, sender types.JID, args []string) error {
	token, _, err := b.requireSession(ctx, chat, sender)
	if err != nil {
		return nil
	}
	if len(args) < 3 {
		return b.replyText(ctx, chat, "Usage: `!hire-many <business_id> <count> <strategy>`")
	}
	id, _ := strconv.ParseInt(args[0], 10, 64)
	count, _ := strconv.Atoi(args[1])
	strat := args[2]

	_, errResp := b.api.HireEmployeesBulk(ctx, token, id, count, strat, "")
	if errResp != nil {
		return b.replyText(ctx, chat, "Error: "+trimAPIError(errResp))
	}

	return b.replyText(ctx, chat, "Successfully hired employees.")
}

func (b *Bot) handleMachinery(ctx context.Context, chat, sender types.JID, args []string) error {
	token, _, err := b.requireSession(ctx, chat, sender)
	if err != nil {
		return nil
	}
	if len(args) < 1 {
		return b.replyText(ctx, chat, "Usage: `!machinery <business_id> [buy_machine_type]`")
	}
	id, _ := strconv.ParseInt(args[0], 10, 64)

	if len(args) > 1 {
		mType := args[1]
		_, errResp := b.api.BuyBusinessMachinery(ctx, token, id, mType, "")
		if errResp != nil {
			return b.replyText(ctx, chat, "Error: "+trimAPIError(errResp))
		}
		return b.replyText(ctx, chat, "Successfully bought machinery.")
	}

	raw, errResp := b.api.ListBusinessMachinery(ctx, token, id)
	if errResp != nil {
		return b.replyText(ctx, chat, "Error: "+trimAPIError(errResp))
	}

	machines, _ := raw["machines"].([]any)
	if len(machines) == 0 {
		return b.replyText(ctx, chat, "No machinery found.")
	}

	sb := strings.Builder{}
	sb.WriteString("*Machinery*\n")
	for _, m := range machines {
		item := m.(map[string]any)
		sb.WriteString(fmt.Sprintf("- %s (Out: %s, Upkeep: %s)\n", item["machine_type"], formatMaybeMicros(item["output_micros"]), formatMaybeMicros(item["upkeep_micros"])))
	}
	return b.replyText(ctx, chat, strings.TrimSpace(sb.String()))
}

func (b *Bot) handleLoans(ctx context.Context, chat, sender types.JID, args []string) error {
	token, _, err := b.requireSession(ctx, chat, sender)
	if err != nil {
		return nil
	}
	if len(args) < 2 {
		return b.replyText(ctx, chat, "Usage: `!loans <business_id> <list|take|repay> [amount]`")
	}
	id, _ := strconv.ParseInt(args[0], 10, 64)
	action := args[1]

	var amt int64
	if len(args) > 2 {
		amt, _ = strconv.ParseInt(args[2], 10, 64)
	}

	if action == "take" {
		_, errResp := b.api.TakeBusinessLoan(ctx, token, id, amt, "")
		if errResp != nil {
			return b.replyText(ctx, chat, "Error: "+trimAPIError(errResp))
		}
		return b.replyText(ctx, chat, "Successfully taken loan.")
	} else if action == "repay" {
		_, errResp := b.api.RepayBusinessLoan(ctx, token, id, amt, "")
		if errResp != nil {
			return b.replyText(ctx, chat, "Error: "+trimAPIError(errResp))
		}
		return b.replyText(ctx, chat, "Successfully repaid loan.")
	}

	raw, errResp := b.api.ListBusinessLoans(ctx, token, id)
	if errResp != nil {
		return b.replyText(ctx, chat, "Error: "+trimAPIError(errResp))
	}
	loans, _ := raw["loans"].([]any)
	if len(loans) == 0 {
		return b.replyText(ctx, chat, "No active loans.")
	}

	sb := strings.Builder{}
	sb.WriteString("*Loans*\n")
	for _, l := range loans {
		item := l.(map[string]any)
		sb.WriteString(fmt.Sprintf("- Out: %s, Due: %s\n", formatMaybeMicros(item["outstanding_micros"]), formatMaybeMicros(item["due_amount_micros"])))
	}
	return b.replyText(ctx, chat, strings.TrimSpace(sb.String()))
}

func (b *Bot) handleStrategy(ctx context.Context, chat, sender types.JID, args []string) error {
	token, _, err := b.requireSession(ctx, chat, sender)
	if err != nil {
		return nil
	}
	if len(args) < 2 {
		return b.replyText(ctx, chat, "Usage: `!strategy <business_id> <mode>`")
	}
	id, _ := strconv.ParseInt(args[0], 10, 64)
	strat := args[1]

	_, errResp := b.api.SetBusinessStrategy(ctx, token, id, strat, "")
	if errResp != nil {
		return b.replyText(ctx, chat, "Error: "+trimAPIError(errResp))
	}
	return b.replyText(ctx, chat, "Strategy updated.")
}

func (b *Bot) handleUpgrades(ctx context.Context, chat, sender types.JID, args []string) error {
	token, _, err := b.requireSession(ctx, chat, sender)
	if err != nil {
		return nil
	}
	if len(args) < 2 {
		return b.replyText(ctx, chat, "Usage: `!upgrades <business_id> <upgrade_type>`")
	}
	id, _ := strconv.ParseInt(args[0], 10, 64)
	upg := args[1]

	_, errResp := b.api.BuyBusinessUpgrade(ctx, token, id, upg, "")
	if errResp != nil {
		return b.replyText(ctx, chat, "Error: "+trimAPIError(errResp))
	}
	return b.replyText(ctx, chat, "Upgrade bought successfully.")
}

func (b *Bot) handleReserve(ctx context.Context, chat, sender types.JID, args []string) error {
	token, _, err := b.requireSession(ctx, chat, sender)
	if err != nil {
		return nil
	}
	if len(args) < 3 {
		return b.replyText(ctx, chat, "Usage: `!reserve <business_id> <deposit|withdraw> <amount>`")
	}
	id, _ := strconv.ParseInt(args[0], 10, 64)
	action := args[1]
	amt, _ := strconv.ParseInt(args[2], 10, 64)

	var errResp error
	if action == "deposit" {
		_, errResp = b.api.BusinessReserveDeposit(ctx, token, id, amt, "")
	} else if action == "withdraw" {
		_, errResp = b.api.BusinessReserveWithdraw(ctx, token, id, amt, "")
	} else {
		return b.replyText(ctx, chat, "Action must be deposit or withdraw.")
	}

	if errResp != nil {
		return b.replyText(ctx, chat, "Error: "+trimAPIError(errResp))
	}
	return b.replyText(ctx, chat, "Reserve updated successfully.")
}

func (b *Bot) handleIPO(ctx context.Context, chat, sender types.JID, args []string) error {
	token, _, err := b.requireSession(ctx, chat, sender)
	if err != nil {
		return nil
	}
	if len(args) < 3 {
		return b.replyText(ctx, chat, "Usage: `!ipo <business_id> <symbol> <price_micros>`")
	}
	id, _ := strconv.ParseInt(args[0], 10, 64)
	symbol := args[1]
	price, _ := strconv.ParseInt(args[2], 10, 64)

	_, errResp := b.api.BusinessIPO(ctx, token, id, symbol, price, "")
	if errResp != nil {
		return b.replyText(ctx, chat, "Error: "+trimAPIError(errResp))
	}
	return b.replyText(ctx, chat, "Business IPO successfully initiated!")
}

func (b *Bot) handleSellBusiness(ctx context.Context, chat, sender types.JID, args []string) error {
	token, _, err := b.requireSession(ctx, chat, sender)
	if err != nil {
		return nil
	}
	if len(args) < 1 {
		return b.replyText(ctx, chat, "Usage: `!sell-business <business_id>`")
	}
	id, _ := strconv.ParseInt(args[0], 10, 64)

	_, errResp := b.api.SellBusinessToBank(ctx, token, id, "")
	if errResp != nil {
		return b.replyText(ctx, chat, "Error: "+trimAPIError(errResp))
	}
	return b.replyText(ctx, chat, "Business sold successfully.")
}

func (b *Bot) handleStakes(ctx context.Context, chat, sender types.JID, args []string) error {
	token, _, err := b.requireSession(ctx, chat, sender)
	if err != nil {
		return nil
	}
	raw, errResp := b.api.ListStakes(ctx, token)
	if errResp != nil {
		return b.replyText(ctx, chat, "Error: "+trimAPIError(errResp))
	}
	out, err := decodeInto[stakesPayload](raw)
	if err != nil || len(out.Stakes) == 0 {
		return b.replyText(ctx, chat, "You do not own any business stakes.")
	}

	sb := strings.Builder{}
	sb.WriteString("*Your Stakes*\n")
	for _, s := range out.Stakes {
		sb.WriteString(fmt.Sprintf("- Biz #%d: %d bps (%.2f%%)\n", s.BusinessID, s.StakeBps, float64(s.StakeBps)/100))
	}
	return b.replyText(ctx, chat, strings.TrimSpace(sb.String()))
}

func (b *Bot) handleGiveStake(ctx context.Context, chat, sender types.JID, args []string) error {
	token, _, err := b.requireSession(ctx, chat, sender)
	if err != nil {
		return nil
	}
	if len(args) < 3 {
		return b.replyText(ctx, chat, "Usage: `!give-stake <business_id> <username> <percent>`")
	}
	id, _ := strconv.ParseInt(args[0], 10, 64)
	user := args[1]
	pct, _ := strconv.ParseFloat(args[2], 64)
	bps := int32(pct * 100)

	_, errResp := b.api.TransferBusinessStake(ctx, token, id, user, bps, "")
	if errResp != nil {
		return b.replyText(ctx, chat, "Error: "+trimAPIError(errResp))
	}
	return b.replyText(ctx, chat, "Stake transferred successfully.")
}

func (b *Bot) handleRevokeStakes(ctx context.Context, chat, sender types.JID, args []string) error {
	token, _, err := b.requireSession(ctx, chat, sender)
	if err != nil {
		return nil
	}
	if len(args) < 3 {
		return b.replyText(ctx, chat, "Usage: `!revoke-stakes <business_id> <username> <percent>`")
	}
	id, _ := strconv.ParseInt(args[0], 10, 64)
	user := args[1]
	pct, _ := strconv.ParseFloat(args[2], 64)
	bps := int32(pct * 100)

	_, errResp := b.api.RevokeBusinessStake(ctx, token, id, user, bps, "")
	if errResp != nil {
		return b.replyText(ctx, chat, "Error: "+trimAPIError(errResp))
	}
	return b.replyText(ctx, chat, "Stake revoked successfully.")
}
