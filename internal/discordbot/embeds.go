package discordbot

import (
	"fmt"
	"math"
	"strings"
	"time"

	"stanks/internal/game"

	"github.com/bwmarrin/discordgo"
)

const (
	colorSuccess  = 0x2ECC71
	colorError    = 0xE74C3C
	colorInfo     = 0x3498DB
	colorWarning  = 0xF39C12
	colorBusiness = 0x9B59B6
	colorMarket   = 0xE67E22
	colorGold     = 0xF1C40F
	colorDark     = 0x2C3E50
)

type EmbedBuilder struct {
	embed *discordgo.MessageEmbed
}

func NewEmbed() *EmbedBuilder {
	return &EmbedBuilder{embed: &discordgo.MessageEmbed{
		Footer:    &discordgo.MessageEmbedFooter{Text: "Stanks"},
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}}
}

func (e *EmbedBuilder) Title(t string) *EmbedBuilder { e.embed.Title = t; return e }
func (e *EmbedBuilder) Desc(d string) *EmbedBuilder  { e.embed.Description = d; return e }
func (e *EmbedBuilder) Color(c int) *EmbedBuilder     { e.embed.Color = c; return e }
func (e *EmbedBuilder) Footer(f string) *EmbedBuilder { e.embed.Footer.Text = f; return e }
func (e *EmbedBuilder) Thumbnail(url string) *EmbedBuilder {
	e.embed.Thumbnail = &discordgo.MessageEmbedThumbnail{URL: url}
	return e
}
func (e *EmbedBuilder) Field(name, value string, inline bool) *EmbedBuilder {
	e.embed.Fields = append(e.embed.Fields, &discordgo.MessageEmbedField{Name: name, Value: value, Inline: inline})
	return e
}
func (e *EmbedBuilder) BlankField() *EmbedBuilder {
	return e.Field("\u200b", "\u200b", false)
}
func (e *EmbedBuilder) Build() *discordgo.MessageEmbed { return e.embed }

func successEmbed(title, description string, fields []*discordgo.MessageEmbedField) *discordgo.MessageEmbed {
	b := NewEmbed().Title(title).Desc(description).Color(colorSuccess)
	for _, f := range fields {
		b.Field(f.Name, f.Value, f.Inline)
	}
	return b.Build()
}

func errorEmbed(message string) *discordgo.MessageEmbed {
	return NewEmbed().Title("Request Failed").Desc(message).Color(colorError).Build()
}

func infoEmbed(title, description string, fields []*discordgo.MessageEmbedField) *discordgo.MessageEmbed {
	b := NewEmbed().Title(title).Desc(description).Color(colorInfo)
	for _, f := range fields {
		b.Field(f.Name, f.Value, f.Inline)
	}
	return b.Build()
}

func warningEmbed(title, description string) *discordgo.MessageEmbed {
	return NewEmbed().Title(title).Desc(description).Color(colorWarning).Build()
}

func marketEmbed(title, description string, fields []*discordgo.MessageEmbedField) *discordgo.MessageEmbed {
	b := NewEmbed().Title(title).Desc(description).Color(colorMarket)
	for _, f := range fields {
		b.Field(f.Name, f.Value, f.Inline)
	}
	return b.Build()
}

func businessEmbed(title, description string, fields []*discordgo.MessageEmbedField) *discordgo.MessageEmbed {
	b := NewEmbed().Title(title).Desc(description).Color(colorBusiness)
	for _, f := range fields {
		b.Field(f.Name, f.Value, f.Inline)
	}
	return b.Build()
}

func leaderboardEmbed(title, description string, fields []*discordgo.MessageEmbedField) *discordgo.MessageEmbed {
	b := NewEmbed().Title(title).Desc(description).Color(colorGold)
	for _, f := range fields {
		b.Field(f.Name, f.Value, f.Inline)
	}
	return b.Build()
}

func fmtStonky(micros int64) string {
	v := game.MicrosToStonky(micros)
	if v >= 1_000_000 {
		return fmt.Sprintf("%.2fM stonky", v/1_000_000)
	}
	if v >= 1_000 {
		return fmt.Sprintf("%.2fK stonky", v/1_000)
	}
	return fmt.Sprintf("%.2f stonky", v)
}

func fmtShares(units int64) string {
	return fmt.Sprintf("%.4f", game.UnitsToShares(units))
}

func fmtPL(micros int64) string {
	v := game.MicrosToStonky(micros)
	if micros > 0 {
		return fmt.Sprintf("+%.2f stonky", v)
	}
	if micros < 0 {
		return fmt.Sprintf("%.2f stonky", v)
	}
	return "0.00 stonky"
}

func fmtPercent(bps int32) string {
	return fmt.Sprintf("%.1f%%", float64(bps)/100)
}

var sparkChars = []rune{'▁', '▂', '▃', '▄', '▅', '▆', '▇', '█'}

func sparkline(prices []int64) string {
	if len(prices) == 0 {
		return "-"
	}
	if len(prices) == 1 {
		return string(sparkChars[3])
	}
	mn, mx := prices[0], prices[0]
	for _, p := range prices[1:] {
		if p < mn {
			mn = p
		}
		if p > mx {
			mx = p
		}
	}
	span := float64(mx - mn)
	if span == 0 {
		span = 1
	}
	var sb strings.Builder
	for _, p := range prices {
		idx := int(math.Round(float64(p-mn) / span * float64(len(sparkChars)-1)))
		if idx < 0 {
			idx = 0
		}
		if idx >= len(sparkChars) {
			idx = len(sparkChars) - 1
		}
		sb.WriteRune(sparkChars[idx])
	}
	return sb.String()
}

func progressBar(current, max int32, width int) string {
	if max <= 0 {
		max = 1
	}
	ratio := float64(current) / float64(max)
	if ratio > 1.0 {
		ratio = 1.0
	}
	if ratio < 0 {
		ratio = 0
	}
	filled := int(math.Round(ratio * float64(width)))
	empty := width - filled
	return strings.Repeat("█", filled) + strings.Repeat("░", empty) + fmt.Sprintf(" %.0f%%", ratio*100)
}

func upgradeBar(level int32, maxLevel int32) string {
	if maxLevel <= 0 {
		maxLevel = 10
	}
	return progressBar(level, maxLevel, 10)
}

type fieldMapping struct {
	Key    string
	Label  string
	Micros bool
}

func fieldsFromMap(raw map[string]any, mappings []fieldMapping) []*discordgo.MessageEmbedField {
	fields := make([]*discordgo.MessageEmbedField, 0, len(mappings))
	for _, m := range mappings {
		value, ok := raw[m.Key]
		if !ok {
			continue
		}
		text := fmt.Sprint(value)
		if m.Micros {
			if micros, ok := toInt64(value); ok {
				text = fmtStonky(micros)
			}
		}
		fields = append(fields, &discordgo.MessageEmbedField{Name: m.Label, Value: text, Inline: true})
	}
	return fields
}
