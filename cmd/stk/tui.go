package main

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	cl "stanks/internal/cli"
	"stanks/internal/game"
	"stanks/internal/syncq"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/google/uuid"
)

type state int

const (
	stateMain state = iota
	stateLogin
	stateSignup
	stateDashboard
	stateStocks
	stateFunds
	stateBusiness
	stateLeaderboard
	stateFriends
	stateSync
)

type mainModel struct {
	state         state
	apiBase       string
	client        *cl.Client
	session       *cl.Session
	dashboard     *game.Dashboard
	lastError     error
	lastSuccess   string
	width, height int

	// Main Menu
	menu list.Model

	// Forms (Login/Signup/Orders/etc)
	inputs     []textinput.Model
	focusIndex int
	subState   string

	// Navigation
	history []state

	// Data
	stocks             []game.StockView
	funds              []fundView
	leaderboard        []game.LeaderboardRow
	candidates         []employeeCandidate
	employees          []businessEmployee
	machinery          []businessMachine
	loans              []businessLoan
	syncQueue          []syncq.Command
	business           *game.BusinessView
	businesses         []game.BusinessView
	selectedBusinessID int64
	friends            []game.LeaderboardRow
}

type item struct {
	title, desc string
	s           state
}

func (i item) Title() string       { return i.title }
func (i item) Description() string { return i.desc }
func (i item) FilterValue() string { return i.title }

func initialModel(apiBase string) mainModel {
	items := []list.Item{
		item{title: "Dashboard", desc: "View your net worth, positions and businesses", s: stateDashboard},
		item{title: "Stock Market", desc: "Buy and sell stocks", s: stateStocks},
		item{title: "Mutual Funds", desc: "Trade fund baskets", s: stateFunds},
		item{title: "Business Management", desc: "Grow and manage your companies", s: stateBusiness},
		item{title: "Leaderboard", desc: "See how you rank against others", s: stateLeaderboard},
		item{title: "Social", desc: "Manage your friends", s: stateFriends},
		item{title: "Sync", desc: "Replay offline actions", s: stateSync},
		item{title: "Login", desc: "Switch account", s: stateLogin},
		item{title: "Signup", desc: "Create new account", s: stateSignup},
	}

	m := mainModel{
		state:   stateMain,
		apiBase: apiBase,
		client:  cl.NewClient(apiBase),
		menu:    list.New(items, list.NewDefaultDelegate(), 0, 0),
	}
	m.menu.Title = "Stanks TUI"

	sess, err := cl.LoadSession()
	if err == nil {
		m.session = &sess
	}

	return m
}

func (m mainModel) Init() tea.Cmd {
	return nil
}

// Msg types
type dashboardMsg *game.Dashboard
type stocksMsg []game.StockView
type fundsMsg []fundView
type leaderboardMsg []game.LeaderboardRow
type syncMsg []syncq.Command
type businessDataMsg struct {
	business           *game.BusinessView
	businesses         []game.BusinessView
	selectedBusinessID int64
}
type candidatesMsg []employeeCandidate
type employeesMsg []businessEmployee
type machineryMsg []businessMachine
type loansMsg []businessLoan
type selectBusinessMsg int64
type successMsg string
type errorMsg error

// Fetch commands
func (m mainModel) fetchDashboard() tea.Cmd {
	return func() tea.Msg {
		if m.session == nil {
			return errorMsg(fmt.Errorf("login required"))
		}
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		raw, err := m.client.Dashboard(ctx, m.session.AccessToken)
		if err != nil {
			return errorMsg(err)
		}
		d, err := decodeInto[game.Dashboard](raw)
		if err != nil {
			return errorMsg(err)
		}
		return dashboardMsg(&d)
	}
}

func (m mainModel) fetchStocks() tea.Cmd {
	return func() tea.Msg {
		if m.session == nil {
			return errorMsg(fmt.Errorf("login required"))
		}
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		raw, err := m.client.ListStocks(ctx, m.session.AccessToken, false)
		if err != nil {
			return errorMsg(err)
		}
		payload, err := decodeInto[stocksPayload](raw)
		if err != nil {
			return errorMsg(err)
		}
		return stocksMsg(payload.Stocks)
	}
}

func (m mainModel) fetchFunds() tea.Cmd {
	return func() tea.Msg {
		if m.session == nil {
			return errorMsg(fmt.Errorf("login required"))
		}
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		raw, err := m.client.ListFunds(ctx, m.session.AccessToken)
		if err != nil {
			return errorMsg(err)
		}
		payload, err := decodeInto[fundsPayload](raw)
		if err != nil {
			return errorMsg(err)
		}
		return fundsMsg(payload.Funds)
	}
}

func (m mainModel) fetchLeaderboard() tea.Cmd {
	return func() tea.Msg {
		if m.session == nil {
			return errorMsg(fmt.Errorf("login required"))
		}
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		raw, err := m.client.LeaderboardGlobal(ctx, m.session.AccessToken)
		if err != nil {
			return errorMsg(err)
		}
		payload, err := decodeInto[leaderboardPayload](raw)
		if err != nil {
			return errorMsg(err)
		}
		return leaderboardMsg(payload.Rows)
	}
}

func (m mainModel) fetchFriends() tea.Cmd {
	return func() tea.Msg {
		if m.session == nil {
			return errorMsg(fmt.Errorf("login required"))
		}
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		raw, err := m.client.LeaderboardFriends(ctx, m.session.AccessToken)
		if err != nil {
			return errorMsg(err)
		}
		payload, err := decodeInto[leaderboardPayload](raw)
		if err != nil {
			return errorMsg(err)
		}
		return leaderboardMsg(payload.Rows)
	}
}

func (m mainModel) fetchSync() tea.Cmd {
	return func() tea.Msg {
		q, err := syncq.Load()
		if err != nil {
			return errorMsg(err)
		}
		return syncMsg(q)
	}
}

func (m mainModel) fetchBusiness() tea.Cmd {
	return func() tea.Msg {
		if m.session == nil {
			return errorMsg(fmt.Errorf("login required"))
		}
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		// Get dashboard first to find active business ID
		dashRaw, err := m.client.Dashboard(ctx, m.session.AccessToken)
		if err != nil {
			return errorMsg(err)
		}
		dash, _ := decodeInto[game.Dashboard](dashRaw)
		var businessID int64
		if m.selectedBusinessID > 0 {
			for _, b := range dash.Businesses {
				if b.ID == m.selectedBusinessID {
					businessID = m.selectedBusinessID
					break
				}
			}
		}
		if businessID == 0 && dash.ActiveBusinessID != nil {
			businessID = *dash.ActiveBusinessID
		}
		if businessID == 0 && len(dash.Businesses) > 0 {
			// Backward-compatible fallback for users created before active_business_id existed.
			businessID = dash.Businesses[0].ID
		}
		if businessID == 0 {
			return businessDataMsg{business: nil, businesses: dash.Businesses, selectedBusinessID: 0}
		}

		raw, err := m.client.BusinessState(ctx, m.session.AccessToken, businessID)
		if err != nil {
			return errorMsg(err)
		}
		b, err := decodeInto[game.BusinessView](raw)
		if err != nil {
			return errorMsg(err)
		}
		return businessDataMsg{business: &b, businesses: dash.Businesses, selectedBusinessID: businessID}
	}
}

func (m mainModel) fetchCandidates() tea.Cmd {
	return func() tea.Msg {
		if m.session == nil {
			return errorMsg(fmt.Errorf("login required"))
		}
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		raw, err := m.client.ListEmployeeCandidates(ctx, m.session.AccessToken)
		if err != nil {
			return errorMsg(err)
		}
		payload, err := decodeInto[candidatesPayload](raw)
		if err != nil {
			return errorMsg(err)
		}
		return candidatesMsg(payload.Candidates)
	}
}

func (m mainModel) fetchEmployees(businessID int64) tea.Cmd {
	return func() tea.Msg {
		if m.session == nil {
			return errorMsg(fmt.Errorf("login required"))
		}
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		raw, err := m.client.ListBusinessEmployees(ctx, m.session.AccessToken, businessID)
		if err != nil {
			return errorMsg(err)
		}
		payload, err := decodeInto[businessEmployeesPayload](raw)
		if err != nil {
			return errorMsg(err)
		}
		return employeesMsg(payload.Employees)
	}
}

func (m mainModel) fetchMachinery(businessID int64) tea.Cmd {
	return func() tea.Msg {
		if m.session == nil {
			return errorMsg(fmt.Errorf("login required"))
		}
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		raw, err := m.client.ListBusinessMachinery(ctx, m.session.AccessToken, businessID)
		if err != nil {
			return errorMsg(err)
		}
		payload, err := decodeInto[machineryPayload](raw)
		if err != nil {
			return errorMsg(err)
		}
		return machineryMsg(payload.Machinery)
	}
}

func (m mainModel) fetchLoans(businessID int64) tea.Cmd {
	return func() tea.Msg {
		if m.session == nil {
			return errorMsg(fmt.Errorf("login required"))
		}
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		raw, err := m.client.ListBusinessLoans(ctx, m.session.AccessToken, businessID)
		if err != nil {
			return errorMsg(err)
		}
		payload, err := decodeInto[loansPayload](raw)
		if err != nil {
			return errorMsg(err)
		}
		return loansMsg(payload.Loans)
	}
}

func (m mainModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		case "q":
			if m.state == stateMain {
				return m, tea.Quit
			}
			m.state = stateMain
			return m, nil
		case "esc":
			if m.subState != "" {
				m.subState = ""
				m.lastError = nil
				m.lastSuccess = ""
				return m, nil
			}
			if len(m.history) > 0 {
				m.state = m.history[len(m.history)-1]
				m.history = m.history[:len(m.history)-1]
			} else {
				m.state = stateMain
			}
			m.lastError = nil
			m.lastSuccess = ""
			return m, nil
		}

	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.menu.SetSize(msg.Width-4, msg.Height-4)

	case dashboardMsg:
		m.dashboard = msg
		m.lastError = nil
		return m, nil

	case stocksMsg:
		m.stocks = msg
		m.lastError = nil
		return m, nil

	case fundsMsg:
		m.funds = msg
		m.lastError = nil
		return m, nil

	case leaderboardMsg:
		if m.state == stateFriends {
			m.friends = msg
		} else {
			m.leaderboard = msg
		}
		m.lastError = nil
		return m, nil

	case syncMsg:
		m.syncQueue = msg
		m.lastError = nil
		return m, nil

	case businessDataMsg:
		m.business = msg.business
		m.businesses = msg.businesses
		m.selectedBusinessID = msg.selectedBusinessID
		m.candidates = nil
		m.employees = nil
		m.machinery = nil
		m.loans = nil
		m.lastError = nil
		return m, nil

	case candidatesMsg:
		m.candidates = msg
		m.lastError = nil
		return m, nil

	case employeesMsg:
		m.employees = msg
		m.lastError = nil
		return m, nil

	case machineryMsg:
		m.machinery = msg
		m.lastError = nil
		return m, nil

	case loansMsg:
		m.loans = msg
		m.lastError = nil
		return m, nil

	case selectBusinessMsg:
		m.selectedBusinessID = int64(msg)
		m.lastError = nil
		return m, m.fetchBusiness()

	case successMsg:
		m.lastSuccess = string(msg)
		m.lastError = nil
		m.subState = ""
		if m.state == stateBusiness {
			return m, m.fetchBusiness()
		}
		return m, nil

	case errorMsg:
		m.lastError = msg
		return m, nil
	}

	switch m.state {
	case stateMain:
		newMenu, cmd := m.menu.Update(msg)
		m.menu = newMenu
		if keyMsg, ok := msg.(tea.KeyMsg); ok && keyMsg.String() == "enter" {
			selected := m.menu.SelectedItem().(item)
			m.history = append(m.history, m.state)
			m.state = selected.s
			return m, m.stateInit()
		}
		return m, cmd

	case stateLogin, stateSignup:
		return m.updateForm(msg)

	case stateDashboard:
		if keyMsg, ok := msg.(tea.KeyMsg); ok && keyMsg.String() == "r" {
			return m, m.fetchDashboard()
		}

	case stateStocks:
		if m.subState != "" {
			return m.updateForm(msg)
		}
		if keyMsg, ok := msg.(tea.KeyMsg); ok {
			if keyMsg.String() == "r" {
				return m, m.fetchStocks()
			}
			if keyMsg.String() == "b" || keyMsg.String() == "s" {
				m.subState = keyMsg.String()
				m.initOrderForm()
			}
		}

	case stateFunds:
		if m.subState != "" {
			return m.updateForm(msg)
		}
		if keyMsg, ok := msg.(tea.KeyMsg); ok {
			if keyMsg.String() == "r" {
				return m, m.fetchFunds()
			}
			if keyMsg.String() == "b" || keyMsg.String() == "s" {
				m.subState = "fund_" + keyMsg.String()
				m.initFundOrderForm()
			}
		}

	case stateBusiness:
		if m.subState != "" {
			return m.updateForm(msg)
		}
		if keyMsg, ok := msg.(tea.KeyMsg); ok {
			bid := m.selectedBusinessID
			if bid == 0 && m.business != nil {
				bid = m.business.ID
			}
			switch keyMsg.String() {
			case "r":
				return m, m.fetchBusiness()
			case "o":
				m.subState = "select_business"
				m.initSelectBusinessForm()
			case "c":
				m.subState = "create_business"
				m.initCreateBusinessForm()
			case "v":
				if bid == 0 {
					m.lastError = fmt.Errorf("no selected business")
					return m, nil
				}
				m.subState = "set_visibility"
				m.initVisibilityForm()
			case "i":
				if bid == 0 {
					m.lastError = fmt.Errorf("no selected business")
					return m, nil
				}
				m.subState = "business_ipo"
				m.initBusinessIPOForm()
			case "e":
				return m, m.fetchCandidates()
			case "y":
				if bid == 0 {
					m.lastError = fmt.Errorf("no selected business")
					return m, nil
				}
				return m, m.fetchEmployees(bid)
			case "h":
				if bid == 0 {
					m.lastError = fmt.Errorf("no selected business")
					return m, nil
				}
				m.subState = "employee_hire"
				m.initHireForm()
			case "t":
				if bid == 0 {
					m.lastError = fmt.Errorf("no selected business")
					return m, nil
				}
				m.subState = "employee_train"
				m.initTrainForm()
			case "m":
				if bid == 0 {
					m.lastError = fmt.Errorf("no selected business")
					return m, nil
				}
				return m, m.fetchMachinery(bid)
			case "b":
				if bid == 0 {
					m.lastError = fmt.Errorf("no selected business")
					return m, nil
				}
				m.subState = "machinery_buy"
				m.initMachineryForm()
			case "k":
				if bid == 0 {
					m.lastError = fmt.Errorf("no selected business")
					return m, nil
				}
				return m, m.fetchLoans(bid)
			case "l":
				if bid == 0 {
					m.lastError = fmt.Errorf("no selected business")
					return m, nil
				}
				m.subState = "loan_take"
				m.initLoanForm()
			case "p":
				if bid == 0 {
					m.lastError = fmt.Errorf("no selected business")
					return m, nil
				}
				m.subState = "loan_repay"
				m.initLoanForm()
			case "s":
				if bid == 0 {
					m.lastError = fmt.Errorf("no selected business")
					return m, nil
				}
				m.subState = "set_strategy"
				m.initStrategyForm()
			case "u":
				if bid == 0 {
					m.lastError = fmt.Errorf("no selected business")
					return m, nil
				}
				m.subState = "buy_upgrade"
				m.initUpgradeForm()
			case "d":
				if bid == 0 {
					m.lastError = fmt.Errorf("no selected business")
					return m, nil
				}
				m.subState = "reserve_deposit"
				m.initReserveForm("deposit")
			case "w":
				if bid == 0 {
					m.lastError = fmt.Errorf("no selected business")
					return m, nil
				}
				m.subState = "reserve_withdraw"
				m.initReserveForm("withdraw")
			case "x":
				if bid == 0 {
					m.lastError = fmt.Errorf("no selected business")
					return m, nil
				}
				m.subState = "sell_business"
				m.inputs = nil
				return m, m.submitForm()
			}
		}

	case stateLeaderboard:
		if keyMsg, ok := msg.(tea.KeyMsg); ok && keyMsg.String() == "r" {
			return m, m.fetchLeaderboard()
		}

	case stateFriends:
		if m.subState != "" {
			return m.updateForm(msg)
		}
		if keyMsg, ok := msg.(tea.KeyMsg); ok {
			if keyMsg.String() == "r" {
				return m, m.fetchFriends()
			}
			if keyMsg.String() == "a" {
				m.subState = "friend_add"
				m.initFriendForm()
			}
		}

	case stateSync:
		if keyMsg, ok := msg.(tea.KeyMsg); ok && keyMsg.String() == "s" {
			return m, m.runSync()
		}
	}

	return m, cmd
}

// Form initialization
func (m *mainModel) initLoginForm() {
	m.inputs = make([]textinput.Model, 2)
	m.inputs[0] = textinput.New()
	m.inputs[0].Placeholder = "Email"
	m.inputs[0].Focus()
	m.inputs[1] = textinput.New()
	m.inputs[1].Placeholder = "Password"
	m.inputs[1].EchoMode = textinput.EchoPassword
	m.focusIndex = 0
}

func (m *mainModel) initSignupForm() {
	m.inputs = make([]textinput.Model, 3)
	m.inputs[0] = textinput.New()
	m.inputs[0].Placeholder = "Email"
	m.inputs[0].Focus()
	m.inputs[1] = textinput.New()
	m.inputs[1].Placeholder = "Password"
	m.inputs[1].EchoMode = textinput.EchoPassword
	m.inputs[2] = textinput.New()
	m.inputs[2].Placeholder = "Username (optional)"
	m.focusIndex = 0
}

func (m *mainModel) initOrderForm() {
	m.inputs = make([]textinput.Model, 2)
	m.inputs[0] = textinput.New()
	m.inputs[0].Placeholder = "SYMBOL"
	m.inputs[0].Focus()
	m.inputs[1] = textinput.New()
	m.inputs[1].Placeholder = "Shares"
	m.focusIndex = 0
}

func (m *mainModel) initFundOrderForm() {
	m.inputs = make([]textinput.Model, 2)
	m.inputs[0] = textinput.New()
	m.inputs[0].Placeholder = "FUND CODE (e.g. TECH6X)"
	m.inputs[0].Focus()
	m.inputs[1] = textinput.New()
	m.inputs[1].Placeholder = "Shares"
	m.focusIndex = 0
}

func (m *mainModel) initCreateBusinessForm() {
	m.inputs = make([]textinput.Model, 2)
	m.inputs[0] = textinput.New()
	m.inputs[0].Placeholder = "Business Name"
	m.inputs[0].Focus()
	m.inputs[1] = textinput.New()
	m.inputs[1].Placeholder = "Visibility (private/public)"
	m.focusIndex = 0
}

func (m *mainModel) initSelectBusinessForm() {
	m.inputs = make([]textinput.Model, 1)
	m.inputs[0] = textinput.New()
	m.inputs[0].Placeholder = "Business ID"
	m.inputs[0].Focus()
	m.focusIndex = 0
}

func (m *mainModel) initVisibilityForm() {
	m.inputs = make([]textinput.Model, 1)
	m.inputs[0] = textinput.New()
	m.inputs[0].Placeholder = "Visibility (private/public)"
	m.inputs[0].Focus()
	m.focusIndex = 0
}

func (m *mainModel) initBusinessIPOForm() {
	m.inputs = make([]textinput.Model, 2)
	m.inputs[0] = textinput.New()
	m.inputs[0].Placeholder = "Stock Symbol"
	m.inputs[0].Focus()
	m.inputs[1] = textinput.New()
	m.inputs[1].Placeholder = "IPO Price (stonky)"
	m.focusIndex = 0
}

func (m *mainModel) initLoanForm() {
	m.inputs = make([]textinput.Model, 1)
	m.inputs[0] = textinput.New()
	m.inputs[0].Placeholder = "Amount (stonky)"
	m.inputs[0].Focus()
	m.focusIndex = 0
}

func (m *mainModel) initHireForm() {
	m.inputs = make([]textinput.Model, 1)
	m.inputs[0] = textinput.New()
	m.inputs[0].Placeholder = "Candidate ID"
	m.inputs[0].Focus()
	m.focusIndex = 0
}

func (m *mainModel) initTrainForm() {
	m.inputs = make([]textinput.Model, 1)
	m.inputs[0] = textinput.New()
	m.inputs[0].Placeholder = "Employee ID"
	m.inputs[0].Focus()
	m.focusIndex = 0
}

func (m *mainModel) initMachineryForm() {
	m.inputs = make([]textinput.Model, 1)
	m.inputs[0] = textinput.New()
	m.inputs[0].Placeholder = "Machine Type (assembly_line, robotics_cell, cloud_cluster, bio_reactor, quantum_rig)"
	m.inputs[0].Focus()
	m.focusIndex = 0
}

func (m *mainModel) initStrategyForm() {
	m.inputs = make([]textinput.Model, 1)
	m.inputs[0] = textinput.New()
	m.inputs[0].Placeholder = "Strategy (aggressive/balanced/defensive)"
	m.inputs[0].Focus()
	m.focusIndex = 0
}

func (m *mainModel) initUpgradeForm() {
	m.inputs = make([]textinput.Model, 1)
	m.inputs[0] = textinput.New()
	m.inputs[0].Placeholder = "Upgrade (marketing/rd/automation/compliance)"
	m.inputs[0].Focus()
	m.focusIndex = 0
}

func (m *mainModel) initReserveForm(direction string) {
	m.inputs = make([]textinput.Model, 1)
	m.inputs[0] = textinput.New()
	m.inputs[0].Placeholder = "Amount (stonky) for " + direction
	m.inputs[0].Focus()
	m.focusIndex = 0
}

func (m *mainModel) initFriendForm() {
	m.inputs = make([]textinput.Model, 1)
	m.inputs[0] = textinput.New()
	m.inputs[0].Placeholder = "Invite Code"
	m.inputs[0].Focus()
	m.focusIndex = 0
}

func (m mainModel) updateForm(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "tab", "shift+tab", "enter", "up", "down":
			s := msg.String()
			if s == "enter" && m.focusIndex == len(m.inputs)-1 {
				return m, m.submitForm()
			}
			if s == "up" || s == "shift+tab" {
				m.focusIndex--
			} else {
				m.focusIndex++
			}
			if m.focusIndex > len(m.inputs)-1 {
				m.focusIndex = 0
			} else if m.focusIndex < 0 {
				m.focusIndex = len(m.inputs) - 1
			}
			for i := 0; i < len(m.inputs); i++ {
				if i == m.focusIndex {
					cmds = append(cmds, m.inputs[i].Focus())
				} else {
					m.inputs[i].Blur()
				}
			}
			return m, tea.Batch(cmds...)
		}
	}
	for i := range m.inputs {
		var cmd tea.Cmd
		m.inputs[i], cmd = m.inputs[i].Update(msg)
		cmds = append(cmds, cmd)
	}
	return m, tea.Batch(cmds...)
}

func (m mainModel) submitForm() tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		businessID := m.selectedBusinessID
		if businessID == 0 && m.business != nil {
			businessID = m.business.ID
		}

		switch m.state {
		case stateLogin:
			sess, err := m.client.Login(ctx, m.inputs[0].Value(), m.inputs[1].Value())
			if err != nil {
				return errorMsg(err)
			}
			s := cl.Session{AccessToken: sess.AccessToken, RefreshToken: sess.RefreshToken, Email: sess.User.Email, UserID: sess.User.ID}
			cl.SaveSession(s)
			m.session = &s
			return successMsg("Logged in successfully!")

		case stateSignup:
			_, err := m.client.Signup(ctx, m.inputs[0].Value(), m.inputs[1].Value(), m.inputs[2].Value())
			if err != nil {
				return errorMsg(err)
			}
			return successMsg("Signup successful! Please verify email and login.")

		case stateStocks:
			symbol := strings.ToUpper(m.inputs[0].Value())
			shares, _ := strconv.ParseFloat(m.inputs[1].Value(), 64)
			units, _ := game.SharesToUnits(shares)
			side := "buy"
			if m.subState == "s" {
				side = "sell"
			}
			_, err := m.client.PlaceOrder(ctx, m.session.AccessToken, symbol, side, uuid.NewString(), units)
			if err != nil {
				return errorMsg(err)
			}
			return successMsg("Order executed!")

		case stateFunds:
			code := strings.ToUpper(m.inputs[0].Value())
			shares, _ := strconv.ParseFloat(m.inputs[1].Value(), 64)
			units, _ := game.SharesToUnits(shares)
			var err error
			if m.subState == "fund_b" {
				_, err = m.client.BuyFund(ctx, m.session.AccessToken, code, uuid.NewString(), units)
			} else {
				_, err = m.client.SellFund(ctx, m.session.AccessToken, code, uuid.NewString(), units)
			}
			if err != nil {
				return errorMsg(err)
			}
			return successMsg("Fund order executed!")

		case stateBusiness:
			if m.subState == "create_business" {
				_, err := m.client.CreateBusiness(ctx, m.session.AccessToken, m.inputs[0].Value(), m.inputs[1].Value(), uuid.NewString())
				if err != nil {
					return errorMsg(err)
				}
				return successMsg("Business created!")
			}
			if m.subState == "select_business" {
				id, err := strconv.ParseInt(strings.TrimSpace(m.inputs[0].Value()), 10, 64)
				if err != nil || id <= 0 {
					return errorMsg(fmt.Errorf("invalid business id"))
				}
				return selectBusinessMsg(id)
			}
			if businessID == 0 {
				return errorMsg(fmt.Errorf("no selected business"))
			}
			if m.subState == "set_visibility" {
				visibility := strings.ToLower(strings.TrimSpace(m.inputs[0].Value()))
				if visibility != "private" && visibility != "public" {
					return errorMsg(fmt.Errorf("visibility must be private or public"))
				}
				_, err := m.client.SetBusinessVisibility(ctx, m.session.AccessToken, businessID, visibility, uuid.NewString())
				if err != nil {
					return errorMsg(err)
				}
				return successMsg("Business visibility updated!")
			}
			if m.subState == "business_ipo" {
				symbol := strings.ToUpper(strings.TrimSpace(m.inputs[0].Value()))
				if err := game.ValidateSymbol(symbol); err != nil {
					return errorMsg(err)
				}
				price, err := strconv.ParseFloat(strings.TrimSpace(m.inputs[1].Value()), 64)
				if err != nil || price <= 0 {
					return errorMsg(fmt.Errorf("invalid ipo price"))
				}
				_, err = m.client.BusinessIPO(ctx, m.session.AccessToken, businessID, symbol, game.StonkyToMicros(price), uuid.NewString())
				if err != nil {
					return errorMsg(err)
				}
				return successMsg("Business IPO created!")
			}
			if m.subState == "employee_hire" {
				candidateID, err := strconv.ParseInt(strings.TrimSpace(m.inputs[0].Value()), 10, 64)
				if err != nil || candidateID <= 0 {
					return errorMsg(fmt.Errorf("invalid candidate id"))
				}
				_, err = m.client.HireEmployee(ctx, m.session.AccessToken, businessID, candidateID, uuid.NewString())
				if err != nil {
					return errorMsg(err)
				}
				return successMsg("Employee hired!")
			}
			if m.subState == "employee_train" {
				employeeID, err := strconv.ParseInt(strings.TrimSpace(m.inputs[0].Value()), 10, 64)
				if err != nil || employeeID <= 0 {
					return errorMsg(fmt.Errorf("invalid employee id"))
				}
				_, err = m.client.TrainProfessional(ctx, m.session.AccessToken, businessID, employeeID, uuid.NewString())
				if err != nil {
					return errorMsg(err)
				}
				return successMsg("Employee trained!")
			}
			if m.subState == "machinery_buy" {
				machineType := strings.ToLower(strings.TrimSpace(m.inputs[0].Value()))
				switch machineType {
				case "assembly_line", "robotics_cell", "cloud_cluster", "bio_reactor", "quantum_rig":
				default:
					return errorMsg(fmt.Errorf("invalid machine type"))
				}
				_, err := m.client.BuyBusinessMachinery(ctx, m.session.AccessToken, businessID, machineType, uuid.NewString())
				if err != nil {
					return errorMsg(err)
				}
				return successMsg("Machinery updated!")
			}
			if m.subState == "loan_take" || m.subState == "loan_repay" {
				amount, err := strconv.ParseFloat(strings.TrimSpace(m.inputs[0].Value()), 64)
				if err != nil || amount <= 0 {
					return errorMsg(fmt.Errorf("invalid amount"))
				}
				micros := game.StonkyToMicros(amount)
				if m.subState == "loan_take" {
					_, err = m.client.TakeBusinessLoan(ctx, m.session.AccessToken, businessID, micros, uuid.NewString())
				} else {
					_, err = m.client.RepayBusinessLoan(ctx, m.session.AccessToken, businessID, micros, uuid.NewString())
				}
				if err != nil {
					return errorMsg(err)
				}
				return successMsg("Loan operation successful!")
			}
			if m.subState == "set_strategy" {
				strategy := strings.ToLower(strings.TrimSpace(m.inputs[0].Value()))
				switch strategy {
				case "aggressive", "balanced", "defensive":
				default:
					return errorMsg(fmt.Errorf("invalid strategy"))
				}
				_, err := m.client.SetBusinessStrategy(ctx, m.session.AccessToken, businessID, strategy, uuid.NewString())
				if err != nil {
					return errorMsg(err)
				}
				return successMsg("Business strategy updated!")
			}
			if m.subState == "buy_upgrade" {
				upgrade := strings.ToLower(strings.TrimSpace(m.inputs[0].Value()))
				switch upgrade {
				case "marketing", "rd", "automation", "compliance":
				default:
					return errorMsg(fmt.Errorf("invalid upgrade"))
				}
				_, err := m.client.BuyBusinessUpgrade(ctx, m.session.AccessToken, businessID, upgrade, uuid.NewString())
				if err != nil {
					return errorMsg(err)
				}
				return successMsg("Business upgraded!")
			}
			if m.subState == "reserve_deposit" || m.subState == "reserve_withdraw" {
				amount, err := strconv.ParseFloat(strings.TrimSpace(m.inputs[0].Value()), 64)
				if err != nil || amount <= 0 {
					return errorMsg(fmt.Errorf("invalid amount"))
				}
				micros := game.StonkyToMicros(amount)
				if m.subState == "reserve_deposit" {
					_, err = m.client.BusinessReserveDeposit(ctx, m.session.AccessToken, businessID, micros, uuid.NewString())
				} else {
					_, err = m.client.BusinessReserveWithdraw(ctx, m.session.AccessToken, businessID, micros, uuid.NewString())
				}
				if err != nil {
					return errorMsg(err)
				}
				return successMsg("Reserve transfer complete!")
			}
			if m.subState == "sell_business" {
				_, err := m.client.SellBusinessToBank(ctx, m.session.AccessToken, businessID, uuid.NewString())
				if err != nil {
					return errorMsg(err)
				}
				return successMsg("Business sold.")
			}

		case stateFriends:
			_, err := m.client.AddFriend(ctx, m.session.AccessToken, m.inputs[0].Value(), uuid.NewString())
			if err != nil {
				return errorMsg(err)
			}
			return successMsg("Friend added!")
		}
		return nil
	}
}

func (m mainModel) runSync() tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()
		queue, _ := syncq.Load()
		if len(queue) == 0 {
			return successMsg("Queue is empty.")
		}
		for _, q := range queue {
			_, err := m.client.Do(ctx, q.Method, q.Path, m.session.AccessToken, q.Body, q.IdempotencyKey)
			if err != nil {
				return errorMsg(err)
			}
		}
		syncq.Save(nil)
		return successMsg("Sync complete!")
	}
}

// Styling
var (
	titleStyle   = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#FAFAFA")).Background(lipgloss.Color("#7D56F4")).Padding(0, 1)
	headerStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("5")).Bold(true).MarginLeft(2)
	infoStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("8")).MarginLeft(2)
	errorStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("9")).Bold(true).MarginLeft(2)
	successStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("10")).Bold(true).MarginLeft(2)
	greenStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("10"))
	redStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("9"))
	cyanStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("6"))
)

func (m mainModel) View() string {
	if m.state == stateMain {
		return lipgloss.NewStyle().Margin(1, 2).Render(m.menu.View())
	}

	header := titleStyle.Render(fmt.Sprintf(" Stanks TUI - %s ", m.stateName()))
	var content string
	if m.lastError != nil {
		content = errorStyle.Render(fmt.Sprintf("Error: %v", m.lastError))
	} else if m.lastSuccess != "" {
		content = successStyle.Render(m.lastSuccess)
	} else {
		switch m.state {
		case stateDashboard:
			content = m.dashboardView()
		case stateStocks:
			content = m.stocksView()
		case stateFunds:
			content = m.fundsView()
		case stateBusiness:
			content = m.businessView()
		case stateLeaderboard:
			content = m.leaderboardView()
		case stateFriends:
			content = m.socialView()
		case stateSync:
			content = m.syncView()
		case stateLogin, stateSignup:
			content = m.formView()
		default:
			content = infoStyle.Render("View not implemented.")
		}
	}

	footer := infoStyle.Render("\n q: main menu | esc: back | r: refresh")
	if m.state == stateStocks && m.subState == "" {
		footer += infoStyle.Render(" | b: buy | s: sell")
	}
	if m.state == stateFunds && m.subState == "" {
		footer += infoStyle.Render(" | b: buy | s: sell")
	}
	if m.state == stateBusiness && m.subState == "" {
		footer += infoStyle.Render(" | o: select | c: create | v: visibility | i: ipo | e/y: candidates/employees | h/t: hire/train")
		footer += infoStyle.Render(" | m/b: machinery list/buy | k/l/p: loans list/take/repay | s/u: strategy/upgrade | d/w: reserve +/- | x: sell")
	}
	if m.state == stateFriends && m.subState == "" {
		footer += infoStyle.Render(" | a: add friend")
	}
	if m.state == stateSync {
		footer += infoStyle.Render(" | s: sync now")
	}

	return lipgloss.JoinVertical(lipgloss.Left, "\n", header, "\n", content, footer)
}

func (m mainModel) dashboardView() string {
	if m.dashboard == nil {
		return infoStyle.Render("Loading dashboard...")
	}
	d := m.dashboard
	s := fmt.Sprintf("  Season: %d\n\n", d.SeasonID)
	s += fmt.Sprintf("  Balance:        %s stonky\n", cyanStyle.Render(formatMicros(d.BalanceMicros)))
	s += fmt.Sprintf("  Net Worth:      %s stonky\n", cyanStyle.Render(formatMicros(d.NetWorthMicros)))
	s += fmt.Sprintf("  P/L vs Start:   %s stonky\n", colorizeMicrosTUI(d.NetWorthMicros-game.StarterBalanceMicros))
	s += "\n" + headerStyle.Render("Positions") + "\n"
	if len(d.Positions) == 0 {
		s += infoStyle.Render("No positions yet.") + "\n"
	} else {
		for _, p := range d.Positions {
			s += fmt.Sprintf("  %-8s %10.4f @ %-12s P/L: %s\n", p.Symbol, game.UnitsToShares(p.QuantityUnits), formatMicros(p.CurrentPriceMicros), colorizeMicrosTUI(p.UnrealizedMicros))
		}
	}
	s += "\n" + headerStyle.Render("Businesses") + "\n"
	if len(d.Businesses) == 0 {
		s += infoStyle.Render("No businesses yet.") + "\n"
	} else {
		for _, b := range d.Businesses {
			s += fmt.Sprintf("  #%-4d %-20s Rev: %-12s Reserve: %s\n", b.ID, truncate(b.Name, 20), formatMicros(b.RevenuePerTickMicros), formatMicros(b.CashReserveMicros))
		}
	}
	return s
}

func (m mainModel) stocksView() string {
	if m.subState != "" {
		return m.formView()
	}
	if len(m.stocks) == 0 {
		return infoStyle.Render("Loading stocks...")
	}
	s := fmt.Sprintf("  %-8s %-24s %12s\n", "SYMBOL", "NAME", "PRICE")
	for _, st := range m.stocks {
		s += fmt.Sprintf("  %-8s %-24s %12s\n", st.Symbol, truncate(st.DisplayName, 24), formatMicros(st.CurrentPriceMicros))
	}
	return s
}

func (m mainModel) fundsView() string {
	if m.subState != "" {
		return m.formView()
	}
	if len(m.funds) == 0 {
		return infoStyle.Render("Loading funds...")
	}
	s := fmt.Sprintf("  %-8s %12s %-40s\n", "CODE", "NAV", "COMPONENTS")
	for _, f := range m.funds {
		s += fmt.Sprintf("  %-8s %12s %-40s\n", f.Code, formatMicros(f.NavMicros), truncate(strings.Join(f.Components, ","), 40))
	}
	return s
}

func (m mainModel) businessView() string {
	if m.subState != "" {
		return m.formView()
	}
	if m.business == nil {
		return infoStyle.Render(
			"No business selected. Use: o=select id, c=create, r=refresh.",
		)
	}
	b := m.business
	s := fmt.Sprintf("  Selected Business ID: %d\n", m.selectedBusinessID)
	if len(m.businesses) > 0 {
		s += "  Owned IDs: "
		for i, ob := range m.businesses {
			if i > 0 {
				s += ", "
			}
			s += fmt.Sprintf("%d", ob.ID)
		}
		s += "\n\n"
	} else {
		s += "\n"
	}
	s += fmt.Sprintf("  Name:       %s (#%d)\n", b.Name, b.ID)
	s += fmt.Sprintf("  Strategy:   %s\n", b.Strategy)
	s += fmt.Sprintf("  Revenue:    %s / tick\n", formatMicros(b.RevenuePerTickMicros))
	s += fmt.Sprintf("  Reserve:    %s stonky\n", formatMicros(b.CashReserveMicros))
	s += fmt.Sprintf("  Debt:       %s stonky\n", formatMicros(b.LoanOutstandingMicros))
	s += fmt.Sprintf("  Employees:  %d\n", b.EmployeeCount)
	s += fmt.Sprintf("  Upgrades:   mkt=%d rd=%d auto=%d comp=%d\n", b.MarketingLevel, b.RDLevel, b.AutomationLevel, b.ComplianceLevel)
	s += "\n" + headerStyle.Render("Actions (CLI parity)") + "\n"
	s += "  o select | c create | v visibility | i ipo | s strategy | u upgrade | x sell\n"
	s += "  e candidates | y employees | h hire | t train\n"
	s += "  m machinery list | b machinery buy\n"
	s += "  k loans list | l loan take | p loan repay\n"
	s += "  d reserve deposit | w reserve withdraw\n"
	if len(m.candidates) > 0 {
		s += "\n" + headerStyle.Render("Candidates") + "\n"
		for _, c := range m.candidates {
			s += fmt.Sprintf("  #%d %-16s %-10s cost=%s rev=%s\n", c.ID, truncate(c.FullName, 16), truncate(c.Role, 10), formatMicros(c.HireCostMicros), formatMicros(c.RevenuePerTickMicros))
		}
	}
	if len(m.employees) > 0 {
		s += "\n" + headerStyle.Render("Employees") + "\n"
		for _, e := range m.employees {
			s += fmt.Sprintf("  #%d %-16s %-10s rev=%s risk=%.2f%%\n", e.ID, truncate(e.FullName, 16), truncate(e.Role, 10), formatMicros(e.RevenuePerTickMicros), float64(e.RiskBps)/100.0)
		}
	}
	if len(m.machinery) > 0 {
		s += "\n" + headerStyle.Render("Machinery") + "\n"
		for _, mc := range m.machinery {
			s += fmt.Sprintf("  #%d %-16s lvl=%d upkeep=%s\n", mc.ID, truncate(mc.MachineType, 16), mc.Level, formatMicros(mc.UpkeepMicros))
		}
	}
	if len(m.loans) > 0 {
		s += "\n" + headerStyle.Render("Loans") + "\n"
		for _, l := range m.loans {
			s += fmt.Sprintf("  #%d out=%s rate=%.2f%% status=%s missed=%d\n", l.ID, formatMicros(l.OutstandingMicros), float64(l.InterestBps)/100.0, l.Status, l.MissedTicks)
		}
	}
	return s
}

func (m mainModel) leaderboardView() string {
	if len(m.leaderboard) == 0 {
		return infoStyle.Render("Loading leaderboard...")
	}
	s := fmt.Sprintf("  %-4s %-18s %14s\n", "RANK", "PLAYER", "NET WORTH")
	for _, r := range m.leaderboard {
		s += fmt.Sprintf("  %-4d %-18s %14s\n", r.Rank, truncate(r.Username, 18), formatMicros(r.NetWorthMicros))
	}
	return s
}

func (m mainModel) socialView() string {
	if m.subState != "" {
		return m.formView()
	}
	if len(m.friends) == 0 {
		return infoStyle.Render("No friends followed. Use 'a' to add.")
	}
	s := headerStyle.Render("Followed Friends") + "\n"
	for _, r := range m.friends {
		s += fmt.Sprintf("  %-18s NW: %14s\n", truncate(r.Username, 18), formatMicros(r.NetWorthMicros))
	}
	return s
}

func (m mainModel) syncView() string {
	if len(m.syncQueue) == 0 {
		return infoStyle.Render("Sync queue is empty.")
	}
	s := fmt.Sprintf("  %d pending operations in queue.\n\n", len(m.syncQueue))
	for _, q := range m.syncQueue {
		s += fmt.Sprintf("  - %s %s\n", q.Method, q.Path)
	}
	return s
}

func (m mainModel) formView() string {
	var s strings.Builder
	for i := range m.inputs {
		s.WriteString(m.inputs[i].View())
		if i < len(m.inputs)-1 {
			s.WriteRune('\n')
		}
	}
	s.WriteString("\n\n(esc to cancel, enter to submit)")
	return lipgloss.NewStyle().MarginLeft(4).Render(s.String())
}

func (m mainModel) stateName() string {
	names := map[state]string{stateDashboard: "Dashboard", stateStocks: "Market", stateFunds: "Funds", stateBusiness: "Business", stateLeaderboard: "Leaderboard", stateFriends: "Social", stateSync: "Sync", stateLogin: "Login", stateSignup: "Signup"}
	if name, ok := names[m.state]; ok {
		return name
	}
	return "Main Menu"
}

func (m mainModel) stateInit() tea.Cmd {
	m.subState = ""
	switch m.state {
	case stateDashboard:
		return m.fetchDashboard()
	case stateStocks:
		return m.fetchStocks()
	case stateFunds:
		return m.fetchFunds()
	case stateBusiness:
		return m.fetchBusiness()
	case stateLeaderboard:
		return m.fetchLeaderboard()
	case stateFriends:
		return m.fetchFriends()
	case stateSync:
		return m.fetchSync()
	case stateLogin:
		m.initLoginForm()
	case stateSignup:
		m.initSignupForm()
	}
	return nil
}

func colorizeMicrosTUI(v int64) string {
	text := signedMicros(v)
	if v > 0 {
		return greenStyle.Render(text)
	}
	if v < 0 {
		return redStyle.Render(text)
	}
	return text
}

func runTUI(apiBase string) error {
	p := tea.NewProgram(initialModel(apiBase), tea.WithAltScreen())
	_, err := p.Run()
	return err
}
